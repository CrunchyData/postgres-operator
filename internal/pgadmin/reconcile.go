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

package pgadmin

import (
	"bytes"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// ConfigMap populates a ConfigMap with the configuration needed to run pgAdmin.
func ConfigMap(
	inCluster *v1beta1.PostgresCluster,
	outConfigMap *corev1.ConfigMap,
) error {
	if inCluster.Spec.UserInterface == nil || inCluster.Spec.UserInterface.PGAdmin == nil {
		// pgAdmin is disabled; there is nothing to do.
		return nil
	}

	initialize.StringMap(&outConfigMap.Data)

	// To avoid spurious reconciles, the following value must not change when
	// the spec does not change. [json.Encoder] and [json.Marshal] do this by
	// emitting map keys in sorted order. Indent so the value is not rendered
	// as one long line by `kubectl`.
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(systemSettings(inCluster.Spec.UserInterface.PGAdmin))
	if err == nil {
		outConfigMap.Data[settingsConfigMapKey] = buffer.String()
	}
	return err
}

// Pod populates a PodSpec with the container and volumes needed to run pgAdmin.
func Pod(
	inCluster *v1beta1.PostgresCluster,
	inConfigMap *corev1.ConfigMap,
	outPod *corev1.PodSpec, pgAdminVolume *corev1.PersistentVolumeClaim,
) {
	if inCluster.Spec.UserInterface == nil || inCluster.Spec.UserInterface.PGAdmin == nil {
		// pgAdmin is disabled; there is nothing to do.
		return
	}

	// if a pgAdmin port is configured, use that. Otherwise, use the default
	pgAdminPort := defaultPort
	if inCluster.Spec.UserInterface.PGAdmin.Port != nil {
		pgAdminPort = int(*inCluster.Spec.UserInterface.PGAdmin.Port)
	}

	// create the pgAdmin Pod volumes
	tmp := corev1.Volume{Name: tmpVolume}
	tmp.EmptyDir = &corev1.EmptyDirVolumeSource{
		Medium: corev1.StorageMediumMemory,
	}

	pgAdminLog := corev1.Volume{Name: logVolume}
	pgAdminLog.EmptyDir = &corev1.EmptyDirVolumeSource{
		Medium: corev1.StorageMediumMemory,
	}

	pgAdminData := corev1.Volume{Name: dataVolume}
	pgAdminData.VolumeSource = corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: pgAdminVolume.Name,
			ReadOnly:  false,
		},
	}

	configVolumeMount := corev1.VolumeMount{
		Name: "pgadmin-config", MountPath: configMountPath, ReadOnly: true,
	}
	configVolume := corev1.Volume{Name: configVolumeMount.Name}
	configVolume.Projected = &corev1.ProjectedVolumeSource{
		Sources: podConfigFiles(inConfigMap),
	}

	startupVolumeMount := corev1.VolumeMount{
		Name: "pgadmin-startup", MountPath: startupMountPath, ReadOnly: true,
	}
	startupVolume := corev1.Volume{Name: startupVolumeMount.Name}
	startupVolume.EmptyDir = &corev1.EmptyDirVolumeSource{
		Medium: corev1.StorageMediumMemory,

		// When this volume is too small, the Pod will be evicted and recreated
		// by the StatefulSet controller.
		// - https://kubernetes.io/docs/concepts/storage/volumes/#emptydir
		// NOTE: tmpfs blocks are PAGE_SIZE, usually 4KiB, and size rounds up.
		SizeLimit: resource.NewQuantity(32<<10, resource.BinarySI),
	}

	// pgadmin container
	container := corev1.Container{
		Name: naming.ContainerPGAdmin,
		Env: []corev1.EnvVar{
			{
				Name:  "PGADMIN_SETUP_EMAIL",
				Value: loginEmail,
			},
			{
				Name:  "PGADMIN_SETUP_PASSWORD",
				Value: loginPassword,
			},
		},
		Image:           config.PGAdminContainerImage(inCluster),
		ImagePullPolicy: inCluster.Spec.ImagePullPolicy,
		Resources:       inCluster.Spec.UserInterface.PGAdmin.Resources,

		SecurityContext: initialize.RestrictedSecurityContext(),

		Ports: []corev1.ContainerPort{{
			Name:          naming.PortPGAdmin,
			ContainerPort: int32(pgAdminPort),
			Protocol:      corev1.ProtocolTCP,
		}},
		VolumeMounts: []corev1.VolumeMount{
			startupVolumeMount,
			configVolumeMount,
			{
				Name:      tmpVolume,
				MountPath: tmpMountPath,
			},
			{
				Name:      tmpVolume,
				MountPath: runMountPath,
			},
			{
				Name:      logVolume,
				MountPath: logMountPath,
			},
			{
				Name:      dataVolume,
				MountPath: dataMountPath,
			},
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(pgAdminPort),
				},
			},
			InitialDelaySeconds: 20,
			PeriodSeconds:       10,
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(pgAdminPort),
				},
			},
			InitialDelaySeconds: 15,
			PeriodSeconds:       20,
		},
	}

	startup := corev1.Container{
		Name:    naming.ContainerPGAdminStartup,
		Command: startupCommand(),

		Image:           container.Image,
		ImagePullPolicy: container.ImagePullPolicy,
		Resources:       container.Resources,
		SecurityContext: initialize.RestrictedSecurityContext(),
		VolumeMounts: []corev1.VolumeMount{
			startupVolumeMount,
			configVolumeMount,
		},
	}

	// The startup container is the only one allowed to write to the startup volume.
	startup.VolumeMounts[0].ReadOnly = false

	outPod.Containers = []corev1.Container{container}
	outPod.InitContainers = []corev1.Container{startup}
	outPod.Volumes = []corev1.Volume{tmp, pgAdminLog, pgAdminData, configVolume, startupVolume}
}
