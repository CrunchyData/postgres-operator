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

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgpgadmins",verbs={list}

// findPGAdminsForPostgresCluster returns PGAdmins that target a given cluster.
func (r *PGAdminReconciler) findPGAdminsForPostgresCluster(
	ctx context.Context, cluster client.Object,
) []*v1beta1.PGAdmin {
	var (
		matching []*v1beta1.PGAdmin
		pgadmins v1beta1.PGAdminList
	)

	// NOTE: If this becomes slow due to a large number of pgadmins in a single
	// namespace, we can configure the [ctrl.Manager] field indexer and pass a
	// [fields.Selector] here.
	// - https://book.kubebuilder.io/reference/watching-resources/externally-managed.html
	if r.List(ctx, &pgadmins, &client.ListOptions{
		Namespace: cluster.GetNamespace(),
	}) == nil {
		for i := range pgadmins.Items {
			for _, serverGroup := range pgadmins.Items[i].Spec.ServerGroups {
				if selector, err := naming.AsSelector(serverGroup.PostgresClusterSelector); err == nil {
					if selector.Matches(labels.Set(cluster.GetLabels())) {
						matching = append(matching, &pgadmins.Items[i])
					}
				}
			}
		}
	}
	return matching
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="postgresclusters",verbs={list,watch}

// getClustersForPGAdmin returns clusters managed by the given pgAdmin
func (r *PGAdminReconciler) getClustersForPGAdmin(
	ctx context.Context,
	pgAdmin *v1beta1.PGAdmin,
) (map[string]*v1beta1.PostgresClusterList, error) {
	matching := make(map[string]*v1beta1.PostgresClusterList)
	var err error
	var selector labels.Selector

	for _, serverGroup := range pgAdmin.Spec.ServerGroups {
		if selector, err = naming.AsSelector(serverGroup.PostgresClusterSelector); err == nil {
			var filteredList v1beta1.PostgresClusterList
			err = r.List(ctx, &filteredList,
				client.InNamespace(pgAdmin.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			)
			if err == nil {
				matching[serverGroup.Name] = &filteredList
			}
		}
	}

	return matching, err
}
