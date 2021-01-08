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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// +kubebuilder:rbac:resources=endpoints;services,verbs=patch

// reconcilePatroniDistributedConfiguration sets labels and ownership on
// objects created by Patroni.
func (r *Reconciler) reconcilePatroniDistributedConfiguration(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
) error {
	// When using Endpoints for DCS, Patroni will create and write an Endpoints
	// object, but it won't set our role label nor cluster ownership.
	dcsEndpoints := &v1.Endpoints{ObjectMeta: naming.PatroniDistributedConfiguration(cluster)}
	dcsEndpoints.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Endpoints"))

	err := errors.WithStack(r.setControllerReference(cluster, dcsEndpoints))

	dcsEndpoints.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    "patroni",
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, dcsEndpoints, client.ForceOwnership))
	}

	// When using Endpoints for DCS, Patroni needs a Service to ensure that the
	// above Endpoints object is not removed by Kubernetes. Patroni will create
	// this object if it has permission to do so, but it won't set our role label
	// nor cluster ownership.
	// - https://github.com/zalando/patroni/blob/v2.0.1/patroni/dcs/kubernetes.py#L865-L881
	dcsService := &v1.Service{ObjectMeta: naming.PatroniDistributedConfiguration(cluster)}
	dcsService.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))

	if err == nil {
		err = errors.WithStack(r.setControllerReference(cluster, dcsService))
	}

	dcsService.Labels = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    "patroni",
	}

	// Allocate no IP address (headless) and create no Endpoints.
	// - https://docs.k8s.io/concepts/services-networking/service/#headless-services
	dcsService.Spec.ClusterIP = v1.ClusterIPNone
	dcsService.Spec.Selector = nil

	if err == nil {
		err = errors.WithStack(r.apply(ctx, dcsService, client.ForceOwnership))
	}

	// TODO(cbandy): Investigate "{scope}-failover" endpoints; is it DCS "failover_path"?
	// TODO(cbandy): Investigate DCS "sync_path".

	return err
}

// +kubebuilder:rbac:resources=pods,verbs=get;list

func (r *Reconciler) reconcilePatroniDynamicConfiguration(
	ctx context.Context, cluster *v1alpha1.PostgresCluster,
) error {
	// TODO(cbandy): Replace this with some automated indication that things
	// are _expected_ to be running. (Status?)
	if cluster.Spec.Patroni.EDC == nil || !*cluster.Spec.Patroni.EDC {
		return nil
	}

	// Deserialize the schemaless field. There will be no error because the
	// Kubernetes API has already ensured it is a JSON object.
	configuration := make(map[string]interface{})
	_ = yaml.Unmarshal(
		cluster.Spec.Patroni.DynamicConfiguration.Raw, &configuration,
	)

	// TODO(cbandy): Accumulate postgres settings. Perhaps arguments to the method?

	pgHBAs := postgres.HBAs{}
	pgHBAs.Mandatory = append(pgHBAs.Mandatory, *postgres.NewHBA().Local().User("postgres").Method("peer"))
	pgHBAs.Mandatory = append(pgHBAs.Mandatory, *postgres.NewHBA().TCP().Replication().Method("trust"))
	pgHBAs.Default = append(pgHBAs.Default, *postgres.NewHBA().TCP().Method("md5"))

	pgParameters := postgres.Parameters{}
	pgParameters.Mandatory = postgres.NewParameterSet()
	pgParameters.Mandatory.Add("wal_level", "logical")
	pgParameters.Default = postgres.NewParameterSet()
	pgParameters.Default.Add("jit", "off")

	configuration = patroni.DynamicConfiguration(configuration, pgHBAs, pgParameters)

	// TODO(cbandy): The above work should also be done at bootstrap. See Patroni
	// "bootstrap.dcs" YAML.

	pods := &v1.PodList{}
	instances, err := naming.AsSelector(naming.ClusterInstances(cluster.Name))
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
