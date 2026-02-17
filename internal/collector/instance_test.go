// Copyright 2024 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestAddToPod(t *testing.T) {
	// setupContext creates a context with the given feature gates enabled.
	setupContext := func(logs, metrics bool) context.Context {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.OpenTelemetryLogs:    logs,
			feature.OpenTelemetryMetrics: metrics,
		}))
		return feature.NewContext(context.Background(), gate)
	}

	// newPodTemplate returns a minimal PodTemplateSpec for testing.
	newPodTemplate := func() *corev1.PodTemplateSpec {
		return &corev1.PodTemplateSpec{}
	}

	// newConfigMap returns a ConfigMap with a name, as used by callers.
	newConfigMap := func() *corev1.ConfigMap {
		cm := &corev1.ConfigMap{}
		cm.Name = "test-configmap"
		return cm
	}

	// findContainer returns the collector container from the template, if present.
	findContainer := func(template *corev1.PodTemplateSpec) *corev1.Container {
		for i := range template.Spec.Containers {
			if template.Spec.Containers[i].Name == naming.ContainerCollector {
				return &template.Spec.Containers[i]
			}
		}
		return nil
	}

	// findVolume returns the named volume from the template, if present.
	findVolume := func(template *corev1.PodTemplateSpec, name string) *corev1.Volume {
		for i := range template.Spec.Volumes {
			if template.Spec.Volumes[i].Name == name {
				return &template.Spec.Volumes[i]
			}
		}
		return nil
	}

	t.Run("NoOpWhenFeaturesDisabled", func(t *testing.T) {
		ctx := setupContext(false, false)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()

		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, false, false)

		assert.Assert(t, cmp.Len(template.Spec.Containers, 0),
			"expected no containers when features are disabled")
		assert.Assert(t, cmp.Len(template.Spec.Volumes, 0),
			"expected no volumes when features are disabled")
	})

	t.Run("NoOpWhenSpecIsNil", func(t *testing.T) {
		ctx := setupContext(true, true)
		template := newPodTemplate()

		AddToPod(ctx, nil, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, false, false)

		assert.Assert(t, cmp.Len(template.Spec.Containers, 0),
			"expected no containers when spec is nil")
	})

	t.Run("CollectorContainerAndLabel", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()
		cm := newConfigMap()

		AddToPod(ctx, spec, corev1.PullAlways, cm,
			template, nil, "test-password", []string{"/pgdata/logs"}, false, false)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")
		assert.Equal(t, container.Name, naming.ContainerCollector)
		assert.Equal(t, container.ImagePullPolicy, corev1.PullAlways)

		assert.Equal(t, template.Labels[naming.LabelCollectorDiscovery], "true")
	})

	t.Run("DefaultEnvVars", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()

		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "my-password", nil, false, false)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		// Verify the three default env vars are present
		assert.Assert(t, cmp.Len(container.Env, 3), "expected 3 environment variables")
		assert.Equal(t, container.Env[0].Name, "K8S_POD_NAMESPACE")
		assert.Equal(t, container.Env[0].ValueFrom.FieldRef.FieldPath, "metadata.namespace")
		assert.Equal(t, container.Env[1].Name, "K8S_POD_NAME")
		assert.Equal(t, container.Env[1].ValueFrom.FieldRef.FieldPath, "metadata.name")
		assert.Equal(t, container.Env[2].Name, "PGPASSWORD")
		assert.Equal(t, container.Env[2].Value, "my-password")
	})

	t.Run("EnvironmentVariables", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{
			Config: &v1beta1.InstrumentationConfigSpec{
				EnvironmentVariables: []corev1.EnvVar{
					{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/etc/otel-collector/gcp-sa.json"},
					{Name: "CUSTOM_VAR", Value: "custom-value"},
				},
			},
		}
		template := newPodTemplate()

		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, false, false)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		// The three default env vars plus two user-specified ones
		assert.Assert(t, cmp.Len(container.Env, 5), "expected 5 environment variables")

		// Default env vars come first
		assert.Equal(t, container.Env[0].Name, "K8S_POD_NAMESPACE")
		assert.Equal(t, container.Env[1].Name, "K8S_POD_NAME")
		assert.Equal(t, container.Env[2].Name, "PGPASSWORD")

		// User env vars are appended
		assert.Equal(t, container.Env[3].Name, "GOOGLE_APPLICATION_CREDENTIALS")
		assert.Equal(t, container.Env[3].Value, "/etc/otel-collector/gcp-sa.json")
		assert.Equal(t, container.Env[4].Name, "CUSTOM_VAR")
		assert.Equal(t, container.Env[4].Value, "custom-value")
	})

	t.Run("FilesNilConfig", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()
		cm := newConfigMap()

		AddToPod(ctx, spec, corev1.PullIfNotPresent, cm,
			template, nil, "", nil, false, false)

		configVolume := findVolume(template, "collector-config")
		assert.Assert(t, configVolume != nil, "expected collector-config volume")
		assert.Assert(t, configVolume.Projected != nil)

		// Only the operator's config map projection
		assert.Assert(t, cmp.Len(configVolume.Projected.Sources, 1), "expected 1 source")
		assert.Assert(t, configVolume.Projected.Sources[0].ConfigMap != nil, "expected config map source")
		assert.Equal(t, configVolume.Projected.Sources[0].ConfigMap.Name, cm.Name)
	})

	t.Run("FilesConfiguredInSpec", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{
			Config: &v1beta1.InstrumentationConfigSpec{
				Files: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "gcp-credentials",
							},
						},
					},
					{
						ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "extra-config",
							},
						},
					},
				},
			},
		}
		template := newPodTemplate()
		cm := newConfigMap()

		AddToPod(ctx, spec, corev1.PullIfNotPresent, cm,
			template, nil, "", nil, false, false)

		configVolume := findVolume(template, "collector-config")
		assert.Assert(t, configVolume != nil, "expected collector-config volume")
		assert.Assert(t, configVolume.Projected != nil)

		// First source is always the operator's config map
		assert.Assert(t, cmp.Len(configVolume.Projected.Sources, 3), "expected 3 sources")
		assert.Assert(t, configVolume.Projected.Sources[0].ConfigMap != nil, "expected config map source")
		assert.Equal(t, configVolume.Projected.Sources[0].ConfigMap.Name, cm.Name)

		// User-specified files are appended
		assert.Assert(t, configVolume.Projected.Sources[1].Secret != nil, "expected secret source")
		assert.Equal(t, configVolume.Projected.Sources[1].Secret.Name, "gcp-credentials")
		assert.Assert(t, configVolume.Projected.Sources[2].ConfigMap != nil, "expected config map source")
		assert.Equal(t, configVolume.Projected.Sources[2].ConfigMap.Name, "extra-config")
	})

	t.Run("ConfigVolumeMount", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()

		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, false, false)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		// The config volume mount should be present and read-only
		var configMount *corev1.VolumeMount
		for i := range container.VolumeMounts {
			if container.VolumeMounts[i].Name == "collector-config" {
				configMount = &container.VolumeMounts[i]
				break
			}
		}
		assert.Assert(t, configMount != nil, "expected collector-config volume mount")
		assert.Equal(t, configMount.MountPath, configDirectory)
		assert.Assert(t, configMount.ReadOnly, "expected read-only volume mount")
	})

	t.Run("AdditionalVolumeMountsIncluded", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()

		extraMounts := []corev1.VolumeMount{
			{Name: "a-volume-mount", MountPath: "/here"},
			{Name: "another-volume-mount", MountPath: "/there"},
		}

		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, extraMounts, "", nil, false, false)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		// Extra mounts plus the config volume mount
		assert.Assert(t, cmp.Len(container.VolumeMounts, 3), "expected 3 volume mounts")
		assert.Equal(t, container.VolumeMounts[0].Name, "a-volume-mount")
		assert.Equal(t, container.VolumeMounts[0].MountPath, "/here")
		assert.Equal(t, container.VolumeMounts[1].Name, "another-volume-mount")
		assert.Equal(t, container.VolumeMounts[1].MountPath, "/there")
		assert.Equal(t, container.VolumeMounts[2].Name, "collector-config")
		assert.Equal(t, container.VolumeMounts[2].MountPath, configDirectory)
	})

	t.Run("MetricsPortAdded", func(t *testing.T) {
		ctx := setupContext(false, true)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()

		// thisPodServesMetrics=true
		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, false, true)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		assert.Assert(t, cmp.Len(container.Ports, 1), "expected 1 port")
		assert.Equal(t, container.Ports[0].ContainerPort, int32(PrometheusPort))
		assert.Equal(t, container.Ports[0].Name, "otel-metrics")
		assert.Equal(t, container.Ports[0].Protocol, corev1.ProtocolTCP)
	})

	t.Run("MetricsPortNotAddedWhenNotServingMetrics", func(t *testing.T) {
		ctx := setupContext(false, true)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()

		// thisPodServesMetrics=false
		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, false, false)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		assert.Assert(t, cmp.Len(container.Ports, 0), "expected no ports")
	})

	t.Run("MetricsPortNotAddedWhenMetricsDisabled", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()

		// thisPodServesMetrics=true, but metrics feature is disabled
		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, false, true)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		assert.Assert(t, cmp.Len(container.Ports, 0), "expected no ports")
	})

	t.Run("CustomQueriesAddedToConfigVolume", func(t *testing.T) {
		ctx := setupContext(false, true)
		spec := &v1beta1.InstrumentationSpec{
			Metrics: &v1beta1.InstrumentationMetricsSpec{
				CustomQueries: &v1beta1.InstrumentationCustomQueriesSpec{
					Add: []v1beta1.InstrumentationCustomQueries{
						{
							Name: "slow-queries",
							Queries: v1beta1.ConfigMapKeyRef{
								Name: "my-custom-queries",
								Key:  "slow.yaml",
							},
						},
						{
							Name: "fast-queries",
							Queries: v1beta1.ConfigMapKeyRef{
								Name: "my-custom-queries",
								Key:  "fast.yaml",
							},
						},
					},
				},
			},
		}
		template := newPodTemplate()
		cm := newConfigMap()

		AddToPod(ctx, spec, corev1.PullIfNotPresent, cm,
			template, nil, "", nil, false, true)

		configVolume := findVolume(template, "collector-config")
		assert.Assert(t, configVolume != nil, "expected collector-config volume")
		assert.Assert(t, configVolume.Projected != nil, "expected projected volume")

		// First source is the operator config map, then the two custom query projections
		assert.Assert(t, cmp.Len(configVolume.Projected.Sources, 3), "expected 3 sources")

		// Operator config map
		assert.Assert(t, configVolume.Projected.Sources[0].ConfigMap != nil, "expected config map source")
		assert.Equal(t, configVolume.Projected.Sources[0].ConfigMap.Name, cm.Name)

		// Custom query projections
		assert.Assert(t, configVolume.Projected.Sources[1].ConfigMap != nil, "expected config map source")
		assert.Equal(t, configVolume.Projected.Sources[1].ConfigMap.Name, "my-custom-queries")
		assert.Assert(t, cmp.Len(configVolume.Projected.Sources[1].ConfigMap.Items, 1), "expected 1 item")
		assert.Equal(t, configVolume.Projected.Sources[1].ConfigMap.Items[0].Key, "slow.yaml")
		assert.Equal(t, configVolume.Projected.Sources[1].ConfigMap.Items[0].Path, "slow-queries/slow.yaml")

		assert.Assert(t, configVolume.Projected.Sources[2].ConfigMap != nil, "expected config map source")
		assert.Equal(t, configVolume.Projected.Sources[2].ConfigMap.Name, "my-custom-queries")
		assert.Assert(t, cmp.Len(configVolume.Projected.Sources[2].ConfigMap.Items, 1), "expected 1 item")
		assert.Equal(t, configVolume.Projected.Sources[2].ConfigMap.Items[0].Key, "fast.yaml")
		assert.Equal(t, configVolume.Projected.Sources[2].ConfigMap.Items[0].Path, "fast-queries/fast.yaml")
	})

	t.Run("CustomQueriesNotAddedWhenMetricsDisabled", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{
			Metrics: &v1beta1.InstrumentationMetricsSpec{
				CustomQueries: &v1beta1.InstrumentationCustomQueriesSpec{
					Add: []v1beta1.InstrumentationCustomQueries{
						{
							Name: "slow-queries",
							Queries: v1beta1.ConfigMapKeyRef{
								Name: "my-custom-queries",
								Key:  "slow.yaml",
							},
						},
					},
				},
			},
		}
		template := newPodTemplate()

		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, false, true)

		configVolume := findVolume(template, "collector-config")
		assert.Assert(t, configVolume != nil, "expected collector-config volume")
		assert.Assert(t, configVolume.Projected != nil, "expected projected volume")

		// Only the operator config map, no custom query projections
		assert.Assert(t, cmp.Len(configVolume.Projected.Sources, 1), "expected 1 source")
	})

	t.Run("LogrotateNotAddedWhenNotRequested", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()

		// includeLogrotate=false, but logs feature is enabled
		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, false, false)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		for _, vm := range container.VolumeMounts {
			assert.Assert(t, vm.Name != "logrotate-config",
				"logrotate volume mount should not be present when not requested")
		}
		assert.Assert(t, findVolume(template, "logrotate-config") == nil)
	})

	t.Run("LogrotateNotAddedWhenLogsDisabled", func(t *testing.T) {
		ctx := setupContext(false, true)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()

		// includeLogrotate=true, but logs feature is disabled
		AddToPod(ctx, spec, corev1.PullIfNotPresent, newConfigMap(),
			template, nil, "", nil, true, false)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		// No logrotate mount should be present
		for _, vm := range container.VolumeMounts {
			assert.Assert(t, vm.Name != "logrotate-config",
				"logrotate volume mount should not be present when logs are disabled")
		}

		// No logrotate volume should be present
		assert.Assert(t, findVolume(template, "logrotate-config") == nil,
			"logrotate volume should not be present when logs are disabled")
	})

	t.Run("LogrotateVolumeAndMount", func(t *testing.T) {
		ctx := setupContext(true, false)
		spec := &v1beta1.InstrumentationSpec{}
		template := newPodTemplate()
		cm := newConfigMap()

		// includeLogrotate=true
		AddToPod(ctx, spec, corev1.PullIfNotPresent, cm,
			template, nil, "", nil, true, false)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		// The logrotate config volume mount should be present
		var logrotateMount *corev1.VolumeMount
		for i := range container.VolumeMounts {
			if container.VolumeMounts[i].Name == "logrotate-config" {
				logrotateMount = &container.VolumeMounts[i]
				break
			}
		}
		assert.Assert(t, logrotateMount != nil, "expected logrotate-config volume mount")
		assert.Equal(t, logrotateMount.MountPath, "/etc/logrotate.d")
		assert.Assert(t, logrotateMount.ReadOnly, "expected read-only volume mount")

		// The logrotate volume should be present with the config map projection
		logrotateVolume := findVolume(template, "logrotate-config")
		assert.Assert(t, logrotateVolume != nil, "expected logrotate-config volume")
		assert.Assert(t, logrotateVolume.Projected != nil, "expected projected volume")
		assert.Assert(t, cmp.Len(logrotateVolume.Projected.Sources, 1), "expected 1 source")
		assert.Assert(t, logrotateVolume.Projected.Sources[0].ConfigMap != nil, "expected config map source")
		assert.Equal(t, logrotateVolume.Projected.Sources[0].ConfigMap.Name, cm.Name)
		assert.Assert(t, cmp.Len(logrotateVolume.Projected.Sources[0].ConfigMap.Items, 1), "expected 1 item")
		assert.Equal(t, logrotateVolume.Projected.Sources[0].ConfigMap.Items[0].Key, "logrotate.conf")
		assert.Equal(t, logrotateVolume.Projected.Sources[0].ConfigMap.Items[0].Path, "logrotate.conf")
	})

	t.Run("FilesAndEnvironmentVariablesTogether", func(t *testing.T) {
		ctx := setupContext(true, true)
		spec := &v1beta1.InstrumentationSpec{
			Config: &v1beta1.InstrumentationConfigSpec{
				Files: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "gcp-sa-secret",
							},
						},
					},
				},
				EnvironmentVariables: []corev1.EnvVar{
					{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/etc/otel-collector/gcp-sa.json"},
				},
			},
		}
		template := newPodTemplate()
		cm := newConfigMap()

		AddToPod(ctx, spec, corev1.PullIfNotPresent, cm,
			template, nil, "", nil, false, true)

		container := findContainer(template)
		assert.Assert(t, container != nil, "expected collector container")

		// Default env vars + 1 user env var
		assert.Assert(t, cmp.Len(container.Env, 4), "expected 4 environment variables")
		assert.Equal(t, container.Env[3].Name, "GOOGLE_APPLICATION_CREDENTIALS")
		assert.Equal(t, container.Env[3].Value, "/etc/otel-collector/gcp-sa.json")

		// Config map + 1 secret file
		configVolume := findVolume(template, "collector-config")
		assert.Assert(t, configVolume != nil, "expected collector-config volume")
		assert.Assert(t, cmp.Len(configVolume.Projected.Sources, 2), "expected 2 sources")
		assert.Assert(t, configVolume.Projected.Sources[0].ConfigMap != nil, "expected config map source")
		assert.Assert(t, configVolume.Projected.Sources[1].Secret != nil, "expected secret source")
		assert.Equal(t, configVolume.Projected.Sources[1].Secret.Name, "gcp-sa-secret")
	})
}
