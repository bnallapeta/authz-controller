package webhooks

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	authv1 "k8s.io/api/authorization/v1"
)

func TestFetchUserRolesFromKeycloak(t *testing.T) {
	// Mock Keycloak server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should be replaced with actual response data
		w.Write([]byte(`{"client":{"realm-management":["view-users","manage-authorization","manage-clients","query-groups","query-users","view-authorization","view-clients","view-events","view-identity-providers","view-realm","view-users"],"account":["manage-account","manage-account-links","view-profile"]},"realm":["offline_access","uma_authorization"],"clientRoles":{"account":["manage-account","manage-account-links","view-profile"]},"compositeRoles":{"client":{"realm-management":["view-users"]}}}`))
	}))
	defer mockServer.Close()

	// Modify the fetchUserRolesFromKeycloak function to accept the server URL as a parameter for testing
	roles, err := fetchUserRolesFromKeycloak("test-user", mockServer.URL, "test-token")
	assert.NoError(t, err)
	assert.Contains(t, roles, "tenant-viewer")
}

func TestIsOperationAllowed(t *testing.T) {
	// Mock SelfSubjectAccessReview
	sar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: "default",
				Verb:      "get",
				Group:     "tenantoperator.stakater.com",
				Resource:  "tenants",
			},
		},
		Status: authv1.SubjectAccessReviewStatus{
			Allowed: true,
			Denied:  false,
		},
	}
	// Assuming isOperationAllowed now takes a sar as input
	allowed, err := isOperationAllowed("test-user", "tenants", "get")
	assert.NoError(t, err)
	assert.True(t, allowed)

	sar.Status.Allowed = false
	sar.Status.Denied = true
	allowed, err = isOperationAllowed("test-user", "tenants", "get")
	assert.NoError(t, err)
	assert.False(t, allowed)
}
