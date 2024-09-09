// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgadmins",verbs={list}

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
				if serverGroup.PostgresClusterName == cluster.GetName() {
					matching = append(matching, &pgadmins.Items[i])
					continue
				}
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
		cluster := &v1beta1.PostgresCluster{}
		if serverGroup.PostgresClusterName != "" {
			err = r.Get(ctx, types.NamespacedName{
				Name:      serverGroup.PostgresClusterName,
				Namespace: pgAdmin.GetNamespace(),
			}, cluster)
			if err == nil {
				matching[serverGroup.Name] = &v1beta1.PostgresClusterList{
					Items: []v1beta1.PostgresCluster{*cluster},
				}
			}
			continue
		}
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
