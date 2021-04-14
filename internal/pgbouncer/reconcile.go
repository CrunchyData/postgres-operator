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

package pgbouncer

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

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

	verifier := inSecret.Data[credentialSecretKey]

	if err == nil && len(verifier) == 0 {
		verifier, err = generateVerifier()
	}
	if err == nil {
		outSecret.Data[authFileSecretKey] = authFileContents(verifier)
		outSecret.Data[credentialSecretKey] = verifier
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

	backend := corev1.Volume{Name: "pgbouncer-backend-tls"}
	backend.Projected = &corev1.ProjectedVolumeSource{
		Sources: []corev1.VolumeProjection{
			backendAuthority(inPostgreSQLCertificate),
		},
	}

	frontend := corev1.Volume{Name: "pgbouncer-frontend-tls"}
	frontend.Projected = &corev1.ProjectedVolumeSource{
		Sources: []corev1.VolumeProjection{
			frontendCertificate(
				inCluster.Spec.Proxy.PGBouncer.CustomTLSSecret, inSecret),
		},
	}

	config := corev1.Volume{Name: "pgbouncer-config"}
	config.Projected = &corev1.ProjectedVolumeSource{
		Sources: podConfigFiles(
			inCluster.Spec.Proxy.PGBouncer.Config, inConfigMap, inSecret),
	}

	container := corev1.Container{
		Name: naming.ContainerPGBouncer,

		Command:   []string{"pgbouncer", iniFileAbsolutePath},
		Image:     inCluster.Spec.Proxy.PGBouncer.Image,
		Resources: inCluster.Spec.Proxy.PGBouncer.Resources,

		SecurityContext: initialize.RestrictedSecurityContext(),

		Ports: []corev1.ContainerPort{{
			Name:          naming.PortPGBouncer,
			ContainerPort: *inCluster.Spec.Proxy.PGBouncer.Port,
			Protocol:      corev1.ProtocolTCP,
		}},
	}

	container.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      config.Name,
			MountPath: configDirectory,
			ReadOnly:  true,
		},
		{
			Name:      backend.Name,
			MountPath: certBackendDirectory,
			ReadOnly:  true,
		},
		{
			Name:      frontend.Name,
			MountPath: certFrontendDirectory,
			ReadOnly:  true,
		},
	}

	// TODO container.LivenessProbe?
	// TODO container.ReadinessProbe?

	reloader := corev1.Container{
		Name: naming.ContainerPGBouncerConfig,

		Command: reloadCommand(),
		Image:   inCluster.Spec.Proxy.PGBouncer.Image,

		SecurityContext: initialize.RestrictedSecurityContext(),

		VolumeMounts: []corev1.VolumeMount{{
			Name:      config.Name,
			MountPath: configDirectory,
			ReadOnly:  true,
		}},
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

	outPod.Containers = []corev1.Container{container, reloader}
	outPod.Volumes = []corev1.Volume{backend, config, frontend}
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

	outHBAs.Mandatory = append(outHBAs.Mandatory, postgresqlHBA())
}
