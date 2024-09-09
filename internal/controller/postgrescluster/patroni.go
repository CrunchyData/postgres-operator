// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="endpoints",verbs={deletecollection}

func (r *Reconciler) deletePatroniArtifacts(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) error {
	// TODO(cbandy): This could also be accomplished by adopting the Endpoints
	// as Patroni creates them. Would their events cause too many reconciles?
	// Foreground deletion may force us to adopt and set finalizers anyway.

	selector, err := naming.AsSelector(naming.ClusterPatronis(cluster))
	if err == nil {
		err = errors.WithStack(
			r.Client.DeleteAllOf(ctx, &corev1.Endpoints{},
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))
	}

	return err
}

func (r *Reconciler) handlePatroniRestarts(
	ctx context.Context, cluster *v1beta1.PostgresCluster, instances *observedInstances,
) error {
	const container = naming.ContainerDatabase
	var primaryNeedsRestart, replicaNeedsRestart *Instance

	// Look for one primary and one replica that need to restart. Ignore
	// containers that are terminating or not running; Kubernetes will start
	// them again, and calls to their Patroni API will likely be interrupted anyway.
	for _, instance := range instances.forCluster {
		if len(instance.Pods) > 0 && patroni.PodRequiresRestart(instance.Pods[0]) {
			if terminating, known := instance.IsTerminating(); terminating || !known {
				continue
			}
			if running, known := instance.IsRunning(container); !running || !known {
				continue
			}

			if primary, _ := instance.IsPrimary(); primary {
				primaryNeedsRestart = instance
			} else {
				replicaNeedsRestart = instance
			}
			if primaryNeedsRestart != nil && replicaNeedsRestart != nil {
				break
			}
		}
	}

	// When the primary instance needs to restart, restart it and return early.
	// Some PostgreSQL settings must be changed on the primary before any
	// progress can be made on the replicas, e.g. decreasing "max_connections".
	// Another reconcile will trigger when an instance with pending restarts
	// updates its status in DCS. See [Reconciler.watchPods].
	//
	// NOTE: In Patroni v2.1.1, regardless of the PostgreSQL parameter, the
	// primary indicates it needs to restart one "loop_wait" *after* the
	// replicas indicate it. So, even though we consider the primary ahead of
	// replicas here, replicas will typically restart first because we see them
	// first.
	if primaryNeedsRestart != nil {
		exec := patroni.Executor(func(
			ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			pod := primaryNeedsRestart.Pods[0]
			return r.PodExec(ctx, pod.Namespace, pod.Name, container, stdin, stdout, stderr, command...)
		})

		return errors.WithStack(exec.RestartPendingMembers(ctx, "master", naming.PatroniScope(cluster)))
	}

	// When the primary does not need to restart but a replica does, restart all
	// replicas that still need it.
	//
	// NOTE: This does not always clear the "needs restart" indicator on a replica.
	// Patroni sets that when a parameter must be increased to match the minimum
	// required of data on disk. When that happens, restarts occur (i.e. downtime)
	// but the affected parameter cannot change until the replica has replayed
	// the new minimum from the primary, e.g. decreasing "max_connections".
	// - https://github.com/zalando/patroni/blob/v2.1.1/patroni/postgresql/config.py#L1069
	//
	// TODO(cbandy): The above could interact badly with delayed replication.
	// When we offer per-instance PostgreSQL configuration, we may need to revisit
	// how we decide when to restart.
	// - https://www.postgresql.org/docs/current/runtime-config-replication.html
	if replicaNeedsRestart != nil {
		exec := patroni.Executor(func(
			ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			pod := replicaNeedsRestart.Pods[0]
			return r.PodExec(ctx, pod.Namespace, pod.Name, container, stdin, stdout, stderr, command...)
		})

		return errors.WithStack(exec.RestartPendingMembers(ctx, "replica", naming.PatroniScope(cluster)))
	}

	// Nothing needs to restart.
	return nil
}

// +kubebuilder:rbac:groups="",resources="services",verbs={create,patch}

// reconcilePatroniDistributedConfiguration sets labels and ownership on the
// objects Patroni creates for its distributed configuration.
func (r *Reconciler) reconcilePatroniDistributedConfiguration(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) error {
	// When using Endpoints for DCS, Patroni needs a Service to ensure that the
	// Endpoints object is not removed by Kubernetes at startup. Patroni will
	// create this object if it has permission to do so, but it won't set any
	// ownership.
	// - https://releases.k8s.io/v1.16.0/pkg/controller/endpoint/endpoints_controller.go#L547
	// - https://releases.k8s.io/v1.20.0/pkg/controller/endpoint/endpoints_controller.go#L580
	// - https://github.com/zalando/patroni/blob/v2.0.1/patroni/dcs/kubernetes.py#L865-L881
	dcsService := &corev1.Service{ObjectMeta: naming.PatroniDistributedConfiguration(cluster)}
	dcsService.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))

	err := errors.WithStack(r.setControllerReference(cluster, dcsService))

	dcsService.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil())
	dcsService.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelPatroni: naming.PatroniScope(cluster),
		})

	// Allocate no IP address (headless) and create no Endpoints.
	// - https://docs.k8s.io/concepts/services-networking/service/#headless-services
	dcsService.Spec.ClusterIP = corev1.ClusterIPNone
	dcsService.Spec.Selector = nil

	if err == nil {
		err = errors.WithStack(r.apply(ctx, dcsService))
	}

	// TODO(cbandy): DCS "failover_path"; `failover` and `switchover` create "{scope}-failover" endpoints.
	// TODO(cbandy): DCS "sync_path"; `synchronous_mode` uses "{scope}-sync" endpoints.

	return err
}

// +kubebuilder:rbac:resources="pods",verbs={get,list}

func (r *Reconciler) reconcilePatroniDynamicConfiguration(
	ctx context.Context, cluster *v1beta1.PostgresCluster, instances *observedInstances,
	pgHBAs postgres.HBAs, pgParameters postgres.Parameters,
) error {
	if !patroni.ClusterBootstrapped(cluster) {
		// Patroni has not yet bootstrapped. Dynamic configuration happens through
		// configuration files during bootstrap, so there's nothing to do here.
		return nil
	}

	var pod *corev1.Pod
	for _, instance := range instances.forCluster {
		if terminating, known := instance.IsTerminating(); !terminating && known {
			running, known := instance.IsRunning(naming.ContainerDatabase)

			if running && known && len(instance.Pods) > 0 {
				pod = instance.Pods[0]
				break
			}
		}
	}
	if pod == nil {
		// There are no running Patroni containers; nothing to do.
		return nil
	}

	// NOTE(cbandy): Despite the guards above, calling PodExec may still fail
	// due to a missing or stopped container.

	exec := func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string) error {
		return r.PodExec(ctx, pod.Namespace, pod.Name, naming.ContainerDatabase, stdin, stdout, stderr, command...)
	}

	var configuration map[string]any
	if cluster.Spec.Patroni != nil {
		configuration = cluster.Spec.Patroni.DynamicConfiguration
	}
	configuration = patroni.DynamicConfiguration(cluster, configuration, pgHBAs, pgParameters)

	return errors.WithStack(
		patroni.Executor(exec).ReplaceConfiguration(ctx, configuration))
}

// generatePatroniLeaderLeaseService returns a v1.Service that exposes the
// Patroni leader when Patroni is using Endpoints for its leader elections.
func (r *Reconciler) generatePatroniLeaderLeaseService(
	cluster *v1beta1.PostgresCluster) (*corev1.Service, error,
) {
	service := &corev1.Service{ObjectMeta: naming.PatroniLeaderEndpoints(cluster)}
	service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))

	service.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil())
	service.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil())

	if spec := cluster.Spec.Service; spec != nil {
		service.Annotations = naming.Merge(service.Annotations,
			spec.Metadata.GetAnnotationsOrNil())
		service.Labels = naming.Merge(service.Labels,
			spec.Metadata.GetLabelsOrNil())
	}

	// add our labels last so they aren't overwritten
	service.Labels = naming.Merge(service.Labels,
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelPatroni: naming.PatroniScope(cluster),
		})

	// Allocate an IP address and/or node port and let Patroni manage the Endpoints.
	// Patroni will ensure that they always route to the elected leader.
	// - https://docs.k8s.io/concepts/services-networking/service/#services-without-selectors
	service.Spec.Selector = nil

	// The TargetPort must be the name (not the number) of the PostgreSQL
	// ContainerPort. This name allows the port number to differ between
	// instances, which can happen during a rolling update.
	servicePort := corev1.ServicePort{
		Name:       naming.PortPostgreSQL,
		Port:       *cluster.Spec.Port,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(naming.PortPostgreSQL),
	}

	if spec := cluster.Spec.Service; spec == nil {
		service.Spec.Type = corev1.ServiceTypeClusterIP
	} else {
		service.Spec.Type = corev1.ServiceType(spec.Type)
		if spec.NodePort != nil {
			if service.Spec.Type == corev1.ServiceTypeClusterIP {
				// The NodePort can only be set when the Service type is NodePort or
				// LoadBalancer. However, due to a known issue prior to Kubernetes
				// 1.20, we clear these errors during our apply. To preserve the
				// appropriate behavior, we log an Event and return an error.
				// TODO(tjmoore4): Once Validation Rules are available, this check
				// and event could potentially be removed in favor of that validation
				r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "MisconfiguredClusterIP",
					"NodePort cannot be set with type ClusterIP on Service %q", service.Name)
				return nil, fmt.Errorf("NodePort cannot be set with type ClusterIP on Service %q", service.Name)
			}
			servicePort.NodePort = *spec.NodePort
		}
	}
	service.Spec.Ports = []corev1.ServicePort{servicePort}

	err := errors.WithStack(r.setControllerReference(cluster, service))
	return service, err
}

// +kubebuilder:rbac:groups="",resources="services",verbs={create,patch}

// reconcilePatroniLeaderLease sets labels and ownership on the objects Patroni
// creates for its leader elections. When Patroni is using Endpoints for this,
// the returned Service resolves to the elected leader. Otherwise, it is nil.
func (r *Reconciler) reconcilePatroniLeaderLease(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*corev1.Service, error) {
	// When using Endpoints for DCS, Patroni needs a Service to ensure that the
	// Endpoints object is not removed by Kubernetes at startup.
	// - https://releases.k8s.io/v1.16.0/pkg/controller/endpoint/endpoints_controller.go#L547
	// - https://releases.k8s.io/v1.20.0/pkg/controller/endpoint/endpoints_controller.go#L580
	service, err := r.generatePatroniLeaderLeaseService(cluster)
	if err == nil {
		err = errors.WithStack(r.apply(ctx, service))
	}
	return service, err
}

// +kubebuilder:rbac:groups="",resources="endpoints",verbs={get}

// reconcilePatroniStatus populates cluster.Status.Patroni with observations.
func (r *Reconciler) reconcilePatroniStatus(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	observedInstances *observedInstances,
) (time.Duration, error) {
	var requeue time.Duration
	log := logging.FromContext(ctx)

	var readyInstance bool
	for _, instance := range observedInstances.forCluster {
		if r, _ := instance.IsReady(); r {
			readyInstance = true
		}
	}

	dcs := &corev1.Endpoints{ObjectMeta: naming.PatroniDistributedConfiguration(cluster)}
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(dcs), dcs)))

	if err == nil {
		if dcs.Annotations["initialize"] != "" {
			// After bootstrap, Patroni writes the cluster system identifier to DCS.
			cluster.Status.Patroni.SystemIdentifier = dcs.Annotations["initialize"]
		} else if readyInstance {
			// While we typically expect a value for the initialize key to be present in the
			// Endpoints above by the time the StatefulSet for any instance indicates "ready"
			// (since Patroni writes this value after successful cluster bootstrap, at which time
			// the initial primary should transition to "ready"), sometimes this is not the case
			// and the "initialize" key is not yet present.  Therefore, if a "ready" instance
			// is detected in the cluster we assume this is the case, and simply log a message and
			// requeue in order to try again until the expected value is found.
			log.Info("detected ready instance but no initialize value")
			requeue = time.Second
		}
	}

	return requeue, err
}

// reconcileReplicationSecret creates a secret containing the TLS
// certificate, key and CA certificate for use with the replication and
// pg_rewind accounts in Postgres.
// TODO: As part of future work we will use this secret to setup a superuser
// account and enable cert authentication for that user
func (r *Reconciler) reconcileReplicationSecret(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	root *pki.RootCertificateAuthority,
) (*corev1.Secret, error) {

	// if a custom postgrescluster secret is provided, just return it
	if cluster.Spec.CustomReplicationClientTLSSecret != nil {
		custom := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Spec.CustomReplicationClientTLSSecret.Name,
			Namespace: cluster.Namespace,
		}}
		err := errors.WithStack(r.Client.Get(ctx,
			client.ObjectKeyFromObject(custom), custom))
		return custom, err
	}

	existing := &corev1.Secret{ObjectMeta: naming.ReplicationClientCertSecret(cluster)}
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing)))

	leaf := &pki.LeafCertificate{}
	commonName := postgres.ReplicationUser
	dnsNames := []string{commonName}

	if err == nil {
		// Unmarshal and validate the stored leaf. These first errors can
		// be ignored because they result in an invalid leaf which is then
		// correctly regenerated.
		_ = leaf.Certificate.UnmarshalText(existing.Data[naming.ReplicationCert])
		_ = leaf.PrivateKey.UnmarshalText(existing.Data[naming.ReplicationPrivateKey])

		leaf, err = root.RegenerateLeafWhenNecessary(leaf, commonName, dnsNames)
		err = errors.WithStack(err)
	}

	intent := &corev1.Secret{ObjectMeta: naming.ReplicationClientCertSecret(cluster)}
	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	intent.Data = make(map[string][]byte)

	// set labels and annotations
	intent.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil())
	intent.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:            cluster.Name,
			naming.LabelClusterCertificate: "replication-client-tls",
		})

	if err := errors.WithStack(r.setControllerReference(cluster, intent)); err != nil {
		return nil, err
	}
	if err == nil {
		intent.Data[naming.ReplicationCert], err = leaf.Certificate.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		intent.Data[naming.ReplicationPrivateKey], err = leaf.PrivateKey.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		intent.Data[naming.ReplicationCACert], err = root.Certificate.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, intent))
	}
	return intent, err
}

// replicationCertSecretProjection returns a secret projection of the postgrescluster's
// client certificate and key to include in the instance configuration volume.
func replicationCertSecretProjection(certificate *corev1.Secret) *corev1.SecretProjection {
	return &corev1.SecretProjection{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: certificate.Name,
		},
		Items: []corev1.KeyToPath{
			{
				Key:  naming.ReplicationCert,
				Path: naming.ReplicationCertPath,
			},
			{
				Key:  naming.ReplicationPrivateKey,
				Path: naming.ReplicationPrivateKeyPath,
			},
			{
				Key:  naming.ReplicationCACert,
				Path: naming.ReplicationCACertPath,
			},
		},
	}
}

func (r *Reconciler) reconcilePatroniSwitchover(ctx context.Context,
	cluster *v1beta1.PostgresCluster, instances *observedInstances) error {
	log := logging.FromContext(ctx)

	// If switchover is not enabled, clear out the Patroni switchover status fields
	// which might have been set by previous switchovers.
	// This also gives the user a way to easily recover and try again: if the operator
	// runs into a problem with a switchover, turning `cluster.Spec.Patroni.Switchover`
	// to `false` will clear the fields before another attempt
	if cluster.Spec.Patroni == nil ||
		cluster.Spec.Patroni.Switchover == nil ||
		!cluster.Spec.Patroni.Switchover.Enabled {
		cluster.Status.Patroni.Switchover = nil
		cluster.Status.Patroni.SwitchoverTimeline = nil
		return nil
	}

	annotation := cluster.GetAnnotations()[naming.PatroniSwitchover]
	spec := cluster.Spec.Patroni.Switchover
	status := cluster.Status.Patroni.Switchover

	// If the status has been updated with the trigger annotation, the requested
	// switchover has been successful, and the `SwitchoverTimeline` field can be cleared
	if annotation == "" || (status != nil && *status == annotation) {
		cluster.Status.Patroni.SwitchoverTimeline = nil
		return nil
	}

	// If we've reached this point, we assume a switchover request or in progress
	// and need to make sure the prerequisites are met, e.g., more than one pod,
	// a running instance to issue the switchover command to, etc.
	if len(instances.forCluster) <= 1 {
		// TODO: event
		// TODO: Possible webhook validation
		return errors.New("Need more than one instance to switchover")
	}

	// 	 TODO: Add webhook validation that requires a targetInstance when requesting failover
	if spec.Type == v1beta1.PatroniSwitchoverTypeFailover {
		if spec.TargetInstance == nil || *spec.TargetInstance == "" {
			// TODO: event
			return errors.New("TargetInstance required when running failover")
		}
	}

	// Determine if user is specifying a target instance. Validate the
	// provided instance has been observed in the cluster.
	var targetInstance *Instance
	if spec.TargetInstance != nil && *spec.TargetInstance != "" {
		for _, instance := range instances.forCluster {
			if *spec.TargetInstance == instance.Name {
				targetInstance = instance
			}
		}
		if targetInstance == nil {
			// TODO: event
			return errors.New("TargetInstance was specified but not found in the cluster")
		}
		if len(targetInstance.Pods) != 1 {
			// We expect that a target instance should have one associated pod.
			return errors.Errorf(
				"TargetInstance should have one pod. Pods (%d)", len(targetInstance.Pods))
		}
	} else {
		log.V(1).Info("TargetInstance not provided")
	}

	// Find a running Pod that can be used to define a PodExec function.
	var runningPod *corev1.Pod
	for _, instance := range instances.forCluster {
		if running, known := instance.IsRunning(naming.ContainerDatabase); running &&
			known && len(instance.Pods) == 1 {

			runningPod = instance.Pods[0]
			break
		}
	}
	if runningPod == nil {
		return errors.New("Could not find a running pod when attempting switchover.")
	}
	exec := func(_ context.Context, stdin io.Reader, stdout, stderr io.Writer,
		command ...string) error {
		return r.PodExec(ctx, runningPod.Namespace, runningPod.Name, naming.ContainerDatabase, stdin,
			stdout, stderr, command...)
	}

	// To ensure idempotency, the operator verifies that the timeline reported by Patroni
	// matches the timeline that was present when the switchover was first requested.
	// TODO(benjaminjb): consider pulling the timeline from the pod annotation; manual experiments
	// have shown that the annotation on the Leader pod is up to date during a switchover, but
	// missing from the Replica pods.
	timeline, err := patroni.Executor(exec).GetTimeline(ctx)

	if err != nil {
		return err
	}

	if timeline == 0 {
		return errors.New("error getting and parsing current timeline")
	}

	statusTimeline := cluster.Status.Patroni.SwitchoverTimeline

	// If the `SwitchoverTimeline` field is empty, this is the first reconcile after
	// a switchover has been requested and we need to fill in the field with the current TL
	// as reported by Patroni.
	// We return from here without calling for an explicit requeue, but since we're updating
	// the object, we will reconcile this again for the actual switchover/failover action.
	if statusTimeline == nil || (statusTimeline != nil && *statusTimeline == 0) {
		log.V(1).Info("Setting SwitchoverTimeline", "timeline", timeline)
		cluster.Status.Patroni.SwitchoverTimeline = &timeline
		return nil
	}

	// If the `SwitchoverTimeline` field does not match the current timeline as reported by Patroni,
	// then we assume a switchover has been completed, and we have reached this point because the
	// cache does not yet have the updated `cluster.Status.Patroni.Switchover` field.
	if statusTimeline != nil && *statusTimeline != timeline {
		log.V(1).Info("SwitchoverTimeline does not match current timeline, assuming already completed switchover")
		cluster.Status.Patroni.Switchover = initialize.String(annotation)
		cluster.Status.Patroni.SwitchoverTimeline = nil
		return nil
	}

	// We have the pod executor, now we need to figure out which API call to use
	// In the default case we will be using SwitchoverAndWait. This API call uses
	// a Patronictl switchover to move to the target instance.
	action := func(ctx context.Context, exec patroni.Executor, next string) (bool, error) {
		success, err := exec.SwitchoverAndWait(ctx, next)
		return success, errors.WithStack(err)
	}

	if spec.Type == v1beta1.PatroniSwitchoverTypeFailover {
		// When a failover has been requested we use FailoverAndWait to change the primary.
		action = func(ctx context.Context, exec patroni.Executor, next string) (bool, error) {
			success, err := exec.FailoverAndWait(ctx, next)
			return success, errors.WithStack(err)
		}
	}

	// If target instance has not been provided, we will pass in an empty string to patronictl
	nextPrimary := ""
	if targetInstance != nil {
		nextPrimary = targetInstance.Pods[0].Name
	}

	success, err := action(ctx, exec, nextPrimary)
	if err = errors.WithStack(err); err == nil && !success {
		err = errors.New("unable to switchover")
	}

	// If we've reached this point, a switchover has successfully been triggered
	// and we set the status accordingly.
	if err == nil {
		cluster.Status.Patroni.Switchover = initialize.String(annotation)
		cluster.Status.Patroni.SwitchoverTimeline = nil
	}

	return err
}
