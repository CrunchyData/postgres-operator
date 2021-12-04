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

package postgres

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

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

func TestDownwardAPIVolumeMount(t *testing.T) {
	mount := DownwardAPIVolumeMount()

	assert.DeepEqual(t, mount, corev1.VolumeMount{
		Name:      "database-containerinfo",
		MountPath: "/etc/database-containerinfo",
		ReadOnly:  true,
	})
}

func TestInstancePod(t *testing.T) {
	ctx := context.Background()

	cluster := new(v1beta1.PostgresCluster)
	cluster.Default()
	cluster.Spec.ImagePullPolicy = corev1.PullAlways
	cluster.Spec.PostgresVersion = 11

	dataVolume := new(corev1.PersistentVolumeClaim)
	dataVolume.Name = "datavol"

	instance := new(v1beta1.PostgresInstanceSetSpec)
	instance.Resources.Requests = corev1.ResourceList{"cpu": resource.MustParse("9m")}
	instance.Sidecars = &v1beta1.InstanceSidecars{
		ReplicaCertCopy: &v1beta1.Sidecar{
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"cpu": resource.MustParse("21m")},
			},
		},
	}

	serverSecretProjection := &corev1.SecretProjection{
		LocalObjectReference: corev1.LocalObjectReference{Name: "srv-secret"},
		Items: []corev1.KeyToPath{
			{
				Key:  naming.ReplicationCert,
				Path: naming.ReplicationCert,
			},
			{
				Key:  naming.ReplicationPrivateKey,
				Path: naming.ReplicationPrivateKey,
			},
			{
				Key:  naming.ReplicationCACert,
				Path: naming.ReplicationCACert,
			},
		},
	}

	clientSecretProjection := &corev1.SecretProjection{
		LocalObjectReference: corev1.LocalObjectReference{Name: "repl-secret"},
		Items: []corev1.KeyToPath{
			{
				Key:  naming.ReplicationCert,
				Path: naming.ReplicationCertPath,
			},
			{
				Key:  naming.ReplicationPrivateKey,
				Path: naming.ReplicationPrivateKeyPath,
			},
		},
	}

	// without WAL volume nor WAL volume spec
	pod := new(corev1.PodSpec)
	InstancePod(ctx, cluster, instance,
		serverSecretProjection, clientSecretProjection, dataVolume, nil, pod)

	assert.Assert(t, marshalMatches(pod, `
containers:
- env:
  - name: PGDATA
    value: /pgdata/pg11
  - name: PGHOST
    value: /tmp/postgres
  - name: PGPORT
    value: "5432"
  - name: KRB5_CONFIG
    value: /etc/postgres/krb5.conf
  - name: KRB5RCACHEDIR
    value: /tmp
  imagePullPolicy: Always
  name: database
  ports:
  - containerPort: 5432
    name: postgres
    protocol: TCP
  resources:
    requests:
      cpu: 9m
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /pgconf/tls
    name: cert-volume
    readOnly: true
  - mountPath: /pgdata
    name: postgres-data
  - mountPath: /etc/database-containerinfo
    name: database-containerinfo
    readOnly: true
- command:
  - bash
  - -ceu
  - --
  - |-
    monitor() {
    declare -r directory="/pgconf/tls"
    exec {fd}<> <(:)
    while read -r -t 5 -u "${fd}" || true; do
      if [ "${directory}" -nt "/proc/self/fd/${fd}" ] &&
        install -D --mode=0600 -t "/tmp/replication" "${directory}"/{replication/tls.crt,replication/tls.key,replication/ca.crt} &&
        pkill -HUP --exact --parent=1 postgres
      then
        exec {fd}>&- && exec {fd}<> <(:)
        stat --format='Loaded certificates dated %y' "${directory}"
      fi
    done
    }; export -f monitor; exec -a "$0" bash -ceu monitor
  - replication-cert-copy
  imagePullPolicy: Always
  name: replication-cert-copy
  resources:
    requests:
      cpu: 21m
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /pgconf/tls
    name: cert-volume
    readOnly: true
initContainers:
- command:
  - bash
  - -ceu
  - --
  - |-
    declare -r expected_major_version="$1" pgwal_directory="$2"
    results() { printf '::postgres-operator: %s::%s\n' "$@"; }
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
    bootstrap_dir="${postgres_data_directory}_bootstrap"
    [ -d "${bootstrap_dir}" ] && results 'bootstrap directory' "${bootstrap_dir}"
    [ -d "${bootstrap_dir}" ] && postgres_data_directory="${bootstrap_dir}"
    install --directory --mode=0700 "${postgres_data_directory}"
    install -D --mode=0600 -t "/tmp/replication" "/pgconf/tls/replication"/{tls.crt,tls.key,ca.crt}
    [ -f "${postgres_data_directory}/PG_VERSION" ] || exit 0
    results 'data version' "${postgres_data_version:=$(< "${postgres_data_directory}/PG_VERSION")}"
    [ "${postgres_data_version}" = "${expected_major_version}" ]
    safelink "${pgwal_directory}" "${postgres_data_directory}/pg_wal"
    results 'wal directory' "$(realpath "${postgres_data_directory}/pg_wal")"
  - startup
  - "11"
  - /pgdata/pg11_wal
  env:
  - name: PGDATA
    value: /pgdata/pg11
  - name: PGHOST
    value: /tmp/postgres
  - name: PGPORT
    value: "5432"
  - name: KRB5_CONFIG
    value: /etc/postgres/krb5.conf
  - name: KRB5RCACHEDIR
    value: /tmp
  imagePullPolicy: Always
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
  - mountPath: /pgconf/tls
    name: cert-volume
    readOnly: true
  - mountPath: /pgdata
    name: postgres-data
volumes:
- name: cert-volume
  projected:
    defaultMode: 384
    sources:
    - secret:
        items:
        - key: tls.crt
          path: tls.crt
        - key: tls.key
          path: tls.key
        - key: ca.crt
          path: ca.crt
        name: srv-secret
    - secret:
        items:
        - key: tls.crt
          path: replication/tls.crt
        - key: tls.key
          path: replication/tls.key
        name: repl-secret
- name: postgres-data
  persistentVolumeClaim:
    claimName: datavol
- downwardAPI:
    items:
    - path: cpu_limit
      resourceFieldRef:
        containerName: database
        divisor: 1m
        resource: limits.cpu
    - path: cpu_request
      resourceFieldRef:
        containerName: database
        divisor: 1m
        resource: requests.cpu
    - path: mem_limit
      resourceFieldRef:
        containerName: database
        divisor: 1Mi
        resource: limits.memory
    - path: mem_request
      resourceFieldRef:
        containerName: database
        divisor: 1Mi
        resource: requests.memory
    - fieldRef:
        apiVersion: v1
        fieldPath: metadata.labels
      path: labels
    - fieldRef:
        apiVersion: v1
        fieldPath: metadata.annotations
      path: annotations
  name: database-containerinfo
	`))

	t.Run("WithWALVolumeWithoutWALVolumeSpec", func(t *testing.T) {
		walVolume := new(corev1.PersistentVolumeClaim)
		walVolume.Name = "walvol"

		pod := new(corev1.PodSpec)
		InstancePod(ctx, cluster, instance,
			serverSecretProjection, clientSecretProjection, dataVolume, walVolume, pod)

		assert.Assert(t, len(pod.Containers) > 0)
		assert.Assert(t, len(pod.InitContainers) > 0)

		// Container has all mountPaths, including downwardAPI
		assert.Assert(t, marshalMatches(pod.Containers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data
- mountPath: /etc/database-containerinfo
  name: database-containerinfo
  readOnly: true
- mountPath: /pgwal
  name: postgres-wal`), "expected WAL and downwardAPI mounts in %q container", pod.Containers[0].Name)

		// InitContainer has all mountPaths, except downwardAPI
		assert.Assert(t, marshalMatches(pod.InitContainers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data
- mountPath: /pgwal
  name: postgres-wal`), "expected WAL mount, no downwardAPI mount in %q container", pod.InitContainers[0].Name)

		assert.Assert(t, marshalMatches(pod.Volumes, `
- name: cert-volume
  projected:
    defaultMode: 384
    sources:
    - secret:
        items:
        - key: tls.crt
          path: tls.crt
        - key: tls.key
          path: tls.key
        - key: ca.crt
          path: ca.crt
        name: srv-secret
    - secret:
        items:
        - key: tls.crt
          path: replication/tls.crt
        - key: tls.key
          path: replication/tls.key
        name: repl-secret
- name: postgres-data
  persistentVolumeClaim:
    claimName: datavol
- downwardAPI:
    items:
    - path: cpu_limit
      resourceFieldRef:
        containerName: database
        divisor: 1m
        resource: limits.cpu
    - path: cpu_request
      resourceFieldRef:
        containerName: database
        divisor: 1m
        resource: requests.cpu
    - path: mem_limit
      resourceFieldRef:
        containerName: database
        divisor: 1Mi
        resource: limits.memory
    - path: mem_request
      resourceFieldRef:
        containerName: database
        divisor: 1Mi
        resource: requests.memory
    - fieldRef:
        apiVersion: v1
        fieldPath: metadata.labels
      path: labels
    - fieldRef:
        apiVersion: v1
        fieldPath: metadata.annotations
      path: annotations
  name: database-containerinfo
- name: postgres-wal
  persistentVolumeClaim:
    claimName: walvol
		`), "expected WAL volume")

		// Startup moves WAL files to data volume.
		assert.DeepEqual(t, pod.InitContainers[0].Command[4:],
			[]string{"startup", "11", "/pgdata/pg11_wal"})
	})

	t.Run("WithAdditionalConfigFiles", func(t *testing.T) {
		clusterWithConfig := cluster.DeepCopy()
		clusterWithConfig.Spec.Config.Files = []corev1.VolumeProjection{
			{
				Secret: &corev1.SecretProjection{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "keytab",
					},
				},
			},
		}

		pod := new(corev1.PodSpec)
		InstancePod(ctx, clusterWithConfig, instance,
			serverSecretProjection, clientSecretProjection, dataVolume, nil, pod)

		assert.Assert(t, len(pod.Containers) > 0)
		assert.Assert(t, len(pod.InitContainers) > 0)

		// Container has all mountPaths, including downwardAPI,
		// and the postgres-config
		assert.Assert(t, marshalMatches(pod.Containers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data
- mountPath: /etc/database-containerinfo
  name: database-containerinfo
  readOnly: true
- mountPath: /etc/postgres
  name: postgres-config
  readOnly: true`), "expected WAL and downwardAPI mounts in %q container", pod.Containers[0].Name)

		// InitContainer has all mountPaths, except downwardAPI and additionalConfig
		assert.Assert(t, marshalMatches(pod.InitContainers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data`), "expected WAL mount, no downwardAPI mount in %q container", pod.InitContainers[0].Name)
	})

	t.Run("WithWALVolumeWithWALVolumeSpec", func(t *testing.T) {
		walVolume := new(corev1.PersistentVolumeClaim)
		walVolume.Name = "walvol"

		instance := new(v1beta1.PostgresInstanceSetSpec)
		instance.WALVolumeClaimSpec = new(corev1.PersistentVolumeClaimSpec)

		pod := new(corev1.PodSpec)
		InstancePod(ctx, cluster, instance,
			serverSecretProjection, clientSecretProjection, dataVolume, walVolume, pod)

		assert.Assert(t, len(pod.Containers) > 0)
		assert.Assert(t, len(pod.InitContainers) > 0)

		assert.Assert(t, marshalMatches(pod.Containers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data
- mountPath: /etc/database-containerinfo
  name: database-containerinfo
  readOnly: true
- mountPath: /pgwal
  name: postgres-wal`), "expected WAL and downwardAPI mounts in %q container", pod.Containers[0].Name)

		assert.Assert(t, marshalMatches(pod.InitContainers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data
- mountPath: /pgwal
  name: postgres-wal`), "expected WAL mount, no downwardAPI mount in %q container", pod.InitContainers[0].Name)

		assert.Assert(t, marshalMatches(pod.Volumes, `
- name: cert-volume
  projected:
    defaultMode: 384
    sources:
    - secret:
        items:
        - key: tls.crt
          path: tls.crt
        - key: tls.key
          path: tls.key
        - key: ca.crt
          path: ca.crt
        name: srv-secret
    - secret:
        items:
        - key: tls.crt
          path: replication/tls.crt
        - key: tls.key
          path: replication/tls.key
        name: repl-secret
- name: postgres-data
  persistentVolumeClaim:
    claimName: datavol
- downwardAPI:
    items:
    - path: cpu_limit
      resourceFieldRef:
        containerName: database
        divisor: 1m
        resource: limits.cpu
    - path: cpu_request
      resourceFieldRef:
        containerName: database
        divisor: 1m
        resource: requests.cpu
    - path: mem_limit
      resourceFieldRef:
        containerName: database
        divisor: 1Mi
        resource: limits.memory
    - path: mem_request
      resourceFieldRef:
        containerName: database
        divisor: 1Mi
        resource: requests.memory
    - fieldRef:
        apiVersion: v1
        fieldPath: metadata.labels
      path: labels
    - fieldRef:
        apiVersion: v1
        fieldPath: metadata.annotations
      path: annotations
  name: database-containerinfo
- name: postgres-wal
  persistentVolumeClaim:
    claimName: walvol
		`), "expected WAL volume")

		// Startup moves WAL files to WAL volume.
		assert.DeepEqual(t, pod.InitContainers[0].Command[4:],
			[]string{"startup", "11", "/pgwal/pg11_wal"})
	})
}

func TestPodSecurityContext(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	cluster.Default()

	assert.Assert(t, marshalMatches(PodSecurityContext(cluster), `
fsGroup: 26
runAsNonRoot: true
	`))

	cluster.Spec.OpenShift = initialize.Bool(true)
	assert.Assert(t, marshalMatches(PodSecurityContext(cluster), `
runAsNonRoot: true
	`))

	cluster.Spec.SupplementalGroups = []int64{}
	assert.Assert(t, marshalMatches(PodSecurityContext(cluster), `
runAsNonRoot: true
	`))

	cluster.Spec.SupplementalGroups = []int64{999, 65000}
	assert.Assert(t, marshalMatches(PodSecurityContext(cluster), `
runAsNonRoot: true
supplementalGroups:
- 999
- 65000
	`))

	*cluster.Spec.OpenShift = false
	assert.Assert(t, marshalMatches(PodSecurityContext(cluster), `
fsGroup: 26
runAsNonRoot: true
supplementalGroups:
- 999
- 65000
	`))

	t.Run("NoRootGID", func(t *testing.T) {
		cluster.Spec.SupplementalGroups = []int64{999, 0, 100, 0}
		assert.DeepEqual(t, []int64{999, 100}, PodSecurityContext(cluster).SupplementalGroups)

		cluster.Spec.SupplementalGroups = []int64{0}
		assert.Assert(t, PodSecurityContext(cluster).SupplementalGroups == nil)
	})
}

func TestUpgradeJob(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: "testnamespace",
		},
	}
	cluster.Spec.ImagePullPolicy = corev1.PullAlways
	cluster.Spec.PostgresVersion = 12
	cluster.Spec.Upgrade = &v1beta1.PGMajorUpgrade{
		Metadata: &v1beta1.Metadata{
			Labels: map[string]string{
				"someLabel": "testvalue1",
			},
			Annotations: map[string]string{
				"someAnnotation": "testvalue2",
			},
		},
		Enabled:             initialize.Bool(true),
		FromPostgresVersion: 11,
		Image:               initialize.String("test-upgrade-image"),
	}

	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "hippo-sa"},
	}

	pgDataVolume := new(corev1.PersistentVolumeClaim)
	pgDataVolume.Name = "datavol"

	pgWALVolume := new(corev1.PersistentVolumeClaim)
	pgWALVolume.Name = "walvol"

	instance := new(v1beta1.PostgresInstanceSetSpec)
	instance.Resources.Requests = corev1.ResourceList{"cpu": resource.MustParse("9m")}
	instance.Sidecars = &v1beta1.InstanceSidecars{
		ReplicaCertCopy: &v1beta1.Sidecar{
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{"cpu": resource.MustParse("21m")},
			},
		},
	}

	serverSecretProjection := &corev1.SecretProjection{
		LocalObjectReference: corev1.LocalObjectReference{Name: "srv-secret"},
		Items: []corev1.KeyToPath{
			{
				Key:  naming.ReplicationCert,
				Path: naming.ReplicationCert,
			},
			{
				Key:  naming.ReplicationPrivateKey,
				Path: naming.ReplicationPrivateKey,
			},
			{
				Key:  naming.ReplicationCACert,
				Path: naming.ReplicationCACert,
			},
		},
	}

	clientSecretProjection := &corev1.SecretProjection{
		LocalObjectReference: corev1.LocalObjectReference{Name: "repl-secret"},
		Items: []corev1.KeyToPath{
			{
				Key:  naming.ReplicationCert,
				Path: naming.ReplicationCertPath,
			},
			{
				Key:  naming.ReplicationPrivateKey,
				Path: naming.ReplicationPrivateKeyPath,
			},
		},
	}

	job, err := GenerateUpgradeJobIntent(cluster, sa.Name, instance, serverSecretProjection,
		clientSecretProjection, pgDataVolume, pgWALVolume)

	assert.NilError(t, err)

	assert.Assert(t, marshalMatches(job, `
metadata:
  annotations:
    postgres-operator.crunchydata.com/pg-upgrade: "12"
    someAnnotation: testvalue2
  creationTimestamp: null
  labels:
    postgres-operator.crunchydata.com/cluster: testcluster
    postgres-operator.crunchydata.com/pgupgrade: ""
    someLabel: testvalue1
  name: testcluster-pgupgrade
  namespace: testnamespace
spec:
  backoffLimit: 0
  template:
    metadata:
      annotations:
        postgres-operator.crunchydata.com/pg-upgrade: "12"
        someAnnotation: testvalue2
      creationTimestamp: null
      labels:
        postgres-operator.crunchydata.com/cluster: testcluster
        postgres-operator.crunchydata.com/pgupgrade: ""
        someLabel: testvalue1
    spec:
      containers:
      - command:
        - bash
        - -ceu
        - --
        - |-
          declare -r old_version="$1" new_version="$2" cluster_name="$3"
          echo -e "Performing PostgreSQL upgrade from version ""${old_version}"" to\
           ""${new_version}"" for cluster ""${cluster_name}"".\n"
          cd /pgdata || exit
          echo -e "Step 1: Making new pgdata directory...\n"
          mkdir /pgdata/pg"${new_version}"
          echo -e "Step 2: Initializing new pgdata directory...\n"
          /usr/pgsql-"${new_version}"/bin/initdb -k -D /pgdata/pg"${new_version}"
          echo -e "\nStep 3: Setting the expected permissions on the old pgdata directory...\n"
          chmod 700 /pgdata/pg"${old_version}"
          echo -e "Step 4: Copying shared_preload_libraries setting to new postgresql.conf file...\n"
          echo "shared_preload_libraries = '$(/usr/pgsql-"""${old_version}"""/bin/postgres -D \
          /pgdata/pg"""${old_version}""" -C shared_preload_libraries)'" >> /pgdata/pg"${new_version}"/postgresql.conf
          echo -e "Step 5: Running pg_upgrade check...\n"
          time /usr/pgsql-"${new_version}"/bin/pg_upgrade --old-bindir /usr/pgsql-"${old_version}"/bin \
          --new-bindir /usr/pgsql-"${new_version}"/bin --old-datadir /pgdata/pg"${old_version}"\
           --new-datadir /pgdata/pg"${new_version}" --link --check
          echo -e "\nStep 6: Running pg_upgrade...\n"
          time /usr/pgsql-"${new_version}"/bin/pg_upgrade --old-bindir /usr/pgsql-"${old_version}"/bin \
          --new-bindir /usr/pgsql-"${new_version}"/bin --old-datadir /pgdata/pg"${old_version}" \
          --new-datadir /pgdata/pg"${new_version}" --link
          echo -e "\nStep 7: Copying patroni.dynamic.json...\n"
          cp /pgdata/pg"${old_version}"/patroni.dynamic.json /pgdata/pg"${new_version}"
          mv /pgdata/pg"${new_version}" /pgdata/pg"${new_version}"_bootstrap
          echo -e "\npg_upgrade Job Complete!"
        - upgrade
        - "11"
        - "12"
        - testcluster
        image: test-upgrade-image
        imagePullPolicy: Always
        name: pgupgrade
        resources: {}
        securityContext:
          allowPrivilegeEscalation: false
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
        volumeMounts:
        - mountPath: /pgconf/tls
          name: cert-volume
          readOnly: true
        - mountPath: /pgdata
          name: postgres-data
        - mountPath: /pgwal
          name: postgres-wal
      restartPolicy: Never
      securityContext:
        fsGroup: 26
        runAsNonRoot: true
      serviceAccountName: hippo-sa
      volumes:
      - name: cert-volume
        projected:
          defaultMode: 384
          sources:
          - secret:
              items:
              - key: tls.crt
                path: tls.crt
              - key: tls.key
                path: tls.key
              - key: ca.crt
                path: ca.crt
              name: srv-secret
          - secret:
              items:
              - key: tls.crt
                path: replication/tls.crt
              - key: tls.key
                path: replication/tls.key
              name: repl-secret
      - name: postgres-data
        persistentVolumeClaim:
          claimName: datavol
      - name: postgres-wal
        persistentVolumeClaim:
          claimName: walvol
status: {}
`))
}

func TestUpgradeCommand(t *testing.T) {
	shellcheck := require.ShellCheck(t)

	cluster := new(v1beta1.PostgresCluster)
	cluster.Name = "test-upgrade-cluster"
	cluster.Spec.PostgresVersion = 13
	cluster.Spec.Upgrade = new(v1beta1.PGMajorUpgrade)
	cluster.Spec.Upgrade.FromPostgresVersion = 12

	command := upgradeCommand(cluster)

	// Expect a bash command with an inline script.
	assert.DeepEqual(t, command[:3], []string{"bash", "-ceu", "--"})
	assert.Assert(t, len(command) > 3)

	// Write out that inline script.
	dir := t.TempDir()
	file := filepath.Join(dir, "script.bash")
	assert.NilError(t, os.WriteFile(file, []byte(command[3]), 0o600))

	// Expect shellcheck to be happy.
	cmd := exec.Command(shellcheck, "--enable=all", file)
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, "%q\n%s", cmd.Args, output)
}
