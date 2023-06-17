package tenant

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/stakater-ab/tenant-operator/api/v1beta2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TenantReconciler struct {
	client.Client
	Log logr.Logger
}

func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("tenant", req.NamespacedName)

	var tenant v1beta2.Tenant
	if err := r.Get(ctx, req.NamespacedName, &tenant); err != nil {
		log.Error(err, "Unable to fetch Tenant")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// implement logic here

	return ctrl.Result{}, nil
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1beta2.Tenant{}).Complete(r)
}
