// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgupgrade

import (
	"context"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// The client used by the controller sets up a cache and an informer for any GVK
// that it GETs. That informer needs the "watch" permission.
// - https://github.com/kubernetes-sigs/controller-runtime/issues/1249
// - https://github.com/kubernetes-sigs/controller-runtime/issues/1454
//+kubebuilder:rbac:groups="postgres-operator.crunchydata.com",resources="postgresclusters",verbs={get,watch}
//+kubebuilder:rbac:groups="",resources="endpoints",verbs={list,watch}
//+kubebuilder:rbac:groups="batch",resources="jobs",verbs={list,watch}
//+kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={list,watch}

func (r *PGUpgradeReconciler) observeWorld(
	ctx context.Context, upgrade *v1beta1.PGUpgrade,
) (*World, error) {
	selectCluster := labels.SelectorFromSet(labels.Set{
		LabelCluster: upgrade.Spec.PostgresClusterName,
	})

	world := NewWorld()
	world.Upgrade = upgrade

	cluster := v1beta1.NewPostgresCluster()
	err := errors.WithStack(
		r.Client.Get(ctx, client.ObjectKey{
			Namespace: upgrade.Namespace,
			Name:      upgrade.Spec.PostgresClusterName,
		}, cluster))
	err = world.populateCluster(cluster, err)

	if err == nil {
		var endpoints corev1.EndpointsList
		err = errors.WithStack(
			r.Client.List(ctx, &endpoints,
				client.InNamespace(upgrade.Namespace),
				client.MatchingLabelsSelector{Selector: selectCluster},
			))
		world.populatePatroniEndpoints(endpoints.Items)
	}

	if err == nil {
		var jobs batchv1.JobList
		err = errors.WithStack(
			r.Client.List(ctx, &jobs,
				client.InNamespace(upgrade.Namespace),
				client.MatchingLabelsSelector{Selector: selectCluster},
			))
		for i := range jobs.Items {
			world.Jobs[jobs.Items[i].Name] = &jobs.Items[i]
		}
	}

	if err == nil {
		var statefulsets appsv1.StatefulSetList
		err = errors.WithStack(
			r.Client.List(ctx, &statefulsets,
				client.InNamespace(upgrade.Namespace),
				client.MatchingLabelsSelector{Selector: selectCluster},
			))
		world.populateStatefulSets(statefulsets.Items)
	}

	if err == nil {
		world.populateShutdown()
	}

	return world, err
}

func (w *World) populateCluster(cluster *v1beta1.PostgresCluster, err error) error {
	if err == nil {
		w.Cluster = cluster
		w.ClusterNotFound = nil

	} else if apierrors.IsNotFound(err) {
		w.Cluster = nil
		w.ClusterNotFound = err
		err = nil
	}
	return err
}

func (w *World) populatePatroniEndpoints(endpoints []corev1.Endpoints) {
	for index, endpoint := range endpoints {
		if endpoint.Labels[LabelPatroni] != "" {
			w.PatroniEndpoints = append(w.PatroniEndpoints, &endpoints[index])
		}
	}
}

// populateStatefulSets assigns
// a) the expected number of replicas -- the number of StatefulSets that have the expected
// LabelInstance label, minus 1 (for the primary)
// b) the primary StatefulSet and replica StatefulSets if the cluster is shutdown.
// When the cluster is not shutdown, we cannot verify which StatefulSet is the primary.
func (w *World) populateStatefulSets(statefulSets []appsv1.StatefulSet) {
	w.ReplicasExpected = -1
	if w.Cluster != nil {
		startup := w.Cluster.Status.StartupInstance
		for index, sts := range statefulSets {
			if sts.Labels[LabelInstance] != "" {
				w.ReplicasExpected++
				if startup != "" {
					switch sts.Name {
					case startup:
						w.ClusterPrimary = &statefulSets[index]
					default:
						w.ClusterReplicas = append(w.ClusterReplicas, &statefulSets[index])
					}
				}
			}
		}
	}
}

func (w *World) populateShutdown() {
	if w.Cluster != nil {
		status := w.Cluster.Status
		generation := status.ObservedGeneration

		// The cluster is "shutdown" only when it is specified *and* the status
		// indicates all instances are stopped.
		shutdownValue := w.Cluster.Spec.Shutdown
		if shutdownValue != nil {
			w.ClusterShutdown = *shutdownValue
		} else {
			w.ClusterShutdown = false
		}
		w.ClusterShutdown = w.ClusterShutdown && generation == w.Cluster.GetGeneration()

		sets := status.InstanceSets
		for _, set := range sets {
			if n := set.Replicas; n != 0 {
				w.ClusterShutdown = false
			}
		}
	}
}

type World struct {
	Cluster *v1beta1.PostgresCluster
	Upgrade *v1beta1.PGUpgrade

	ClusterNotFound  error
	ClusterPrimary   *appsv1.StatefulSet
	ClusterReplicas  []*appsv1.StatefulSet
	ClusterShutdown  bool
	ReplicasExpected int

	PatroniEndpoints []*corev1.Endpoints
	Jobs             map[string]*batchv1.Job
}

func NewWorld() *World {
	return &World{
		Jobs: make(map[string]*batchv1.Job),
	}
}
