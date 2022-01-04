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

package pgbouncer

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// ConfigMap populates the PgBouncer ConfigMap.
func ConfigMap(
	inCluster *v1beta1.PostgresCluster,
	outConfigMap *corev1.ConfigMap,
) {
	if inCluster.Spec.Proxy == nil || inCluster.Spec.Proxy.PGBouncer == nil {
		// PgBouncer is disabled; there is nothing to do.
		return
	}

	initialize.StringMap(&outConfigMap.Data)

	outConfigMap.Data[emptyConfigMapKey] = ""
	outConfigMap.Data[iniFileConfigMapKey] = clusterINI(inCluster)
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
	initialize.ByteMap(&outSecret.Data)

	// Use the existing password and verifier. Generate both when either is missing.
	// NOTE(cbandy): We don't have a function to compare a plaintext password
	// to a SCRAM verifier.
	password := string(inSecret.Data[passwordSecretKey])
	verifier := string(inSecret.Data[verifierSecretKey])

	if err == nil && (len(password) == 0 || len(verifier) == 0) {
		password, verifier, err = generatePassword()
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
		leaf := pki.NewLeafCertificate("", nil, nil)
		leaf.DNSNames = naming.ServiceDNSNames(ctx, inService)
		leaf.CommonName = leaf.DNSNames[0] // FQDN

		if err == nil {
			var parse error
			if data, ok := inSecret.Data[certFrontendSecretKey]; parse == nil && ok {
				leaf.Certificate, parse = pki.ParseCertificate(data)
			}
			if data, ok := inSecret.Data[certFrontendPrivateKeySecretKey]; parse == nil && ok {
				leaf.PrivateKey, parse = pki.ParsePrivateKey(data)
			}
			if parse != nil || pki.LeafCertIsBad(ctx, leaf, inRoot, inCluster.Namespace) {
				err = errors.WithStack(leaf.Generate(inRoot))
			}
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
	inCluster *v1beta1.PostgresCluster,
	inConfigMap *corev1.ConfigMap,
	inPostgreSQLCertificate *corev1.SecretProjection,
	inSecret *corev1.Secret,
	outPod *corev1.PodSpec,
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

	container := corev1.Container{
		Name: naming.ContainerPGBouncer,

		Command:         []string{"pgbouncer", iniFileAbsolutePath},
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

	outPod.Containers = []corev1.Container{container, reloader}
	outPod.Volumes = []corev1.Volume{configVolume}
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
