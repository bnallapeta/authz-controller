package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/stakater-ab/tenant-operator/api/v1beta2"
	authv1 "k8s.api/authorization/v1"
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

	// Core authorization logic
	// Extract user from request attributes
	user := req.UserInfo.Username
	userGroups := req.UserInfo.Groups

	// Authn process with Keycloak
	// Fetch user roles from Keycloak
	keycloakserver := ""
	accessToken := ""
	userRoles, err := fetchUserRolesFromKeycloak(user, keycloakserver, accessToken)
	if err != nil {
		log.Error(err, "Unable to fetch roles from Keycloak")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Check if user roles are allowed the requested operation
	allowed, err := isOperationAllowed(userRoles, userGroups, &obj)
	if err != nil {
		log.Error(err, "Could not evaluate user's permissions")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// If not allowed, deny the request
	if !allowed {
		log.Info("Denying request operation")
		return admission.Denied("User is not authorized to perform this operation")
	}

	// If allowed, permit the request
	log.Info("Allowing operations")
	return admission.Allowed("")
}

func (a *AuthzWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

func (a *AuthzWebhook) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1beta2.Tenant{}).WithValidator(a).Complete()
}

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
