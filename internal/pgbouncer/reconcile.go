// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbouncer

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/collector"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	passwd "github.com/crunchydata/postgres-operator/internal/postgres/password"
	"github.com/crunchydata/postgres-operator/internal/shell"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// ConfigMap populates the PgBouncer ConfigMap.
func ConfigMap(
	ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	outConfigMap *corev1.ConfigMap,
) {
	if inCluster.Spec.Proxy == nil || inCluster.Spec.Proxy.PGBouncer == nil {
		// PgBouncer is disabled; there is nothing to do.
		return
	}

	initialize.Map(&outConfigMap.Data)

	outConfigMap.Data[emptyConfigMapKey] = ""
	outConfigMap.Data[iniFileConfigMapKey] = clusterINI(ctx, inCluster)
}

// Secret populates the PgBouncer Secret.
func Secret(ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	inRoot *pki.RootCertificateAuthority,
	inSecret *corev1.Secret,
	inService *corev1.Service,
	outSecret *corev1.Secret,
) error {
	if inCluster.Spec.Proxy == nil || inCluster.Spec.Proxy.PGBouncer == nil {
		// PgBouncer is disabled; there is nothing to do.
		return nil
	}

	var err error
	initialize.Map(&outSecret.Data)

	// Use the existing password and verifier. Generate when one is missing.
	// PgBouncer can login to PostgreSQL using either MD5 or SCRAM-SHA-256.
	// When using MD5, the (hashed) verifier can be stored in PgBouncer's
	// authentication file. When using SCRAM, the plaintext password must be
	// stored.
	// - https://www.pgbouncer.org/config.html#authentication-file-format
	// - https://github.com/pgbouncer/pgbouncer/issues/508#issuecomment-713339834
	// NOTE(cbandy): We don't have a function to compare a plaintext password
	// to a SCRAM verifier.
	password := string(inSecret.Data[passwordSecretKey])
	verifier := string(inSecret.Data[verifierSecretKey])

	if len(password) == 0 {
		// If the password is empty, generate new password and verifier.
		password, err = util.GenerateASCIIPassword(32)
		err = errors.WithStack(err)
		if err == nil {
			verifier, err = passwd.NewSCRAMPassword(password).Build()
			err = errors.WithStack(err)
		}
	} else if len(password) != 0 && len(verifier) == 0 {
		// If the password is non-empty and the verifier is empty, generate a new verifier.
		verifier, err = passwd.NewSCRAMPassword(password).Build()
		err = errors.WithStack(err)
	}

	if err == nil {
		// Store the SCRAM verifier alongside the plaintext password so that
		// later reconciles don't generate it repeatedly.
		outSecret.Data[authFileSecretKey] = authFileContents(password)
		outSecret.Data[passwordSecretKey] = []byte(password)
		outSecret.Data[verifierSecretKey] = []byte(verifier)
	}

	if inCluster.Spec.Proxy.PGBouncer.CustomTLSSecret == nil {
		leaf := &pki.LeafCertificate{}
		dnsNames := naming.ServiceDNSNames(ctx, inService)
		dnsFQDN := dnsNames[0]

		if err == nil {
			// Unmarshal and validate the stored leaf. These first errors can
			// be ignored because they result in an invalid leaf which is then
			// correctly regenerated.
			_ = leaf.Certificate.UnmarshalText(inSecret.Data[certFrontendSecretKey])
			_ = leaf.PrivateKey.UnmarshalText(inSecret.Data[certFrontendPrivateKeySecretKey])

			leaf, err = inRoot.RegenerateLeafWhenNecessary(leaf, dnsFQDN, dnsNames)
			err = errors.WithStack(err)
		}

		if err == nil {
			outSecret.Data[certFrontendAuthoritySecretKey], err = inRoot.Certificate.MarshalText()
		}
		if err == nil {
			outSecret.Data[certFrontendPrivateKeySecretKey], err = leaf.PrivateKey.MarshalText()
		}
		if err == nil {
			outSecret.Data[certFrontendSecretKey], err = leaf.Certificate.MarshalText()
		}
	}

	return err
}

// Pod populates a PodSpec with the container and volumes needed to run PgBouncer.
func Pod(
	ctx context.Context,
	inCluster *v1beta1.PostgresCluster,
	inConfigMap *corev1.ConfigMap,
	inPostgreSQLCertificate *corev1.SecretProjection,
	inSecret *corev1.Secret,
	template *corev1.PodTemplateSpec,
	logfile string,
) {
	if inCluster.Spec.Proxy == nil || inCluster.Spec.Proxy.PGBouncer == nil {
		// PgBouncer is disabled; there is nothing to do.
		return
	}

	configVolumeMount := corev1.VolumeMount{
		Name: "pgbouncer-config", MountPath: configDirectory, ReadOnly: true,
	}
	configVolume := corev1.Volume{Name: configVolumeMount.Name}
	configVolume.Projected = &corev1.ProjectedVolumeSource{
		Sources: append(append([]corev1.VolumeProjection{},
			podConfigFiles(inCluster.Spec.Proxy.PGBouncer.Config, inConfigMap, inSecret)...),
			frontendCertificate(inCluster.Spec.Proxy.PGBouncer.CustomTLSSecret, inSecret),
			backendAuthority(inPostgreSQLCertificate),
		),
	}

	mkdirCommand := ""
	// filepath.Dir will return an "" when given an ""
	logPath := filepath.Dir(logfile)
	// If the logpath is `/tmp`, we don't need to worry about creating/chmoding it.
	// Otherwise, use `MakeDirectories` to create/chmod that specific directory,
	// without worrying about parent directories.
	if logfile != "" && logPath != "/tmp" {
		mkdirCommand = shell.MakeDirectories(logPath, logPath) + "; "
	}

	container := corev1.Container{
		Name: naming.ContainerPGBouncer,

		Command:         []string{"sh", "-c", "--", mkdirCommand + `exec "$@"`, "--", "pgbouncer", iniFileAbsolutePath},
		Image:           config.PGBouncerContainerImage(inCluster),
		ImagePullPolicy: inCluster.Spec.ImagePullPolicy,
		Resources:       inCluster.Spec.Proxy.PGBouncer.Resources,
		SecurityContext: initialize.RestrictedSecurityContext(),

		Ports: []corev1.ContainerPort{{
			Name:          naming.PortPGBouncer,
			ContainerPort: *inCluster.Spec.Proxy.PGBouncer.Port,
			Protocol:      corev1.ProtocolTCP,
		}},

		VolumeMounts: []corev1.VolumeMount{configVolumeMount},
	}

	// TODO container.LivenessProbe?
	// TODO container.ReadinessProbe?

	reloader := corev1.Container{
		Name: naming.ContainerPGBouncerConfig,

		Command:         reloadCommand(naming.ContainerPGBouncerConfig),
		Image:           container.Image,
		ImagePullPolicy: container.ImagePullPolicy,
		SecurityContext: initialize.RestrictedSecurityContext(),

		VolumeMounts: []corev1.VolumeMount{configVolumeMount},
	}

	// Let the PgBouncer container drive the QoS of the pod. Set resources only
	// when that container has some.
	// - https://docs.k8s.io/tasks/configure-pod-container/quality-service-pod/
	if len(container.Resources.Limits)+len(container.Resources.Requests) > 0 {
		// Limits without Requests implies Requests that match.
		reloader.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("5m"),
			corev1.ResourceMemory: resource.MustParse("16Mi"),
		}
	}

	// When resources are explicitly set, overwrite the above.
	if inCluster.Spec.Proxy.PGBouncer.Sidecars != nil &&
		inCluster.Spec.Proxy.PGBouncer.Sidecars.PGBouncerConfig != nil &&
		inCluster.Spec.Proxy.PGBouncer.Sidecars.PGBouncerConfig.Resources != nil {
		reloader.Resources = *inCluster.Spec.Proxy.PGBouncer.Sidecars.PGBouncerConfig.Resources
	}

	template.Spec.Containers = []corev1.Container{container, reloader}

	// If the PGBouncerSidecars feature gate is enabled and custom pgBouncer
	// sidecars are defined, add the defined container to the Pod.
	if feature.Enabled(ctx, feature.PGBouncerSidecars) &&
		inCluster.Spec.Proxy.PGBouncer.Containers != nil {
		template.Spec.Containers = append(template.Spec.Containers, inCluster.Spec.Proxy.PGBouncer.Containers...)
	}

	template.Spec.Volumes = []corev1.Volume{configVolume}

	if collector.OpenTelemetryLogsOrMetricsEnabled(ctx, inCluster) {
		collector.AddToPod(ctx, inCluster.Spec.Instrumentation, inCluster.Spec.ImagePullPolicy, inConfigMap,
			template, []corev1.VolumeMount{configVolumeMount}, string(inSecret.Data["pgbouncer-password"]),
			[]string{naming.PGBouncerLogPath}, true, true)
	}
}

// PostgreSQL populates outHBAs with any records needed to run PgBouncer.
func PostgreSQL(
	inCluster *v1beta1.PostgresCluster,
	outHBAs *postgres.HBAs,
) {
	if inCluster.Spec.Proxy == nil || inCluster.Spec.Proxy.PGBouncer == nil {
		// PgBouncer is disabled; there is nothing to do.
		return
	}

	outHBAs.Mandatory = append(outHBAs.Mandatory, postgresqlHBAs()...)
}
