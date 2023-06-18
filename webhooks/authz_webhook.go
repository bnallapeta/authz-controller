package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/stakater-ab/tenant-operator/api/v1beta2"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type AuthzWebhook struct {
	Client  client.Client
	decoder *admission.Decoder
	Log     logr.Logger
}

func (a *AuthzWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := a.Log.WithValues("webhook", req.UID)

	var obj v1beta2.Tenant
	if err := a.decoder.Decode(req, &obj); err != nil {
		log.Error(err, "Unable to decode object")
		return admission.Errored(http.StatusBadRequest, err)
	}

	// extract the user's roles and groups from the request
	userName := req.UserInfo.Username
	userRoles, err := fetchUserRolesFromKeycloak(userName, "http://keycloak-server.com", "access-token")
	if err != nil {
		log.Error(err, "Unable to fetch user roles from Keycloak")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	resource := "tenant"
	verb := string(req.Operation)

	for _, role := range userRoles {
		allowed, err := isOperationAllowed(role, resource, verb)
		if err != nil {
			log.Error(err, "Unable to determine if operation is allowed")
			return admission.Errored(http.StatusInternalServerError, err)
		}
		if allowed {
			log.Info("Allowing operations")
			return admission.Allowed("")
		}
	}

	log.Info("Denying operations")
	return admission.Denied("User does not have permission to perform the operation")
}

func (a *AuthzWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

func (a *AuthzWebhook) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1beta2.Tenant{}).Complete()
}

// func (a *AuthzWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
// 	// TODO: Add your own custom validation logic for create operations here
// 	return nil, nil
// }

// func (a *AuthzWebhook) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
// 	// TODO: Add your own custom validation logic for update operations here
// 	return nil, nil
// }

// func (a *AuthzWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
// 	// TODO: Add your own custom validation logic for delete operations here
// 	return nil, nil
// }

func fetchUserRolesFromKeycloak(username string, keycloakServer string, accessToken string) ([]string, error) {
	req, err := http.NewRequest("GET", keycloakServer+"/auth/admin/realms/poc-realm/users/"+username+"/role-mappings", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var roleMappings map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&roleMappings)
	if err != nil {
		return nil, err
	}
	roles := make([]string, 0)
	for _, v := range roleMappings {
		roles = append(roles, v.(string))
	}

	return roles, nil
}

func isOperationAllowed(username string, resource string, verb string) (bool, error) {
	config, err := clientcmd.BuildConfigFromFlags("", "~/.kube/config")
	if err != nil {
		return false, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return false, err
	}

	sar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: "default",
				Verb:      verb,
				Group:     "tenantoperator.stakater.com",
				Resource:  resource,
			},
		},
	}

	res, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return res.Status.Allowed, nil
}
