// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package patroni

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// ClusterBootstrapped returns a bool indicating whether or not Patroni has successfully
// bootstrapped the PostgresCluster
func ClusterBootstrapped(postgresCluster *v1beta1.PostgresCluster) bool {
	return postgresCluster.Status.Patroni.SystemIdentifier != ""
}

// ClusterConfigMap populates the shared ConfigMap with fields needed to run Patroni.
func ClusterConfigMap(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	inHBAs postgres.HBAs,
	inParameters postgres.Parameters,
	outClusterConfigMap *corev1.ConfigMap,
) error {
	var err error

	initialize.StringMap(&outClusterConfigMap.Data)

	outClusterConfigMap.Data[configMapFileKey], err = clusterYAML(inCluster, inHBAs,
		inParameters)

	return err
}

// InstanceConfigMap populates the shared ConfigMap with fields needed to run Patroni.
func InstanceConfigMap(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	inInstanceSpec *v1beta1.PostgresInstanceSetSpec,
	outInstanceConfigMap *corev1.ConfigMap,
) error {
	var err error

	initialize.StringMap(&outInstanceConfigMap.Data)

	command := pgbackrest.ReplicaCreateCommand(inCluster, inInstanceSpec)

	outInstanceConfigMap.Data[configMapFileKey], err = instanceYAML(
		inCluster, inInstanceSpec, command)

	return err
}

// InstanceCertificates populates the shared Secret with certificates needed to run Patroni.
func InstanceCertificates(ctx context.Context,
	inRoot pki.Certificate, inDNS pki.Certificate,
	inDNSKey pki.PrivateKey, outInstanceCertificates *corev1.Secret,
) error {
	initialize.ByteMap(&outInstanceCertificates.Data)

	var err error
	outInstanceCertificates.Data[certAuthorityFileKey], err = certFile(inRoot)

	if err == nil {
		outInstanceCertificates.Data[certServerFileKey], err = certFile(inDNSKey, inDNS)
	}

	return err
}

// InstancePod populates a PodTemplateSpec with the fields needed to run Patroni.
// The database container must already be in the template.
func InstancePod(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	inClusterConfigMap *corev1.ConfigMap,
	inClusterPodService *corev1.Service,
	inPatroniLeaderService *corev1.Service,
	inInstanceSpec *v1beta1.PostgresInstanceSetSpec,
	inInstanceCertificates *corev1.Secret,
	inInstanceConfigMap *corev1.ConfigMap,
	outInstancePod *corev1.PodTemplateSpec,
) error {
	initialize.Labels(outInstancePod)

	// When using Kubernetes for DCS, Patroni discovers members by listing Pods
	// that have the "scope" label. See the "kubernetes.scope_label" and
	// "kubernetes.labels" settings.
	outInstancePod.Labels[naming.LabelPatroni] = naming.PatroniScope(inCluster)

	var container *corev1.Container
	for i := range outInstancePod.Spec.Containers {
		if outInstancePod.Spec.Containers[i].Name == naming.ContainerDatabase {
			container = &outInstancePod.Spec.Containers[i]
		}
	}

	container.Command = []string{"patroni", configDirectory}

	container.Env = append(container.Env,
		instanceEnvironment(inCluster, inClusterPodService, inPatroniLeaderService,
			outInstancePod.Spec.Containers)...)

	volume := corev1.Volume{Name: "patroni-config"}
	volume.Projected = new(corev1.ProjectedVolumeSource)

	// Add our projections after those specified in the CR. Items later in the
	// list take precedence over earlier items (that is, last write wins).
	// - https://kubernetes.io/docs/concepts/storage/volumes/#projected
	volume.Projected.Sources = append(append(volume.Projected.Sources,
		instanceConfigFiles(inClusterConfigMap, inInstanceConfigMap)...),
		instanceCertificates(inInstanceCertificates)...)

	outInstancePod.Spec.Volumes = append(outInstancePod.Spec.Volumes, volume)

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: configDirectory,
		ReadOnly:  true,
	})

	instanceProbes(inCluster, container)

	return nil
}

// instanceProbes adds Patroni liveness and readiness probes to container.
func instanceProbes(cluster *v1beta1.PostgresCluster, container *corev1.Container) {

	// Patroni uses a watchdog to ensure that PostgreSQL does not accept commits
	// after the leader lock expires, even if Patroni becomes unresponsive.
	// - https://github.com/zalando/patroni/blob/v2.0.1/docs/watchdog.rst
	//
	// Similar functionality is provided by a liveness probe. When the probe
	// finally fails, kubelet will send a SIGTERM to the Patroni process.
	// If the process does not stop, kubelet will send a SIGKILL after the pod's
	// TerminationGracePeriodSeconds.
	// - https://docs.k8s.io/concepts/workloads/pods/pod-lifecycle/
	//
	// TODO(cbandy): Consider TerminationGracePeriodSeconds' impact here.
	// TODO(cbandy): Consider if a PreStop hook is necessary.
	container.LivenessProbe = probeTiming(cluster.Spec.Patroni)
	container.LivenessProbe.InitialDelaySeconds = 3
	container.LivenessProbe.HTTPGet = &corev1.HTTPGetAction{
		Path:   "/liveness",
		Port:   intstr.FromInt(int(*cluster.Spec.Patroni.Port)),
		Scheme: corev1.URISchemeHTTPS,
	}

	// Readiness is reflected in the controlling object's status (e.g. ReadyReplicas)
	// and allows our controller to react when Patroni bootstrap completes.
	//
	// When using Endpoints for DCS, this probe does not affect the availability
	// of the leader Pod in the leader Service.
	container.ReadinessProbe = probeTiming(cluster.Spec.Patroni)
	container.ReadinessProbe.InitialDelaySeconds = 3
	container.ReadinessProbe.HTTPGet = &corev1.HTTPGetAction{
		Path:   "/readiness",
		Port:   intstr.FromInt(int(*cluster.Spec.Patroni.Port)),
		Scheme: corev1.URISchemeHTTPS,
	}
}

// PodIsPrimary returns whether or not pod is currently acting as the leader with
// the "master" role. This role will be called "primary" in the future, see:
// - https://github.com/zalando/patroni/blob/master/docs/releases.rst?plain=1#L213
func PodIsPrimary(pod metav1.Object) bool {
	if pod == nil {
		return false
	}

	// TODO(cbandy): This works only when using Kubernetes for DCS.

	// - https://github.com/zalando/patroni/blob/v3.1.1/patroni/ha.py#L296
	// - https://github.com/zalando/patroni/blob/v3.1.1/patroni/ha.py#L583
	// - https://github.com/zalando/patroni/blob/v3.1.1/patroni/ha.py#L782
	// - https://github.com/zalando/patroni/blob/v3.1.1/patroni/ha.py#L1574
	status := pod.GetAnnotations()["status"]
	return strings.Contains(status, `"role":"master"`)
}

// PodIsStandbyLeader returns whether or not pod is currently acting as a "standby_leader".
func PodIsStandbyLeader(pod metav1.Object) bool {
	if pod == nil {
		return false
	}

	// TODO(cbandy): This works only when using Kubernetes for DCS.

	// - https://github.com/zalando/patroni/blob/v2.0.2/patroni/ha.py#L190
	// - https://github.com/zalando/patroni/blob/v2.0.2/patroni/ha.py#L294
	// - https://github.com/zalando/patroni/blob/v2.0.2/patroni/ha.py#L353
	status := pod.GetAnnotations()["status"]
	return strings.Contains(status, `"role":"standby_leader"`)
}

// PodRequiresRestart returns whether or not PostgreSQL inside pod has (pending)
// parameter changes that require a PostgreSQL restart.
func PodRequiresRestart(pod metav1.Object) bool {
	if pod == nil {
		return false
	}

	// TODO(cbandy): This works only when using Kubernetes for DCS.

	// - https://github.com/zalando/patroni/blob/v2.1.1/patroni/ha.py#L198
	// - https://github.com/zalando/patroni/blob/v2.1.1/patroni/postgresql/config.py#L977
	// - https://github.com/zalando/patroni/blob/v2.1.1/patroni/postgresql/config.py#L1007
	status := pod.GetAnnotations()["status"]
	return strings.Contains(status, `"pending_restart":true`)
}
