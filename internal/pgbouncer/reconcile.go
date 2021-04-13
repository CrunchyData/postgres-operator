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
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/naming"
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

	if outConfigMap.Data == nil {
		outConfigMap.Data = make(map[string]string)
	}

	outConfigMap.Data[iniFileConfigMapKey] = clusterINI(inCluster)
}

// Secret populates the PgBouncer Secret.
func Secret(
	inSecret *corev1.Secret,
	outSecret *corev1.Secret,
) error {
	var err error
	if outSecret.Data == nil {
		outSecret.Data = make(map[string][]byte)
	}

	verifier := inSecret.Data[credentialSecretKey]

	if err == nil && len(verifier) == 0 {
		verifier, err = generateVerifier()
	}
	if err == nil {
		outSecret.Data[authFileSecretKey] = authFileContents(verifier)
		outSecret.Data[credentialSecretKey] = verifier
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

	config := corev1.Volume{Name: "pgbouncer-config"}
	config.Projected = new(corev1.ProjectedVolumeSource)

	// Add our projections after those specified in the CR. Items later in the
	// list take precedence over earlier items (that is, last write wins).
	// - https://docs.k8s.io/concepts/storage/volumes/#projected
	config.Projected.Sources = append(append(
		// TODO(cbandy): User config will come from the spec.
		config.Projected.Sources, []corev1.VolumeProjection(nil)...),
		podConfigFiles(inConfigMap, inSecret)...)

	container := corev1.Container{
		Name: naming.ContainerPGBouncer,

		Command:   []string{"pgbouncer", iniFileAbsolutePath},
		Image:     inCluster.Spec.Proxy.PGBouncer.Image,
		Resources: inCluster.Spec.Proxy.PGBouncer.Resources,

		Ports: []corev1.ContainerPort{{
			Name:          naming.PortPGBouncer,
			ContainerPort: *inCluster.Spec.Proxy.PGBouncer.Port,
			Protocol:      corev1.ProtocolTCP,
		}},
	}

	False := false
	True := true
	container.SecurityContext = &corev1.SecurityContext{
		AllowPrivilegeEscalation: &False,
		Privileged:               &False,
		ReadOnlyRootFilesystem:   &True,
		RunAsNonRoot:             &True,
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
	}

	// TODO container.LivenessProbe?
	// TODO container.ReadinessProbe?

	outPod.Containers = []corev1.Container{container}
	outPod.Volumes = []corev1.Volume{backend, config}
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
