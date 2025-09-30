// Copyright 2023 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
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
	// namespace, we can configure the [manager.Manager] field indexer and pass a
	// [fields.Selector] here.
	// - https://book.kubebuilder.io/reference/watching-resources/externally-managed.html
	if r.Reader.List(ctx, &pgadmins, &client.ListOptions{
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

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="pgadmins",verbs={list}

// findPGAdminsForSecret returns PGAdmins that have a user or users that have their password
// stored in the Secret
func (r *PGAdminReconciler) findPGAdminsForSecret(
	ctx context.Context, secret client.ObjectKey,
) []*v1beta1.PGAdmin {
	var matching []*v1beta1.PGAdmin
	var pgadmins v1beta1.PGAdminList

	// NOTE: If this becomes slow due to a large number of PGAdmins in a single
	// namespace, we can configure the [manager.Manager] field indexer and pass a
	// [fields.Selector] here.
	// - https://book.kubebuilder.io/reference/watching-resources/externally-managed.html
	if err := r.Reader.List(ctx, &pgadmins, &client.ListOptions{
		Namespace: secret.Namespace,
	}); err == nil {
		for i := range pgadmins.Items {
			for j := range pgadmins.Items[i].Spec.Users {
				if pgadmins.Items[i].Spec.Users[j].PasswordRef.Name == secret.Name {
					matching = append(matching, &pgadmins.Items[i])
					break
				}
			}
		}
	}
	return matching
}

//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="postgresclusters",verbs={get,list}

// getClustersForPGAdmin returns clusters managed by the given pgAdmin
func (r *PGAdminReconciler) getClustersForPGAdmin(
	ctx context.Context,
	pgAdmin *v1beta1.PGAdmin,
) (map[string][]*v1beta1.PostgresCluster, error) {
	matching := make(map[string][]*v1beta1.PostgresCluster)
	var err error
	var selector labels.Selector

	for _, serverGroup := range pgAdmin.Spec.ServerGroups {
		var cluster v1beta1.PostgresCluster
		if serverGroup.PostgresClusterName != "" {
			err = r.Reader.Get(ctx, client.ObjectKey{
				Name:      serverGroup.PostgresClusterName,
				Namespace: pgAdmin.GetNamespace(),
			}, &cluster)
			if err == nil {
				matching[serverGroup.Name] = []*v1beta1.PostgresCluster{&cluster}
			}
			continue
		}
		if selector, err = naming.AsSelector(serverGroup.PostgresClusterSelector); err == nil {
			var list v1beta1.PostgresClusterList
			err = r.Reader.List(ctx, &list,
				client.InNamespace(pgAdmin.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			)
			if err == nil {
				matching[serverGroup.Name] = initialize.Pointers(list.Items...)
			}
		}
	}

	return matching, err
}
