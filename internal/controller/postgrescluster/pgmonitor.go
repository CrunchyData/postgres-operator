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
	"fmt"
	"io"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgmonitor"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	exporterPort = int32(9187)

	// TODO: With the current implementation of the crunchy-postgres-exporter
	// it makes sense to hard-code the database. When moving away from the
	// crunchy-postgres-exporter start.sh script we should re-evaluate always
	// setting the exporter database to `postgres`.
	exporterDB = "postgres"

	// The exporter connects to all databases over loopback using a password.
	// Kubernetes guarantees localhost resolves to loopback:
	// https://kubernetes.io/docs/concepts/cluster-administration/networking/
	// https://releases.k8s.io/v1.21.0/pkg/kubelet/kubelet_pods.go#L343
	exporterHost = "localhost"
)

var (
	oneMillicore = resource.MustParse("1m")
	oneMebibyte  = resource.MustParse("1Mi")
)

// If pgMonitor is enabled the pgMonitor sidecar(s) have been added to the
// instance pod. reconcilePGMonitor will update the database to
// create the necessary objects for the tool to run
func (r *Reconciler) reconcilePGMonitor(ctx context.Context,
	cluster *v1beta1.PostgresCluster, instances *observedInstances,
	monitoringSecret *corev1.Secret) error {

	err := r.reconcilePGMonitorExporter(ctx, cluster, instances, monitoringSecret)

	return err
}

// reconcilePGMonitorExporter performs setup the postgres_exporter sidecar
// - PodExec to get setup.sql file for the postgres version
// - PodExec to run the sql in the primary database
// Status.Monitoring.ExporterConfiguration is used to determine when the
// pgMonitor postgres_exporter configuration should be added/changed to
// limit how often PodExec is used
// - TODO jmckulk: kube perms comment?
func (r *Reconciler) reconcilePGMonitorExporter(ctx context.Context,
	cluster *v1beta1.PostgresCluster, instances *observedInstances,
	monitoringSecret *corev1.Secret) error {

	var (
		writableInstance *Instance
		writablePod      *corev1.Pod
		setup            string
	)

	// Find the PostgreSQL instance that can execute SQL that writes to every
	// database. When there is none, return early.

	for _, instance := range instances.forCluster {
		if terminating, known := instance.IsTerminating(); terminating || !known {
			continue
		}
		if writable, known := instance.IsWritable(); !writable || !known {
			continue
		}

		running, known := instance.IsRunning(naming.ContainerDatabase)
		if running && known && len(instance.Pods) > 0 {
			writableInstance = instance
		}
	}

	if writableInstance == nil {
		return nil
	}

	writablePod = writableInstance.Pods[0]

	if pgmonitor.ExporterEnabled(cluster) {
		running, known := writableInstance.IsRunning(naming.ContainerPGMonitorExporter)
		if !running || !known {
			// Exporter container needs to be available to get setup.sql;
			return nil
		}

		for _, containerStatus := range writablePod.Status.ContainerStatuses {
			if containerStatus.Name == naming.ContainerPGMonitorExporter {
				setup = containerStatus.ImageID
			}
		}
		if setup == "" {
			// Could not get exporter container imageID
			return nil
		}
	}

	// PostgreSQL is available for writes. Prepare to either add or remove
	// pgMonitor objects.

	action := func(ctx context.Context, exec postgres.Executor) error {
		return pgmonitor.EnableExporterInPostgreSQL(ctx, exec, monitoringSecret, exporterDB, setup)
	}

	if !pgmonitor.ExporterEnabled(cluster) {
		action = func(ctx context.Context, exec postgres.Executor) error {
			return pgmonitor.DisableExporterInPostgreSQL(ctx, exec)
		}
	}

	revision, err := safeHash32(func(hasher io.Writer) error {
		// Discard log message from pgmonitor package about executing SQL.
		// Nothing is being "executed" yet.
		return action(logging.NewContext(ctx, logging.Discard()), func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			_, err := io.Copy(hasher, stdin)
			if err == nil {
				_, err = fmt.Fprint(hasher, command)
			}
			return err
		})
	})
	if err != nil {
		return err
	}

	if revision != cluster.Status.Monitoring.ExporterConfiguration {
		// The configuration is out of date and needs to be updated.
		// Include the revision hash in any log messages.
		ctx := logging.NewContext(ctx, logging.FromContext(ctx).WithValues("revision", revision))

		if pgmonitor.ExporterEnabled(cluster) {
			exec := func(_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string) error {
				return r.PodExec(writablePod.Namespace, writablePod.Name, naming.ContainerPGMonitorExporter, stdin, stdout, stderr, command...)
			}
			setup, _, err = pgmonitor.Executor(exec).GetExporterSetupSQL(ctx, cluster.Spec.PostgresVersion)
		}

		// Apply the necessary SQL and record its hash in cluster.Status

		if err == nil {
			err = action(ctx, func(_ context.Context, stdin io.Reader,
				stdout, stderr io.Writer, command ...string) error {
				return r.PodExec(writablePod.Namespace, writablePod.Name, naming.ContainerDatabase, stdin, stdout, stderr, command...)
			})
		}
		if err == nil {
			cluster.Status.Monitoring.ExporterConfiguration = revision
		}
	}

	return err
}

// reconcileMonitoringSecret reconciles the secret containing authentication
// for monitoring tools
func (r *Reconciler) reconcileMonitoringSecret(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster) (*corev1.Secret, error) {

	existing := &corev1.Secret{ObjectMeta: naming.MonitoringUserSecret(cluster)}
	err := errors.WithStack(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing))
	if client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	if !pgmonitor.ExporterEnabled(cluster) {
		// TODO: Checking if the exporter is enabled to determine when monitoring
		// secret should be created. If more tools are added to the monitoring
		// suite, they could need the secret when the exporter is not enabled.
		// This check may need to be updated.
		// Exporter is disabled; delete monitoring secret if it exists.
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, existing))
		}
		return nil, client.IgnoreNotFound(err)
	}

	intent := &corev1.Secret{ObjectMeta: naming.MonitoringUserSecret(cluster)}
	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))

	intent.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
	)
	intent.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RoleMonitoring,
		})

	intent.Data = make(map[string][]byte)

	if len(existing.Data["password"]) == 0 || len(existing.Data["verifier"]) == 0 {
		password, err := util.GeneratePassword(util.DefaultGeneratedPasswordLength)
		if err != nil {
			return nil, err
		}

		// Generate the SCRAM verifier now and store alongside the plaintext
		// password so that later reconciles don't generate it repeatedly.
		// NOTE(cbandy): We don't have a function to compare a plaintext password
		// to a SCRAM verifier.
		verifier, err := pgpassword.NewSCRAMPassword(password).Build()
		if err != nil {
			return nil, err
		}
		intent.Data["password"] = []byte(password)
		intent.Data["verifier"] = []byte(verifier)
	} else {
		intent.Data["password"] = existing.Data["password"]
		intent.Data["verifier"] = existing.Data["verifier"]
	}

	err = errors.WithStack(r.setControllerReference(cluster, intent))
	if err == nil {
		err = errors.WithStack(r.apply(ctx, intent))
	}
	if err == nil {
		return intent, nil
	}

	return nil, err
}

// addPGMonitorToInstancePodSpec performs the necessary setup to add
// pgMonitor resources on a PodTemplateSpec
func addPGMonitorToInstancePodSpec(
	cluster *v1beta1.PostgresCluster,
	template *corev1.PodTemplateSpec) error {

	err := addPGMonitorExporterToInstancePodSpec(cluster, template)

	return err
}

// addPGMonitorExporterToInstancePodSpec performs the necessary setup to
// add pgMonitor exporter resources to a PodTemplateSpec
// TODO jmckulk: refactor to pass around monitoring secret; Without the secret
// the exporter container cannot be created; Testing relies on ensuring the
// monitoring secret is available
func addPGMonitorExporterToInstancePodSpec(
	cluster *v1beta1.PostgresCluster,
	template *corev1.PodTemplateSpec) error {

	if !pgmonitor.ExporterEnabled(cluster) {
		return nil
	}

	securityContext := initialize.RestrictedSecurityContext()
	exporterContainer := corev1.Container{
		Name:      naming.ContainerPGMonitorExporter,
		Image:     config.PGExporterContainerImage(cluster),
		Resources: cluster.Spec.Monitoring.PGMonitor.Exporter.Resources,
		Command: []string{
			"/opt/cpm/bin/start.sh",
		},
		Env: []corev1.EnvVar{
			{Name: "CONFIG_DIR", Value: "/opt/cpm/conf"},
			{Name: "POSTGRES_EXPORTER_PORT", Value: fmt.Sprint(exporterPort)},
			{Name: "PGBACKREST_INFO_THROTTLE_MINUTES", Value: "10"},
			{Name: "PG_STAT_STATEMENTS_LIMIT", Value: "20"},
			{Name: "PG_STAT_STATEMENTS_THROTTLE_MINUTES", Value: "-1"},
			{Name: "EXPORTER_PG_HOST", Value: exporterHost},
			{Name: "EXPORTER_PG_PORT", Value: fmt.Sprint(*cluster.Spec.Port)},
			{Name: "EXPORTER_PG_DATABASE", Value: exporterDB},
			{Name: "EXPORTER_PG_USER", Value: pgmonitor.MonitoringUser},
			{Name: "EXPORTER_PG_PASSWORD", ValueFrom: &corev1.EnvVarSource{
				// Environment variables are not updated after a secret update.
				// This could lead to a state where the exporter does not have
				// the correct password and the container needs to restart.
				// https://kubernetes.io/docs/concepts/configuration/secret/#environment-variables-are-not-updated-after-a-secret-update
				// https://github.com/kubernetes/kubernetes/issues/29761
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: naming.MonitoringUserSecret(cluster).Name,
					},
					Key: "password",
				},
			}},
		},
		SecurityContext: securityContext,
		// ContainerPort is needed to support proper target discovery by Prometheus for pgMonitor
		// integration
		Ports: []corev1.ContainerPort{{
			ContainerPort: exporterPort,
			Name:          naming.PortExporter,
			Protocol:      corev1.ProtocolTCP,
		}},
		VolumeMounts: []corev1.VolumeMount{{
			Name: "exporter-config",
			// this is the path for custom config as defined in the start.sh script for the exporter container
			MountPath: "/conf",
		}},
	}
	template.Spec.Containers = append(template.Spec.Containers, exporterContainer)

	// add custom exporter config volume
	configVolume := corev1.Volume{
		Name: "exporter-config",
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: cluster.Spec.Monitoring.PGMonitor.Exporter.Configuration,
			},
		},
	}
	template.Spec.Volumes = append(template.Spec.Volumes, configVolume)

	podInfoVolume := corev1.Volume{
		Name: "podinfo",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				// The paths defined in Items (cpu_limit, cpu_request, etc.)
				// are hard coded in the pgnodemx queries defined by
				// pgMonitor configuration (queries_nodemx.yml)
				// https://github.com/CrunchyData/pgmonitor/blob/master/exporter/postgres/queries_nodemx.yml
				Items: []corev1.DownwardAPIVolumeFile{{
					Path: "cpu_limit",
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: naming.ContainerDatabase,
						Resource:      "limits.cpu",
						Divisor:       oneMillicore,
					},
				}, {
					Path: "cpu_request",
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: naming.ContainerDatabase,
						Resource:      "requests.cpu",
						Divisor:       oneMillicore,
					},
				}, {
					Path: "mem_limit",
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: naming.ContainerDatabase,
						Resource:      "limits.memory",
						Divisor:       oneMebibyte,
					},
				}, {
					Path: "mem_request",
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: naming.ContainerDatabase,
						Resource:      "requests.memory",
						Divisor:       oneMebibyte,
					},
				}, {
					Path: "labels",
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: corev1.SchemeGroupVersion.Version,
						FieldPath:  "metadata.labels",
					},
				}, {
					Path: "annotations",
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: corev1.SchemeGroupVersion.Version,
						FieldPath:  "metadata.annotations",
					},
				}},
			},
		},
	}
	template.Spec.Volumes = append(template.Spec.Volumes, podInfoVolume)

	var index int
	found := false
	for i, c := range template.Spec.Containers {
		if c.Name == naming.ContainerDatabase {
			index = i
			found = true
			break
		}
	}
	if !found {
		return errors.New("could not find database container to mount downstream api")
	}

	volumeMount := corev1.VolumeMount{
		Name: "podinfo",
		// This is the default value for `pgnodemx.kdapi_path` PostgreSQL
		// parameter
		// https://github.com/CrunchyData/pgnodemx#kubernetes-downwardapi-related-functions
		// https://github.com/CrunchyData/pgnodemx#configuration
		MountPath: "/etc/podinfo",
	}
	template.Spec.Containers[index].VolumeMounts = append(
		template.Spec.Containers[index].VolumeMounts,
		volumeMount)

	// add the proper label to support Pod discovery by Prometheus per pgMonitor configuration
	initialize.Labels(template)
	template.Labels[naming.LabelPGMonitorDiscovery] = "true"

	return nil
}
