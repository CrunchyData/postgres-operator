/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package postgrescluster

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources=endpoints,verbs=deletecollection

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
			return r.PodExec(pod.Namespace, pod.Name, container, stdin, stdout, stderr, command...)
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
			return r.PodExec(pod.Namespace, pod.Name, container, stdin, stdout, stderr, command...)
		})

		return errors.WithStack(exec.RestartPendingMembers(ctx, "replica", naming.PatroniScope(cluster)))
	}

	// Nothing needs to restart.
	return nil
}

// +kubebuilder:rbac:groups="",resources=services,verbs=create;patch

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

// +kubebuilder:rbac:resources=pods,verbs=get;list

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

	exec := func(_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string) error {
		return r.PodExec(pod.Namespace, pod.Name, naming.ContainerDatabase, stdin, stdout, stderr, command...)
	}

	var configuration map[string]interface{}
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
		cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelPatroni: naming.PatroniScope(cluster),
		})

	// Allocate an IP address and/or node port and let Patroni manage the Endpoints.
	// Patroni will ensure that they always route to the elected leader.
	// - https://docs.k8s.io/concepts/services-networking/service/#services-without-selectors
	service.Spec.Selector = nil
	if cluster.Spec.Service != nil {
		service.Spec.Type = corev1.ServiceType(cluster.Spec.Service.Type)
	} else {
		service.Spec.Type = corev1.ServiceTypeClusterIP
	}

	// The TargetPort must be the name (not the number) of the PostgreSQL
	// ContainerPort. This name allows the port number to differ between
	// instances, which can happen during a rolling update.
	service.Spec.Ports = []corev1.ServicePort{{
		Name:       naming.PortPostgreSQL,
		Port:       *cluster.Spec.Port,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(naming.PortPostgreSQL),
	}}

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

// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get

// reconcilePatroniStatus populates cluster.Status.Patroni with observations.
func (r *Reconciler) reconcilePatroniStatus(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	observedInstances *observedInstances,
) (reconcile.Result, error) {
	result := reconcile.Result{}
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
			result.RequeueAfter = 1 * time.Second
			return result, nil
		}
	}

	return result, err
}

// reconcileReplicationSecret creates a secret containing the TLS
// certificate, key and CA certificate for use with the replication and
// pg_rewind accounts in Postgres.
// TODO: As part of future work we will use this secret to setup a superuser
// account and enable cert authentication for that user
func (r *Reconciler) reconcileReplicationSecret(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	rootCACert *pki.RootCertificateAuthority,
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

	clientLeaf := pki.NewLeafCertificate("", nil, nil)
	clientLeaf.DNSNames = []string{postgres.ReplicationUser}
	clientLeaf.CommonName = clientLeaf.DNSNames[0]

	if data, ok := existing.Data[naming.ReplicationCert]; err == nil && ok {
		clientLeaf.Certificate, err = pki.ParseCertificate(data)
		err = errors.WithStack(err)
	}
	if data, ok := existing.Data[naming.ReplicationPrivateKey]; err == nil && ok {
		clientLeaf.PrivateKey, err = pki.ParsePrivateKey(data)
		err = errors.WithStack(err)
	}

	// if there is an error or the client leaf certificate is bad, generate a new one
	if err != nil || pki.LeafCertIsBad(ctx, clientLeaf, rootCACert, cluster.Namespace) {
		err = errors.WithStack(clientLeaf.Generate(rootCACert))
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
		intent.Data[naming.ReplicationCert], err = clientLeaf.Certificate.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		intent.Data[naming.ReplicationPrivateKey], err = clientLeaf.PrivateKey.MarshalText()
		err = errors.WithStack(err)
	}
	if err == nil {
		intent.Data[naming.ReplicationCACert], err = rootCACert.Certificate.MarshalText()
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

	if cluster.Spec.Patroni == nil || cluster.Spec.Patroni.Switchover == nil ||
		!cluster.Spec.Patroni.Switchover.Enabled {
		cluster.Status.Patroni.Switchover = nil
		return nil
	}

	annotation := cluster.GetAnnotations()[naming.PatroniSwitchover]
	spec := cluster.Spec.Patroni.Switchover
	status := cluster.Status.Patroni.Switchover

	if annotation == "" || (status != nil && *status == annotation) {
		return nil
	}

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
		return r.PodExec(runningPod.Namespace, runningPod.Name, naming.ContainerDatabase, stdin,
			stdout, stderr, command...)
	}

	// We have the pod executor, now we need to figure out which API call to use
	// In the default case we will be using SwitchoverAndWait. This API call uses
	// a Patronictl switchover to move to the target instance.
	action := func(ctx context.Context, exec patroni.Executor, next string) (bool, error) {
		success, err := patroni.Executor(exec).SwitchoverAndWait(ctx, next)
		return success, errors.WithStack(err)
	}

	if spec.Type == v1beta1.PatroniSwitchoverTypeFailover {
		// When a failover has been requested we use FailoverAndWait to change the primary.
		action = func(ctx context.Context, exec patroni.Executor, next string) (bool, error) {
			success, err := patroni.Executor(exec).FailoverAndWait(ctx, next)
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
	if err == nil {
		cluster.Status.Patroni.Switchover = initialize.String(annotation)
	}

	return err
}
