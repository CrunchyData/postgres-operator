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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// Pod populates a PodSpec with the container and volumes needed to run pgAdmin.
func Pod(
	inCluster *v1beta1.PostgresCluster,
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

	outPod.Containers = []corev1.Container{container}
	outPod.Volumes = []corev1.Volume{tmp, pgAdminLog, pgAdminData}
}
