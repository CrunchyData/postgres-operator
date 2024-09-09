// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
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

func TestTablespaceVolumeMount(t *testing.T) {
	mount := TablespaceVolumeMount("trial")

	assert.DeepEqual(t, mount, corev1.VolumeMount{
		Name:      "tablespace-trial",
		MountPath: "/tablespaces/trial",
		ReadOnly:  false,
	})
}

func TestInstancePod(t *testing.T) {
	t.Parallel()

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
		serverSecretProjection, clientSecretProjection, dataVolume, nil, nil, pod)

	assert.Assert(t, cmp.MarshalMatches(pod, `
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
  - name: LDAPTLS_CACERT
    value: /etc/postgres/ldap/ca.crt
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
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
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
    # Parameters for curl when managing autogrow annotation.
    APISERVER="https://kubernetes.default.svc"
    SERVICEACCOUNT="/var/run/secrets/kubernetes.io/serviceaccount"
    NAMESPACE=$(cat ${SERVICEACCOUNT}/namespace)
    TOKEN=$(cat ${SERVICEACCOUNT}/token)
    CACERT=${SERVICEACCOUNT}/ca.crt

    declare -r directory="/pgconf/tls"
    exec {fd}<> <(:||:)
    while read -r -t 5 -u "${fd}" ||:; do
      # Manage replication certificate.
      if [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] &&
        install -D --mode=0600 -t "/tmp/replication" "${directory}"/{replication/tls.crt,replication/tls.key,replication/ca.crt} &&
        pkill -HUP --exact --parent=1 postgres
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --format='Loaded certificates dated %y' "${directory}"
      fi

      # Manage autogrow annotation.
      # Return size in Mebibytes.
      size=$(df --human-readable --block-size=M /pgdata | awk 'FNR == 2 {print $2}')
      use=$(df --human-readable /pgdata | awk 'FNR == 2 {print $5}')
      sizeInt="${size//M/}"
      # Use the sed punctuation class, because the shell will not accept the percent sign in an expansion.
      useInt=$(echo $use | sed 's/[[:punct:]]//g')
      triggerExpansion="$((useInt > 75))"
      if [ $triggerExpansion -eq 1 ]; then
        newSize="$(((sizeInt / 2)+sizeInt))"
        newSizeMi="${newSize}Mi"
        d='[{"op": "add", "path": "/metadata/annotations/suggested-pgdata-pvc-size", "value": "'"$newSizeMi"'"}]'
        curl --cacert ${CACERT} --header "Authorization: Bearer ${TOKEN}" -XPATCH "${APISERVER}/api/v1/namespaces/${NAMESPACE}/pods/${HOSTNAME}?fieldManager=kubectl-annotate" -H "Content-Type: application/json-patch+json" --data "$d"
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
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  volumeMounts:
  - mountPath: /pgconf/tls
    name: cert-volume
    readOnly: true
  - mountPath: /pgdata
    name: postgres-data
initContainers:
- command:
  - bash
  - -ceu
  - --
  - |-
    declare -r expected_major_version="$1" pgwal_directory="$2" pgbrLog_directory="$3"
    permissions() { while [[ -n "$1" ]]; do set "${1%/*}" "$@"; done; shift; stat -Lc '%A %4u %4g %n' "$@"; }
    halt() { local rc=$?; >&2 echo "$@"; exit "${rc/#0/1}"; }
    results() { printf '::postgres-operator: %s::%s\n' "$@"; }
    recreate() (
      local tmp; tmp=$(mktemp -d -p "${1%/*}"); GLOBIGNORE='.:..'; set -x
      chmod "$2" "${tmp}"; mv "$1"/* "${tmp}"; rmdir "$1"; mv "${tmp}" "$1"
    )
    safelink() (
      local desired="$1" name="$2" current
      current=$(realpath "${name}")
      if [[ "${current}" == "${desired}" ]]; then return; fi
      set -x; mv --no-target-directory "${current}" "${desired}"
      ln --no-dereference --force --symbolic "${desired}" "${name}"
    )
    echo Initializing ...
    results 'uid' "$(id -u ||:)" 'gid' "$(id -G ||:)"
    if [[ "${pgwal_directory}" == *"pgwal/"* ]] && [[ ! -d "/pgwal/pgbackrest-spool" ]];then rm -rf "/pgdata/pgbackrest-spool" && mkdir -p "/pgwal/pgbackrest-spool" && ln --force --symbolic "/pgwal/pgbackrest-spool" "/pgdata/pgbackrest-spool";fi
    if [[ ! -e "/pgdata/pgbackrest-spool" ]];then rm -rf /pgdata/pgbackrest-spool;fi
    results 'postgres path' "$(command -v postgres ||:)"
    results 'postgres version' "${postgres_version:=$(postgres --version ||:)}"
    [[ "${postgres_version}" =~ ") ${expected_major_version}"($|[^0-9]) ]] ||
    halt Expected PostgreSQL version "${expected_major_version}"
    results 'config directory' "${PGDATA:?}"
    postgres_data_directory=$([[ -d "${PGDATA}" ]] && postgres -C data_directory || echo "${PGDATA}")
    results 'data directory' "${postgres_data_directory}"
    [[ "${postgres_data_directory}" == "${PGDATA}" ]] ||
    halt Expected matching config and data directories
    bootstrap_dir="${postgres_data_directory}_bootstrap"
    [[ -d "${bootstrap_dir}" ]] && results 'bootstrap directory' "${bootstrap_dir}"
    [[ -d "${bootstrap_dir}" ]] && postgres_data_directory="${bootstrap_dir}"
    if [[ ! -e "${postgres_data_directory}" || -O "${postgres_data_directory}" ]]; then
    install --directory --mode=0700 "${postgres_data_directory}"
    elif [[ -w "${postgres_data_directory}" && -g "${postgres_data_directory}" ]]; then
    recreate "${postgres_data_directory}" '0700'
    else (halt Permissions!); fi ||
    halt "$(permissions "${postgres_data_directory}" ||:)"
    results 'pgBackRest log directory' "${pgbrLog_directory}"
    install --directory --mode=0775 "${pgbrLog_directory}" ||
    halt "$(permissions "${pgbrLog_directory}" ||:)"
    install -D --mode=0600 -t "/tmp/replication" "/pgconf/tls/replication"/{tls.crt,tls.key,ca.crt}


    [[ -f "${postgres_data_directory}/PG_VERSION" ]] || exit 0
    results 'data version' "${postgres_data_version:=$(< "${postgres_data_directory}/PG_VERSION")}"
    [[ "${postgres_data_version}" == "${expected_major_version}" ]] ||
    halt Expected PostgreSQL data version "${expected_major_version}"
    [[ ! -f "${postgres_data_directory}/postgresql.conf" ]] &&
    touch "${postgres_data_directory}/postgresql.conf"
    safelink "${pgwal_directory}" "${postgres_data_directory}/pg_wal"
    results 'wal directory' "$(realpath "${postgres_data_directory}/pg_wal" ||:)"
    rm -f "${postgres_data_directory}/recovery.signal"
  - startup
  - "11"
  - /pgdata/pg11_wal
  - /pgdata/pgbackrest/log
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
  - name: LDAPTLS_CACERT
    value: /etc/postgres/ldap/ca.crt
  imagePullPolicy: Always
  name: postgres-startup
  resources:
    requests:
      cpu: 9m
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
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
			serverSecretProjection, clientSecretProjection, dataVolume, walVolume, nil, pod)

		assert.Assert(t, len(pod.Containers) > 0)
		assert.Assert(t, len(pod.InitContainers) > 0)

		// Container has all mountPaths, including downwardAPI
		assert.Assert(t, cmp.MarshalMatches(pod.Containers[0].VolumeMounts, `
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
		assert.Assert(t, cmp.MarshalMatches(pod.InitContainers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data
- mountPath: /pgwal
  name: postgres-wal`), "expected WAL mount, no downwardAPI mount in %q container", pod.InitContainers[0].Name)

		assert.Assert(t, cmp.MarshalMatches(pod.Volumes, `
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
			[]string{"startup", "11", "/pgdata/pg11_wal", "/pgdata/pgbackrest/log"})
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
			serverSecretProjection, clientSecretProjection, dataVolume, nil, nil, pod)

		assert.Assert(t, len(pod.Containers) > 0)
		assert.Assert(t, len(pod.InitContainers) > 0)

		// Container has all mountPaths, including downwardAPI,
		// and the postgres-config
		assert.Assert(t, cmp.MarshalMatches(pod.Containers[0].VolumeMounts, `
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
		assert.Assert(t, cmp.MarshalMatches(pod.InitContainers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data`), "expected WAL mount, no downwardAPI mount in %q container", pod.InitContainers[0].Name)
	})

	t.Run("WithCustomSidecarContainer", func(t *testing.T) {
		sidecarInstance := new(v1beta1.PostgresInstanceSetSpec)
		sidecarInstance.Containers = []corev1.Container{
			{Name: "customsidecar1"},
		}

		t.Run("SidecarNotEnabled", func(t *testing.T) {
			InstancePod(ctx, cluster, sidecarInstance,
				serverSecretProjection, clientSecretProjection, dataVolume, nil, nil, pod)

			assert.Equal(t, len(pod.Containers), 2, "expected 2 containers in Pod, got %d", len(pod.Containers))
		})

		t.Run("SidecarEnabled", func(t *testing.T) {
			gate := feature.NewGate()
			assert.NilError(t, gate.SetFromMap(map[string]bool{
				feature.InstanceSidecars: true,
			}))
			ctx := feature.NewContext(ctx, gate)

			InstancePod(ctx, cluster, sidecarInstance,
				serverSecretProjection, clientSecretProjection, dataVolume, nil, nil, pod)

			assert.Equal(t, len(pod.Containers), 3, "expected 3 containers in Pod, got %d", len(pod.Containers))

			var found bool
			for i := range pod.Containers {
				if pod.Containers[i].Name == "customsidecar1" {
					found = true
					break
				}
			}
			assert.Assert(t, found, "expected custom sidecar 'customsidecar1', but container not found")
		})
	})

	t.Run("WithTablespaces", func(t *testing.T) {
		clusterWithTablespaces := cluster.DeepCopy()
		clusterWithTablespaces.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{
			{
				TablespaceVolumes: []v1beta1.TablespaceVolume{
					{Name: "trial"},
					{Name: "castle"},
				},
			},
		}

		tablespaceVolume1 := new(corev1.PersistentVolumeClaim)
		tablespaceVolume1.Labels = map[string]string{
			"postgres-operator.crunchydata.com/data": "castle",
		}
		tablespaceVolume2 := new(corev1.PersistentVolumeClaim)
		tablespaceVolume2.Labels = map[string]string{
			"postgres-operator.crunchydata.com/data": "trial",
		}
		tablespaceVolumes := []*corev1.PersistentVolumeClaim{tablespaceVolume1, tablespaceVolume2}

		InstancePod(ctx, cluster, instance,
			serverSecretProjection, clientSecretProjection, dataVolume, nil, tablespaceVolumes, pod)

		assert.Assert(t, cmp.MarshalMatches(pod.Containers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data
- mountPath: /etc/database-containerinfo
  name: database-containerinfo
  readOnly: true
- mountPath: /tablespaces/castle
  name: tablespace-castle
- mountPath: /tablespaces/trial
  name: tablespace-trial`), "expected tablespace mount(s) in %q container", pod.Containers[0].Name)

		// InitContainer has all mountPaths, except downwardAPI and additionalConfig
		assert.Assert(t, cmp.MarshalMatches(pod.InitContainers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data
- mountPath: /tablespaces/castle
  name: tablespace-castle
- mountPath: /tablespaces/trial
  name: tablespace-trial`), "expected tablespace mount(s) in %q container", pod.InitContainers[0].Name)
	})

	t.Run("WithWALVolumeWithWALVolumeSpec", func(t *testing.T) {
		walVolume := new(corev1.PersistentVolumeClaim)
		walVolume.Name = "walvol"

		instance := new(v1beta1.PostgresInstanceSetSpec)
		instance.WALVolumeClaimSpec = new(corev1.PersistentVolumeClaimSpec)

		pod := new(corev1.PodSpec)
		InstancePod(ctx, cluster, instance,
			serverSecretProjection, clientSecretProjection, dataVolume, walVolume, nil, pod)

		assert.Assert(t, len(pod.Containers) > 0)
		assert.Assert(t, len(pod.InitContainers) > 0)

		assert.Assert(t, cmp.MarshalMatches(pod.Containers[0].VolumeMounts, `
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

		assert.Assert(t, cmp.MarshalMatches(pod.InitContainers[0].VolumeMounts, `
- mountPath: /pgconf/tls
  name: cert-volume
  readOnly: true
- mountPath: /pgdata
  name: postgres-data
- mountPath: /pgwal
  name: postgres-wal`), "expected WAL mount, no downwardAPI mount in %q container", pod.InitContainers[0].Name)

		assert.Assert(t, cmp.MarshalMatches(pod.Volumes, `
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
			[]string{"startup", "11", "/pgwal/pg11_wal", "/pgdata/pgbackrest/log"})
	})
}

func TestPodSecurityContext(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	cluster.Default()

	assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(cluster), `
fsGroup: 26
fsGroupChangePolicy: OnRootMismatch
	`))

	cluster.Spec.OpenShift = initialize.Bool(true)
	assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(cluster), `
fsGroupChangePolicy: OnRootMismatch
	`))

	cluster.Spec.SupplementalGroups = []int64{}
	assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(cluster), `
fsGroupChangePolicy: OnRootMismatch
	`))

	cluster.Spec.SupplementalGroups = []int64{999, 65000}
	assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(cluster), `
fsGroupChangePolicy: OnRootMismatch
supplementalGroups:
- 999
- 65000
	`))

	*cluster.Spec.OpenShift = false
	assert.Assert(t, cmp.MarshalMatches(PodSecurityContext(cluster), `
fsGroup: 26
fsGroupChangePolicy: OnRootMismatch
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
