// Copyright 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package standalone_pgadmin

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// pod populates a PodSpec with the container and volumes needed to run pgAdmin.
func pod(
	inPGAdmin *v1beta1.PGAdmin,
	inConfigMap *corev1.ConfigMap,
	outPod *corev1.PodSpec,
	pgAdminVolume *corev1.PersistentVolumeClaim,
) {
	const (
		// config and data volume names
		configVolumeName = "standalone-pgadmin-config"
		dataVolumeName   = "standalone-pgadmin-data"
	)

	// create the pgAdmin Pod volumes
	pgAdminData := corev1.Volume{Name: dataVolumeName}
	pgAdminData.VolumeSource = corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: pgAdminVolume.Name,
			ReadOnly:  false,
		},
	}

	configVolume := corev1.Volume{Name: configVolumeName}
	configVolume.Projected = &corev1.ProjectedVolumeSource{
		Sources: podConfigFiles(inConfigMap, *inPGAdmin),
	}

	// pgadmin container
	container := corev1.Container{
		Name: naming.ContainerPGAdmin,
		// TODO(tjmoore4): Update command and image details
		Command:         []string{"bash", "-c", "while true; do echo 'Hello!'; sleep 2; done"},
		Image:           config.StandalonePGAdminContainerImage(inPGAdmin),
		ImagePullPolicy: inPGAdmin.Spec.ImagePullPolicy,
		Resources:       inPGAdmin.Spec.Resources,

		SecurityContext: initialize.RestrictedSecurityContext(),

		Ports: []corev1.ContainerPort{{
			Name:          naming.PortPGAdmin,
			ContainerPort: int32(5050),
			Protocol:      corev1.ProtocolTCP,
		}},

		Env: []corev1.EnvVar{
			{
				Name:  "PGADMIN_SETUP_EMAIL",
				Value: inPGAdmin.Spec.AdminUsername,
			},
			{
				Name: "PGADMIN_SETUP_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: naming.StandalonePGAdmin(inPGAdmin).Name,
					},
					Key: "password",
				}},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      configVolumeName,
				MountPath: "/etc/pgadmin/conf.d",
				ReadOnly:  true,
			},
			{
				Name:      dataVolumeName,
				MountPath: "/var/lib/pgadmin",
			},
		},
	}

	// add volumes and containers
	outPod.Volumes = []corev1.Volume{pgAdminData, configVolume}
	outPod.Containers = []corev1.Container{container}
}

// podConfigFiles returns projections of pgAdmin's configuration files to
// include in the configuration volume.
func podConfigFiles(configmap *corev1.ConfigMap, pgadmin v1beta1.PGAdmin) []corev1.VolumeProjection {
	config := append(append([]corev1.VolumeProjection{}, pgadmin.Spec.Config.Files...),
		[]corev1.VolumeProjection{
			{
				ConfigMap: &corev1.ConfigMapProjection{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configmap.Name,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  settingsConfigMapKey,
							Path: "~postgres-operator/pgadmin.json",
						},
					},
				},
			},
		}...)

	// To enable LDAP authentication for pgAdmin, various LDAP settings must be configured.
	// While most of the required configuration can be set using the 'settings'
	// feature on the spec (.Spec.UserInterface.PGAdmin.Config.Settings), those
	// values are stored in a ConfigMap in plaintext.
	// As a special case, here we mount a provided Secret containing the LDAP_BIND_PASSWORD
	// for use with the other pgAdmin LDAP configuration.
	// - https://www.pgadmin.org/docs/pgadmin4/latest/config_py.html
	// - https://www.pgadmin.org/docs/pgadmin4/development/enabling_ldap_authentication.html
	if pgadmin.Spec.Config.LDAPBindPassword != nil {
		config = append(config, corev1.VolumeProjection{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: pgadmin.Spec.Config.LDAPBindPassword.LocalObjectReference,
				Optional:             pgadmin.Spec.Config.LDAPBindPassword.Optional,
				Items: []corev1.KeyToPath{
					{
						Key:  pgadmin.Spec.Config.LDAPBindPassword.Key,
						Path: "~postgres-operator/ldap-bind-password",
					},
				},
			},
		})
	}

	return config
}
