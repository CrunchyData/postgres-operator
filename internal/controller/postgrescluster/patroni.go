/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

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
			r.Client.DeleteAllOf(ctx, &v1.Endpoints{},
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))
	}

	return err
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
	dcsService := &v1.Service{ObjectMeta: naming.PatroniDistributedConfiguration(cluster)}
	dcsService.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))

	err := errors.WithStack(r.setControllerReference(cluster, dcsService))

	dcsService.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelPatroni: naming.PatroniScope(cluster),
	}

	// Allocate no IP address (headless) and create no Endpoints.
	// - https://docs.k8s.io/concepts/services-networking/service/#headless-services
	dcsService.Spec.ClusterIP = v1.ClusterIPNone
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
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	pgHBAs postgres.HBAs, pgParameters postgres.Parameters,
) error {
	if !patroni.ClusterBootstrapped(cluster) {
		// Patroni has not yet bootstrapped. Dynamic configuration happens through
		// configuration files during bootstrap, so there's nothing to do here.
		return nil
	}

	// Deserialize the schemaless field. There will be no error because the
	// Kubernetes API has already ensured it is a JSON object.
	configuration := make(map[string]interface{})
	_ = yaml.Unmarshal(
		cluster.Spec.Patroni.DynamicConfiguration.Raw, &configuration,
	)

	configuration = patroni.DynamicConfiguration(cluster, configuration, pgHBAs, pgParameters)

	pods := &v1.PodList{}
	instances, err := naming.AsSelector(naming.ClusterPatronis(cluster))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, pods,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: instances},
			))
	}

	var pod *v1.Pod
	if err == nil {
		for i := range pods.Items {
			if pods.Items[i].Status.Phase == v1.PodRunning {
				pod = &pods.Items[i]
				break
			}
		}
		if pod == nil {
			err = errors.New("could not find a running pod")
		}
	}
	if err == nil {
		exec := func(_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string) error {
			return r.PodExec(pod.Namespace, pod.Name, naming.ContainerDatabase, stdin, stdout, stderr, command...)
		}
		err = errors.WithStack(
			patroni.Executor(exec).ReplaceConfiguration(ctx, configuration))
	}

	return err
}

// +kubebuilder:rbac:groups="",resources=services,verbs=create;patch

// reconcilePatroniLeaderLease sets labels and ownership on the objects Patroni
// creates for its leader elections. When Patroni is using Endpoints for this,
// the returned Service resolves to the elected leader. Otherwise, it is nil.
func (r *Reconciler) reconcilePatroniLeaderLease(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*v1.Service, error) {
	// When using Endpoints for DCS, Patroni needs a Service to ensure that the
	// Endpoints object is not removed by Kubernetes at startup.
	// - https://releases.k8s.io/v1.16.0/pkg/controller/endpoint/endpoints_controller.go#L547
	// - https://releases.k8s.io/v1.20.0/pkg/controller/endpoint/endpoints_controller.go#L580
	leaderService := &v1.Service{ObjectMeta: naming.PatroniLeaderEndpoints(cluster)}
	leaderService.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))

	err := errors.WithStack(r.setControllerReference(cluster, leaderService))

	leaderService.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelPatroni: naming.PatroniScope(cluster),
	}

	// Allocate an IP address and let Patroni manage the Endpoints. Patroni will
	// ensure that they always route to the elected leader.
	// - https://docs.k8s.io/concepts/services-networking/service/#services-without-selectors
	leaderService.Spec.Type = v1.ServiceTypeClusterIP
	leaderService.Spec.Selector = nil

	// The TargetPort must be the name (not the number) of the PostgreSQL
	// ContainerPort. This name allows the port number to differ between
	// instances, which can happen during a rolling update.
	leaderService.Spec.Ports = []v1.ServicePort{{
		Name:       naming.PortPostgreSQL,
		Port:       *cluster.Spec.Port,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromString(naming.PortPostgreSQL),
	}}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, leaderService))
	}

	return leaderService, err
}

// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get

// reconcilePatroniStatus populates cluster.Status.Patroni with observations.
func (r *Reconciler) reconcilePatroniStatus(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) error {
	var status v1beta1.PatroniStatus

	dcs := &v1.Endpoints{ObjectMeta: naming.PatroniDistributedConfiguration(cluster)}
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(dcs), dcs)))

	if err == nil && dcs.Annotations["initialize"] != "" {
		// After bootstrap, Patroni writes the cluster system identifier to DCS.
		status.SystemIdentifier = dcs.Annotations["initialize"]
		cluster.Status.Patroni = &status
		return nil
	}

	// While we typically expect a value for the initialize key to be present in the Endpoints
	// above by the time the StatefulSet for any instance indicates "ready" (since Patroni writes
	// this value after successful cluster bootstrap, at which time the initial primary should
	// transition to "ready"), sometimes this is not the case and the "initialize" key is not yet
	// present.  Therefore, if a "ready" instance StatefulSet is detected in the cluster (as
	// determined by its ReadyReplicas) we assume this is the case, and return an error in order to
	// requeue and try again until the expected value is found.  Please note that another option
	// would be to watch the Endpoints for the "initialize" value, which may be considered in a
	// future implementation.
	var instanceSelector labels.Selector
	if err == nil {
		instanceSelector, err = naming.AsSelector(naming.ClusterInstances(cluster.GetName()))
	}

	instances := &appsv1.StatefulSetList{}
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, instances,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: instanceSelector},
			))
	}

	if err == nil {
		for _, instance := range instances.Items {
			if instance.Status.ReadyReplicas != 0 {
				// TODO(andrewlecuyer): Returning an error to address a missing identifier in the
				// Endpoints (despite a "ready" instance) is a symptom of a missed event.  Consider
				// watching Endpoints instead to ensure the required events are not missed.
				return errors.New("detected ready instance but no initialize value")
			}
		}
	}

	return err
}

// reconcileReplicationSecret creates a secret containing the TLS
// certificate, key and CA certificate for use with the replication and
// pg_rewind accounts in Postgres.
// TODO: As part of future work we will use this secret to setup a superuser
// account and enable cert authentication for that user
func (r *Reconciler) reconcileReplicationSecret(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	rootCACert *pki.RootCertificateAuthority,
) (*v1.Secret, error) {

	// if a custom postgrescluster secret is provided, just return it
	if cluster.Spec.CustomReplicationClientTLSSecret != nil {
		custom := &v1.Secret{ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Spec.CustomReplicationClientTLSSecret.Name,
			Namespace: cluster.Namespace,
		}}
		err := errors.WithStack(r.Client.Get(ctx,
			client.ObjectKeyFromObject(custom), custom))
		if err == nil {
			return custom, err
		}
		return nil, err
	}

	existing := &v1.Secret{ObjectMeta: naming.ReplicationClientCertSecret(cluster)}
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing)))

	clientLeaf := pki.NewLeafCertificate("", nil, nil)
	clientLeaf.DNSNames = []string{naming.PGReplicationUsername}
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

	intent := &v1.Secret{ObjectMeta: naming.ReplicationClientCertSecret(cluster)}
	intent.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
	intent.Data = make(map[string][]byte)

	// set labels
	intent.Labels = map[string]string{
		naming.LabelCluster:            cluster.Name,
		naming.LabelClusterCertificate: "replication-client-tls",
	}

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
	if err == nil {
		return intent, err
	}
	return nil, err
}

// replicationCertSecretProjection returns a secret projection of the postgrescluster's
// client certificate and key to include in the instance configuration volume.
func replicationCertSecretProjection(certificate *v1.Secret) *v1.SecretProjection {
	return &v1.SecretProjection{
		LocalObjectReference: v1.LocalObjectReference{
			Name: certificate.Name,
		},
		Items: []v1.KeyToPath{
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
