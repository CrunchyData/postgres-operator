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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// +kubebuilder:rbac:resources=endpoints,verbs=deletecollection

func (r *Reconciler) deletePatroniArtifacts(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
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

// +kubebuilder:rbac:resources=services,verbs=patch

// reconcilePatroniDistributedConfiguration sets labels and ownership on the
// objects Patroni creates for its distributed configuration.
func (r *Reconciler) reconcilePatroniDistributedConfiguration(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
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
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
	pgHBAs postgres.HBAs, pgParameters postgres.Parameters,
) error {
	if cluster.Status.Patroni == nil || cluster.Status.Patroni.SystemIdentifier == "" {
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

// +kubebuilder:rbac:resources=services,verbs=patch

// reconcilePatroniLeaderLease sets labels and ownership on the objects Patroni
// creates for its leader elections. When Patroni is using Endpoints for this,
// the returned Service resolves to the elected leader. Otherwise, it is nil.
func (r *Reconciler) reconcilePatroniLeaderLease(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
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

// +kubebuilder:rbac:resources=endpoints,verbs=get

// reconcilePatroniStatus populates cluster.Status.Patroni with observations.
func (r *Reconciler) reconcilePatroniStatus(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
) error {
	var status v1alpha1.PatroniStatus

	dcs := &v1.Endpoints{ObjectMeta: naming.PatroniDistributedConfiguration(cluster)}
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(dcs), dcs)))

	if err == nil {
		// After bootstrap, Patroni writes the cluster system identifier to DCS.
		status.SystemIdentifier = dcs.Annotations["initialize"]
	}

	cluster.Status.Patroni = &status
	return err
}
