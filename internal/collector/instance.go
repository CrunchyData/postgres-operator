// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/shell"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const configDirectory = "/etc/otel-collector"

// AddToConfigMap populates the shared ConfigMap with fields needed to run the Collector.
func AddToConfigMap(
	ctx context.Context,
	inConfig *Config,
	outInstanceConfigMap *corev1.ConfigMap,
) error {
	var err error
	if outInstanceConfigMap.Data == nil {
		outInstanceConfigMap.Data = make(map[string]string)
	}

	outInstanceConfigMap.Data["collector.yaml"], err = inConfig.ToYAML()

	return err
}

// AddToPod adds the OpenTelemetry collector container to a given Pod
func AddToPod(
	ctx context.Context,
	spec *v1beta1.InstrumentationSpec,
	pullPolicy corev1.PullPolicy,
	inInstanceConfigMap *corev1.ConfigMap,
	template *corev1.PodTemplateSpec,
	volumeMounts []corev1.VolumeMount,
	sqlQueryPassword string,
	logDirectories []string,
	includeLogrotate bool,
	thisPodServesMetrics bool,
) {
	if !OpenTelemetryLogsOrMetricsEnabled(ctx, spec) {
		return
	}

	// We only want to include log rotation if this type of pod requires it
	// (indicate by the includeLogrotate boolean) AND if logging is enabled
	// for this PostgresCluster/PGAdmin
	includeLogrotate = includeLogrotate && OpenTelemetryLogsEnabled(ctx, spec)

	// Create volume and volume mount for otel collector config
	configVolumeMount := corev1.VolumeMount{
		Name:      "collector-config",
		MountPath: configDirectory,
		ReadOnly:  true,
	}
	configVolume := corev1.Volume{Name: configVolumeMount.Name}
	configVolume.Projected = &corev1.ProjectedVolumeSource{
		Sources: []corev1.VolumeProjection{{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: inInstanceConfigMap.Name,
				},
				Items: []corev1.KeyToPath{{
					Key:  "collector.yaml",
					Path: "config.yaml",
				}},
			},
		}},
	}

	// If the user has specified files to be mounted in the spec, add them to
	// the projected config volume
	if spec.Config != nil && spec.Config.Files != nil {
		configVolume.Projected.Sources = append(configVolume.Projected.Sources,
			spec.Config.Files...)
	}

	// Create collector container
	container := corev1.Container{
		Name:            naming.ContainerCollector,
		Image:           config.CollectorContainerImage(spec),
		ImagePullPolicy: pullPolicy,
		Command:         startCommand(logDirectories, includeLogrotate),
		Env: []corev1.EnvVar{
			{
				Name: "K8S_POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				}},
			},
			{
				Name: "K8S_POD_NAME",
				ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				}},
			},
			{
				Name:  "PGPASSWORD",
				Value: sqlQueryPassword,
			},
		},
		Resources:       spec.Resources,
		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts:    append(volumeMounts, configVolumeMount),
	}

	// Add any user specified environment variables to the collector container
	if spec.Config != nil && spec.Config.EnvironmentVariables != nil {
		container.Env = append(container.Env, spec.Config.EnvironmentVariables...)
	}

	// If metrics feature is enabled and this Pod serves metrics, add the
	// Prometheus port to this container
	if feature.Enabled(ctx, feature.OpenTelemetryMetrics) && thisPodServesMetrics {
		container.Ports = []corev1.ContainerPort{{
			ContainerPort: int32(PrometheusPort),
			Name:          "otel-metrics",
			Protocol:      corev1.ProtocolTCP,
		}}

		// If the user has specified custom queries to add, put the queries
		// file(s) in the projected config volume
		if spec.Metrics != nil && spec.Metrics.CustomQueries != nil &&
			spec.Metrics.CustomQueries.Add != nil {
			for _, querySet := range spec.Metrics.CustomQueries.Add {
				projection := querySet.Queries.AsProjection(querySet.Name +
					"/" + querySet.Queries.Key)
				configVolume.Projected.Sources = append(configVolume.Projected.Sources,
					corev1.VolumeProjection{ConfigMap: &projection})
			}
		}
	}

	// If this is a pod that uses logrotate for log rotation, add config volume
	// and mount for logrotate config
	if includeLogrotate {
		logrotateConfigVolumeMount := corev1.VolumeMount{
			Name:      "logrotate-config",
			MountPath: "/etc/logrotate.d",
			ReadOnly:  true,
		}
		logrotateConfigVolume := corev1.Volume{Name: logrotateConfigVolumeMount.Name}
		logrotateConfigVolume.Projected = &corev1.ProjectedVolumeSource{
			Sources: []corev1.VolumeProjection{{
				ConfigMap: &corev1.ConfigMapProjection{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: inInstanceConfigMap.Name,
					},
					Items: []corev1.KeyToPath{{
						Key:  "logrotate.conf",
						Path: "logrotate.conf",
					}},
				},
			}},
		}
		container.VolumeMounts = append(container.VolumeMounts, logrotateConfigVolumeMount)
		template.Spec.Volumes = append(template.Spec.Volumes, logrotateConfigVolume)
	}

	// Add configVolume to the Pod's volumes and add the collector container to
	// the Pod's containers
	template.Spec.Volumes = append(template.Spec.Volumes, configVolume)
	template.Spec.Containers = append(template.Spec.Containers, container)

	// add the OTel collector label to the Pod
	initialize.Labels(template)
	template.Labels[naming.LabelCollectorDiscovery] = "true"
}

// startCommand generates the command script used by the collector container
func startCommand(logDirectories []string, includeLogrotate bool) []string {
	var mkdirScript string
	if len(logDirectories) != 0 {
		for _, logDir := range logDirectories {
			mkdirScript = mkdirScript + `
` + shell.MakeDirectories(logDir, "receiver")
		}
	}

	var logrotateCommand string
	if includeLogrotate {
		logrotateCommand = `logrotate -s /tmp/logrotate.status /etc/logrotate.d/logrotate.conf`
	}

	var startScript = fmt.Sprintf(`
%s
OTEL_PIDFILE=/tmp/otel.pid

start_otel_collector() {
	echo "Starting OTel Collector"
	/otelcol-contrib --config %s/config.yaml &
	echo $! > $OTEL_PIDFILE
}
start_otel_collector

exec {fd}<> <(:||:)
while read -r -t 5 -u "${fd}" ||:; do
	%s
	if [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] && kill -HUP $(head -1 ${OTEL_PIDFILE?});
	then
		echo "OTel configuration changed..."
		exec {fd}>&- && exec {fd}<> <(:||:)
		stat --format='Loaded configuration dated %%y' "${directory}"
	fi
	if [[ ! -e /proc/$(head -1 ${OTEL_PIDFILE?}) ]] ; then
		start_otel_collector
	fi
done
`, mkdirScript, configDirectory, logrotateCommand)

	wrapper := `monitor() {` + startScript +
		`}; export directory="$1"; export -f monitor; exec -a "$0" bash -ceu monitor`

	return []string{"bash", "-ceu", "--", wrapper, "collector", configDirectory}
}
