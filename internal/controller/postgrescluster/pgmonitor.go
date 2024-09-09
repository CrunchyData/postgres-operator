// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgmonitor"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
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
// - PodExec to run the sql in the primary database
// Status.Monitoring.ExporterConfiguration is used to determine when the
// pgMonitor postgres_exporter configuration should be added/changed to
// limit how often PodExec is used
// - TODO (jmckulk): kube perms comment?
func (r *Reconciler) reconcilePGMonitorExporter(ctx context.Context,
	cluster *v1beta1.PostgresCluster, instances *observedInstances,
	monitoringSecret *corev1.Secret) error {

	var (
		writableInstance *Instance
		writablePod      *corev1.Pod
		setup            string
		pgImageSHA       string
	)

	// Find the PostgreSQL instance that can execute SQL that writes to every
	// database. When there is none, return early.
	writablePod, writableInstance = instances.writablePod(naming.ContainerDatabase)
	if writableInstance == nil || writablePod == nil {
		return nil
	}

	// For the writableInstance found above
	// 1) get and save the imageIDs for `database` container, and
	// 2) exit early if we can't get the ImageID of this container.
	// We use this ImageID and the setup.sql file in the hash we make to see if the operator needs to rerun
	// the `EnableExporterInPostgreSQL` funcs; that way we are always running
	// that function against an updated and running pod.
	if pgmonitor.ExporterEnabled(cluster) {
		sql, err := os.ReadFile(fmt.Sprintf("%s/pg%d/setup.sql", pgmonitor.GetQueriesConfigDir(ctx), cluster.Spec.PostgresVersion))
		if err != nil {
			return err
		}

		// TODO: Revisit how pgbackrest_info.sh is used with pgMonitor.
		// pgMonitor queries expect a path to a script that runs pgBackRest
		// info and provides json output. In the queries yaml for pgBackRest
		// the default path is `/usr/bin/pgbackrest-info.sh`. We update
		// the path to point to the script in our database image.
		setup = strings.ReplaceAll(string(sql), "/usr/bin/pgbackrest-info.sh",
			"/opt/crunchy/bin/postgres/pgbackrest_info.sh")

		for _, containerStatus := range writablePod.Status.ContainerStatuses {
			if containerStatus.Name == naming.ContainerDatabase {
				pgImageSHA = containerStatus.ImageID
			}
		}

		// Could not get container imageID
		if pgImageSHA == "" {
			return nil
		}
	}

	// PostgreSQL is available for writes. Prepare to either add or remove
	// pgMonitor objects.

	action := func(ctx context.Context, exec postgres.Executor) error {
		return pgmonitor.EnableExporterInPostgreSQL(ctx, exec, monitoringSecret, pgmonitor.ExporterDB, setup)
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
				// Use command and image tag in hash to execute hash on image update
				_, err = fmt.Fprint(hasher, command, pgImageSHA, setup)
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

		// Apply the necessary SQL and record its hash in cluster.Status
		if err == nil {
			err = action(ctx, func(ctx context.Context, stdin io.Reader,
				stdout, stderr io.Writer, command ...string) error {
				return r.PodExec(ctx, writablePod.Namespace, writablePod.Name, naming.ContainerDatabase, stdin, stdout, stderr, command...)
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

	// Copy existing password and verifier into the intent
	if existing.Data != nil {
		intent.Data["password"] = existing.Data["password"]
		intent.Data["verifier"] = existing.Data["verifier"]
	}

	// When password is unset, generate a new one
	if len(intent.Data["password"]) == 0 {
		password, err := util.GenerateASCIIPassword(util.DefaultGeneratedPasswordLength)
		if err != nil {
			return nil, err
		}
		intent.Data["password"] = []byte(password)
		// We generated a new password, unset the verifier so that it is regenerated
		intent.Data["verifier"] = nil
	}

	// When a password has been generated or the verifier is empty,
	// generate a verifier based on the current password.
	// NOTE(cbandy): We don't have a function to compare a plaintext
	// password to a SCRAM verifier.
	if len(intent.Data["verifier"]) == 0 {
		verifier, err := pgpassword.NewSCRAMPassword(string(intent.Data["password"])).Build()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		intent.Data["verifier"] = []byte(verifier)
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
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	template *corev1.PodTemplateSpec,
	exporterQueriesConfig, exporterWebConfig *corev1.ConfigMap) error {

	err := addPGMonitorExporterToInstancePodSpec(ctx, cluster, template, exporterQueriesConfig, exporterWebConfig)

	return err
}

// addPGMonitorExporterToInstancePodSpec performs the necessary setup to
// add pgMonitor exporter resources to a PodTemplateSpec
// TODO (jmckulk): refactor to pass around monitoring secret; Without the secret
// the exporter container cannot be created; Testing relies on ensuring the
// monitoring secret is available
func addPGMonitorExporterToInstancePodSpec(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	template *corev1.PodTemplateSpec,
	exporterQueriesConfig, exporterWebConfig *corev1.ConfigMap) error {

	if !pgmonitor.ExporterEnabled(cluster) {
		return nil
	}

	certSecret := cluster.Spec.Monitoring.PGMonitor.Exporter.CustomTLSSecret
	withBuiltInCollectors :=
		!strings.EqualFold(cluster.Annotations[naming.PostgresExporterCollectorsAnnotation], "None")

	securityContext := initialize.RestrictedSecurityContext()
	exporterContainer := corev1.Container{
		Name:            naming.ContainerPGMonitorExporter,
		Image:           config.PGExporterContainerImage(cluster),
		ImagePullPolicy: cluster.Spec.ImagePullPolicy,
		Resources:       cluster.Spec.Monitoring.PGMonitor.Exporter.Resources,
		Command:         pgmonitor.ExporterStartCommand(withBuiltInCollectors),
		Env: []corev1.EnvVar{
			{Name: "DATA_SOURCE_URI", Value: fmt.Sprintf("%s:%d/%s", pgmonitor.ExporterHost, *cluster.Spec.Port, pgmonitor.ExporterDB)},
			{Name: "DATA_SOURCE_USER", Value: pgmonitor.MonitoringUser},
			{Name: "DATA_SOURCE_PASS_FILE", Value: "/opt/crunchy/password"},
		},
		SecurityContext: securityContext,
		// ContainerPort is needed to support proper target discovery by Prometheus for pgMonitor
		// integration
		Ports: []corev1.ContainerPort{{
			ContainerPort: pgmonitor.ExporterPort,
			Name:          naming.PortExporter,
			Protocol:      corev1.ProtocolTCP,
		}},
		VolumeMounts: []corev1.VolumeMount{{
			Name: "exporter-config",
			// this is the path for both custom and default queries files
			MountPath: "/conf",
		}, {
			Name:      "monitoring-secret",
			MountPath: "/opt/crunchy/",
		}},
	}

	passwordVolume := corev1.Volume{
		Name: "monitoring-secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: naming.MonitoringUserSecret(cluster).Name,
			},
		},
	}

	// add custom exporter config volume
	configVolume := corev1.Volume{
		Name: "exporter-config",
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: cluster.Spec.Monitoring.PGMonitor.Exporter.Configuration,
			},
		},
	}
	template.Spec.Volumes = append(template.Spec.Volumes, configVolume, passwordVolume)

	// The original "custom queries" ability allowed users to provide a file with custom queries;
	// however, it would turn off the default queries. The new "custom queries" ability allows
	// users to append custom queries to the default queries. This new behavior is feature gated.
	// Therefore, we only want to add the default queries ConfigMap as a source for the
	// "exporter-config" volume if the AppendCustomQueries feature gate is turned on OR if the
	// user has not provided any custom configuration.
	if feature.Enabled(ctx, feature.AppendCustomQueries) ||
		cluster.Spec.Monitoring.PGMonitor.Exporter.Configuration == nil {

		defaultConfigVolumeProjection := corev1.VolumeProjection{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: exporterQueriesConfig.Name,
				},
			},
		}
		configVolume.VolumeSource.Projected.Sources = append(configVolume.VolumeSource.Projected.Sources,
			defaultConfigVolumeProjection)
	}

	if certSecret != nil {
		// TODO (jmckulk): params for paths and such
		certVolume := corev1.Volume{Name: "exporter-certs"}
		certVolume.Projected = &corev1.ProjectedVolumeSource{
			Sources: append([]corev1.VolumeProjection{},
				corev1.VolumeProjection{
					Secret: certSecret,
				},
			),
		}

		webConfigVolume := corev1.Volume{Name: "web-config"}
		webConfigVolume.ConfigMap = &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: exporterWebConfig.Name,
			},
		}
		template.Spec.Volumes = append(template.Spec.Volumes, certVolume, webConfigVolume)

		mounts := []corev1.VolumeMount{{
			Name:      "exporter-certs",
			MountPath: "/certs",
		}, {
			Name:      "web-config",
			MountPath: "/web-config",
		}}

		exporterContainer.VolumeMounts = append(exporterContainer.VolumeMounts, mounts...)
		exporterContainer.Command = pgmonitor.ExporterStartCommand(
			withBuiltInCollectors, pgmonitor.ExporterWebConfigFileFlag)
	}

	template.Spec.Containers = append(template.Spec.Containers, exporterContainer)

	// add the proper label to support Pod discovery by Prometheus per pgMonitor configuration
	initialize.Labels(template)
	template.Labels[naming.LabelPGMonitorDiscovery] = "true"

	return nil
}

// reconcileExporterWebConfig reconciles the configmap containing the webconfig for exporter tls
func (r *Reconciler) reconcileExporterWebConfig(ctx context.Context,
	cluster *v1beta1.PostgresCluster) (*corev1.ConfigMap, error) {

	existing := &corev1.ConfigMap{ObjectMeta: naming.ExporterWebConfigMap(cluster)}
	err := errors.WithStack(r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing))
	if client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	if !pgmonitor.ExporterEnabled(cluster) || cluster.Spec.Monitoring.PGMonitor.Exporter.CustomTLSSecret == nil {
		// We could still have a NotFound error here so check the err.
		// If no error that means the configmap is found and needs to be deleted
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, existing))
		}
		return nil, client.IgnoreNotFound(err)
	}

	intent := &corev1.ConfigMap{
		ObjectMeta: naming.ExporterWebConfigMap(cluster),
		Data: map[string]string{
			"web-config.yml": `
# Generated by postgres-operator. DO NOT EDIT.
# Your changes will not be saved.


# A certificate and a key file are needed to enable TLS.
tls_server_config:
  cert_file: /certs/tls.crt
  key_file: /certs/tls.key`,
		},
	}

	intent.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
	)
	intent.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RoleMonitoring,
		})

	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	err = errors.WithStack(r.setControllerReference(cluster, intent))
	if err == nil {
		err = errors.WithStack(r.apply(ctx, intent))
	}
	if err == nil {
		return intent, nil
	}

	return nil, err
}

// reconcileExporterQueriesConfig reconciles the configmap containing the default queries for exporter
func (r *Reconciler) reconcileExporterQueriesConfig(ctx context.Context,
	cluster *v1beta1.PostgresCluster) (*corev1.ConfigMap, error) {

	existing := &corev1.ConfigMap{ObjectMeta: naming.ExporterQueriesConfigMap(cluster)}
	err := errors.WithStack(r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing))
	if client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	if !pgmonitor.ExporterEnabled(cluster) {
		// We could still have a NotFound error here so check the err.
		// If no error that means the configmap is found and needs to be deleted
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, existing))
		}
		return nil, client.IgnoreNotFound(err)
	}

	intent := &corev1.ConfigMap{
		ObjectMeta: naming.ExporterQueriesConfigMap(cluster),
		Data:       map[string]string{"defaultQueries.yml": pgmonitor.GenerateDefaultExporterQueries(ctx, cluster)},
	}

	intent.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
	)
	intent.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RoleMonitoring,
		})

	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	err = errors.WithStack(r.setControllerReference(cluster, intent))
	if err == nil {
		err = errors.WithStack(r.apply(ctx, intent))
	}
	if err == nil {
		return intent, nil
	}

	return nil, err
}
