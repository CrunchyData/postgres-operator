// Copyright 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package standalone_pgadmin

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// PGAdminReconciler reconciles a PGAdmin object
type PGAdminReconciler struct {
	client.Client
	Owner  client.FieldOwner
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=postgres-operator.crunchydata.com,resources=pgadmins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=postgres-operator.crunchydata.com,resources=pgadmins/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=postgres-operator.crunchydata.com,resources=pgadmins/finalizers,verbs=update

// Reconcile which aims to move the current state of the pgAdmin closer to the
// desired state described in a [v1beta1.PGAdmin] identified by request.
func (r *PGAdminReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := logging.FromContext(ctx)

	pgAdmin := &v1beta1.PGAdmin{}
	if err := r.Get(ctx, req.NamespacedName, pgAdmin); err != nil {
		if err = client.IgnoreNotFound(err); err != nil {
			log.Error(err, "unable to fetch PGAdmin")
		}
		return ctrl.Result{}, err
	}
	log.Info("Reconciling pgAdmin")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PGAdminReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.PGAdmin{}).
		Complete(r)
}
