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

package postgres

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestCopyClientTLS(t *testing.T) {

	postgresCluster := &v1beta1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}
	template := &v1.PodTemplateSpec{}

	InitCopyReplicationTLS(postgresCluster, template)

	var foundPGDATAInitContainer bool
	for _, c := range template.Spec.InitContainers {
		if c.Name == naming.ContainerClientCertInit {
			foundPGDATAInitContainer = true
			break
		}
	}

	assert.Assert(t, foundPGDATAInitContainer)
}

func TestAddCertVolumeToPod(t *testing.T) {

	postgresCluster := &v1beta1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}
	template := &v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name: "database",
			}, {
				Name: "replication-cert-copy",
			}},
			InitContainers: []v1.Container{{
				Name: "database-client-cert-init",
			},
			},
		},
	}
	mode := int32(0o600)
	// example auto-generated secret projection
	testServerSecretProjection := &v1.SecretProjection{
		LocalObjectReference: v1.LocalObjectReference{
			Name: naming.PostgresTLSSecret(postgresCluster).Name,
		},
		Items: []v1.KeyToPath{
			{
				Key:  naming.ReplicationCert,
				Path: naming.ReplicationCert,
				Mode: &mode,
			},
			{
				Key:  naming.ReplicationPrivateKey,
				Path: naming.ReplicationPrivateKey,
				Mode: &mode,
			},
			{
				Key:  naming.ReplicationCACert,
				Path: naming.ReplicationCACert,
				Mode: &mode,
			},
		},
	}

	testClientSecretProjection := &v1.SecretProjection{
		LocalObjectReference: v1.LocalObjectReference{
			Name: naming.ReplicationClientCertSecret(postgresCluster).Name,
		},
		Items: []v1.KeyToPath{
			{
				Key:  naming.ReplicationCert,
				Path: naming.ReplicationCertPath,
				Mode: &mode,
			},
			{
				Key:  naming.ReplicationPrivateKey,
				Path: naming.ReplicationPrivateKeyPath,
				Mode: &mode,
			},
		},
	}

	err := AddCertVolumeToPod(postgresCluster, template,
		naming.ContainerClientCertInit, naming.ContainerDatabase,
		naming.ContainerClientCertCopy, testServerSecretProjection,
		testClientSecretProjection)
	assert.NilError(t, err)

	var foundCertVol bool
	var certVol *v1.Volume
	for i, v := range template.Spec.Volumes {
		if v.Name == naming.CertVolume {
			foundCertVol = true
			certVol = &template.Spec.Volumes[i]
			break
		}
	}

	assert.Assert(t, foundCertVol)
	assert.Assert(t, len(certVol.Projected.Sources) > 1)

	var serverSecret *v1.SecretProjection
	var clientSecret *v1.SecretProjection

	for _, source := range certVol.Projected.Sources {

		if source.Secret.Name == naming.PostgresTLSSecret(postgresCluster).Name {
			serverSecret = source.Secret
		}
		if source.Secret.Name == naming.ReplicationClientCertSecret(postgresCluster).Name {
			clientSecret = source.Secret
		}
	}

	if assert.Check(t, serverSecret != nil) {
		assert.Assert(t, len(serverSecret.Items) == 3)

		assert.Equal(t, serverSecret.Items[0].Key, naming.ReplicationCert)
		assert.Equal(t, serverSecret.Items[0].Path, naming.ReplicationCert)
		assert.Equal(t, serverSecret.Items[0].Mode, &mode)

		assert.Equal(t, serverSecret.Items[1].Key, naming.ReplicationPrivateKey)
		assert.Equal(t, serverSecret.Items[1].Path, naming.ReplicationPrivateKey)
		assert.Equal(t, serverSecret.Items[1].Mode, &mode)

		assert.Equal(t, serverSecret.Items[2].Key, naming.ReplicationCACert)
		assert.Equal(t, serverSecret.Items[2].Path, naming.ReplicationCACert)
		assert.Equal(t, serverSecret.Items[2].Mode, &mode)
	}

	if assert.Check(t, clientSecret != nil) {
		assert.Assert(t, len(clientSecret.Items) == 2)

		assert.Equal(t, clientSecret.Items[0].Key, naming.ReplicationCert)
		assert.Equal(t, clientSecret.Items[0].Path, naming.ReplicationCertPath)
		assert.Equal(t, clientSecret.Items[0].Mode, &mode)

		assert.Equal(t, clientSecret.Items[1].Key, naming.ReplicationPrivateKey)
		assert.Equal(t, clientSecret.Items[1].Path, naming.ReplicationPrivateKeyPath)
		assert.Equal(t, clientSecret.Items[1].Mode, &mode)
	}
}

func TestDataVolumeMount(t *testing.T) {
	mount := DataVolumeMount()

	assert.DeepEqual(t, mount, corev1.VolumeMount{
		Name:      "postgres-data",
		MountPath: "/pgdata",
		ReadOnly:  false,
	})
}

func TestWALVolumeMount(t *testing.T) {
	mount := WALVolumeMount()

	assert.DeepEqual(t, mount, corev1.VolumeMount{
		Name:      "postgres-wal",
		MountPath: "/pgwal",
		ReadOnly:  false,
	})
}

func TestInstancePod(t *testing.T) {
	ctx := context.Background()

	cluster := new(v1beta1.PostgresCluster)
	cluster.Default()
	cluster.Spec.PostgresVersion = 11

	dataVolume := new(corev1.PersistentVolumeClaim)
	dataVolume.Name = "datavol"

	instance := new(v1beta1.PostgresInstanceSetSpec)
	instance.Resources.Requests = corev1.ResourceList{"cpu": resource.MustParse("9m")}

	// without WAL volume nor WAL volume spec
	pod := new(corev1.PodSpec)
	InstancePod(ctx, cluster, instance, dataVolume, nil, pod)

	assert.Assert(t, marshalMatches(pod, `
containers:
- env:
  - name: PGDATA
    value: /pgdata/pg11
  - name: PGPORT
    value: "5432"
  name: database
  ports:
  - containerPort: 5432
    name: postgres
    protocol: TCP
  resources:
    requests:
      cpu: 9m
  volumeMounts:
  - mountPath: /pgdata
    name: postgres-data
initContainers:
- command:
  - bash
  - -ceu
  - --
  - |-
    declare -r expected_major_version="$1" pgwal_directory="$2"
    results() { printf '::postgres-operator: %s::%s\n' "$@"; }
    directory() { [ -d "$1" ] || install --directory --mode=0700 "$1"; }
    safelink() (
      local desired="$1" name="$2" current
      current=$(realpath "${name}")
      if [ "${current}" = "${desired}" ]; then return; fi
      set -x; mv --no-target-directory "${current}" "${desired}"
      ln --no-dereference --force --symbolic "${desired}" "${name}"
    )
    echo Initializing ...
    results 'uid' "$(id -u)" 'gid' "$(id -G)"
    results 'postgres path' "$(command -v postgres)"
    results 'postgres version' "${postgres_version:=$(postgres --version)}"
    [[ "${postgres_version}" == *") ${expected_major_version}."* ]]
    results 'config directory' "${PGDATA:?}"
    postgres_data_directory=$([ -d "${PGDATA}" ] && postgres -C data_directory || echo "${PGDATA}")
    results 'data directory' "${postgres_data_directory}"
    [ "${postgres_data_directory}" = "${PGDATA}" ]
    directory "${postgres_data_directory}"
    [ -f "${postgres_data_directory}/PG_VERSION" ] || exit 0
    results 'data version' "${postgres_data_version:=$(< "${postgres_data_directory}/PG_VERSION")}"
    [ "${postgres_data_version}" = "${expected_major_version}" ]
    safelink "${pgwal_directory}" "${postgres_data_directory}/pg_wal"
    results 'wal directory' "$(readlink -f "${postgres_data_directory}/pg_wal")"
  - startup
  - "11"
  - /pgdata/pg11_wal
  env:
  - name: PGDATA
    value: /pgdata/pg11
  - name: PGPORT
    value: "5432"
  name: postgres-startup
  resources:
    requests:
      cpu: 9m
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /pgdata
    name: postgres-data
volumes:
- name: postgres-data
  persistentVolumeClaim:
    claimName: datavol
	`))

	t.Run("WithWALVolumeWithoutWALVolumeSpec", func(t *testing.T) {
		walVolume := new(corev1.PersistentVolumeClaim)
		walVolume.Name = "walvol"

		pod := new(corev1.PodSpec)
		InstancePod(ctx, cluster, instance, dataVolume, walVolume, pod)

		containers := pod.Containers[:0:0]
		containers = append(containers, pod.Containers...)
		containers = append(containers, pod.InitContainers...)

		for _, container := range containers {
			assert.Assert(t, marshalMatches(container.VolumeMounts, `
- mountPath: /pgdata
  name: postgres-data
- mountPath: /pgwal
  name: postgres-wal
			`), "expected WAL mount in %q container", container.Name)
		}

		assert.Assert(t, marshalMatches(pod.Volumes, `
- name: postgres-data
  persistentVolumeClaim:
    claimName: datavol
- name: postgres-wal
  persistentVolumeClaim:
    claimName: walvol
		`), "expected WAL volume")

		// Startup moves WAL files to data volume.
		assert.DeepEqual(t, pod.InitContainers[0].Command[4:],
			[]string{"startup", "11", "/pgdata/pg11_wal"})
	})

	t.Run("WithWALVolumeWithWALVolumeSpec", func(t *testing.T) {
		walVolume := new(corev1.PersistentVolumeClaim)
		walVolume.Name = "walvol"

		instance := new(v1beta1.PostgresInstanceSetSpec)
		instance.WALVolumeClaimSpec = new(corev1.PersistentVolumeClaimSpec)

		pod := new(corev1.PodSpec)
		InstancePod(ctx, cluster, instance, dataVolume, walVolume, pod)

		containers := pod.Containers[:0:0]
		containers = append(containers, pod.Containers...)
		containers = append(containers, pod.InitContainers...)

		for _, container := range containers {
			assert.Assert(t, marshalMatches(container.VolumeMounts, `
- mountPath: /pgdata
  name: postgres-data
- mountPath: /pgwal
  name: postgres-wal
			`), "expected WAL mount in %s", container.Name)
		}

		assert.Assert(t, marshalMatches(pod.Volumes, `
- name: postgres-data
  persistentVolumeClaim:
    claimName: datavol
- name: postgres-wal
  persistentVolumeClaim:
    claimName: walvol
		`), "expected WAL volume")

		// Startup moves WAL files to WAL volume.
		assert.DeepEqual(t, pod.InitContainers[0].Command[4:],
			[]string{"startup", "11", "/pgwal/pg11_wal"})
	})
}
