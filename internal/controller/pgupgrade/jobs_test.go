// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgupgrade

import (
	"context"
	"os"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestLargestWholeCPU(t *testing.T) {
	assert.Equal(t, int64(0),
		largestWholeCPU(corev1.ResourceRequirements{}),
		"expected the zero value to be zero")

	for _, tt := range []struct {
		Name, ResourcesYAML string
		Result              int64
	}{
		{
			Name: "Negatives", ResourcesYAML: `{requests: {cpu: -3}, limits: {cpu: -5}}`,
			Result: 0,
		},
		{
			Name: "SmallPositive", ResourcesYAML: `limits: {cpu: 600m}`,
			Result: 0,
		},
		{
			Name: "FractionalPositive", ResourcesYAML: `requests: {cpu: 2200m}`,
			Result: 2,
		},
		{
			Name: "LargePositive", ResourcesYAML: `limits: {cpu: 10}`,
			Result: 10,
		},
		{
			Name: "RequestsAndLimits", ResourcesYAML: `{requests: {cpu: 2}, limits: {cpu: 4}}`,
			Result: 4,
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			var resources corev1.ResourceRequirements
			require.UnmarshalInto(t, &resources, tt.ResourcesYAML)
			assert.Equal(t, tt.Result, largestWholeCPU(resources))
		})
	}
}

func TestUpgradeCommand(t *testing.T) {
	expectScript := func(t *testing.T, script string) {
		t.Helper()

		t.Run("PrettyYAML", func(t *testing.T) {
			b, err := yaml.Marshal(script)
			assert.NilError(t, err)
			assert.Assert(t, strings.HasPrefix(string(b), `|`),
				"expected literal block scalar, got:\n%s", b)
		})
	}

	t.Run("Jobs", func(t *testing.T) {
		for _, tt := range []struct {
			Spec int32
			Args string
		}{
			{Spec: -1, Args: "--jobs=1"},
			{Spec: 0, Args: "--jobs=1"},
			{Spec: 1, Args: "--jobs=1"},
			{Spec: 2, Args: "--jobs=2"},
			{Spec: 10, Args: "--jobs=10"},
		} {
			spec := &v1beta1.PGUpgradeSettings{Jobs: tt.Spec}
			command := upgradeCommand(spec, "")
			assert.Assert(t, len(command) > 3)
			assert.DeepEqual(t, []string{"bash", "-c", "--"}, command[:3])

			script := command[3]
			assert.Assert(t, cmp.Contains(script, tt.Args))

			expectScript(t, script)
		}
	})

	t.Run("Method", func(t *testing.T) {
		for _, tt := range []struct {
			Spec string
			Args string
		}{
			{Spec: "", Args: "--link"},
			{Spec: "mystery!", Args: "--link"},
			{Spec: "Link", Args: "--link"},
			{Spec: "Clone", Args: "--clone"},
			{Spec: "Copy", Args: "--copy"},
			{Spec: "CopyFileRange", Args: "--copy-file-range"},
		} {
			spec := &v1beta1.PGUpgradeSettings{TransferMethod: tt.Spec}
			command := upgradeCommand(spec, "")
			assert.Assert(t, len(command) > 3)
			assert.DeepEqual(t, []string{"bash", "-c", "--"}, command[:3])

			script := command[3]
			assert.Assert(t, cmp.Contains(script, tt.Args))

			expectScript(t, script)
		}

	})
}

func TestGenerateUpgradeJob(t *testing.T) {
	ctx := context.Background()
	reconciler := &PGUpgradeReconciler{}

	upgrade := &v1beta1.PGUpgrade{}
	upgrade.Namespace = "ns1"
	upgrade.Name = "pgu2"
	upgrade.UID = "uid3"
	upgrade.Spec.Image = initialize.Pointer("img4")
	upgrade.Spec.PostgresClusterName = "pg5"
	upgrade.Spec.FromPostgresVersion = 19
	upgrade.Spec.ToPostgresVersion = 25
	upgrade.Spec.Resources.Requests = corev1.ResourceList{
		corev1.ResourceCPU: resource.MustParse("3.14"),
	}

	startup := &appsv1.StatefulSet{}
	startup.Spec.Template.Spec = corev1.PodSpec{
		Containers: []corev1.Container{{
			Name: ContainerDatabase,

			SecurityContext: &corev1.SecurityContext{Privileged: new(bool)},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "vm1", MountPath: "/mnt/some/such"},
			},
		}},
		Volumes: []corev1.Volume{
			{
				Name: "vol2",
				VolumeSource: corev1.VolumeSource{
					HostPath: new(corev1.HostPathVolumeSource),
				},
			},
		},
	}

	job := reconciler.generateUpgradeJob(ctx, upgrade, startup, "")
	assert.Assert(t, cmp.MarshalMatches(job, `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    kubectl.kubernetes.io/default-container: database
  labels:
    postgres-operator.crunchydata.com/cluster: pg5
    postgres-operator.crunchydata.com/pgupgrade: pgu2
    postgres-operator.crunchydata.com/role: pgupgrade
    postgres-operator.crunchydata.com/version: "25"
  name: pgu2-pgdata
  namespace: ns1
  ownerReferences:
  - apiVersion: postgres-operator.crunchydata.com/v1beta1
    blockOwnerDeletion: true
    controller: true
    kind: PGUpgrade
    name: pgu2
    uid: uid3
spec:
  backoffLimit: 0
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: database
      labels:
        postgres-operator.crunchydata.com/cluster: pg5
        postgres-operator.crunchydata.com/pgupgrade: pgu2
        postgres-operator.crunchydata.com/role: pgupgrade
        postgres-operator.crunchydata.com/version: "25"
    spec:
      containers:
      - command:
        - bash
        - -c
        - --
        - |-
          shopt -so errexit nounset
          declare -r data_volume='/pgdata' old_version="$1" new_version="$2"
          printf 'Performing PostgreSQL upgrade from version "%s" to "%s" ...\n' "$@"
          section() { printf '\n\n%s\n' "$@"; }
          section 'Step 1 of 7: Ensuring username is postgres...'
          gid=$(id -G); NSS_WRAPPER_GROUP=$(mktemp)
          (sed "/^postgres:x:/ d; /^[^:]*:x:${gid%% *}:/ d" /etc/group
          echo "postgres:x:${gid%% *}:") > "${NSS_WRAPPER_GROUP}"
          uid=$(id -u); NSS_WRAPPER_PASSWD=$(mktemp)
          (sed "/^postgres:x:/ d; /^[^:]*:x:${uid}:/ d" /etc/passwd
          echo "postgres:x:${uid}:${gid%% *}::${data_volume}:") > "${NSS_WRAPPER_PASSWD}"
          export LD_PRELOAD='libnss_wrapper.so' NSS_WRAPPER_GROUP NSS_WRAPPER_PASSWD
          id; [[ "$(id -nu)" == 'postgres' && "$(id -ng)" == 'postgres' ]]
          section 'Step 2 of 7: Finding data and tools...'
          old_data="${data_volume}/pg${old_version}" && [[ -d "${old_data}" ]]
          new_data="${data_volume}/pg${new_version}"
          old_bin=$(PATH="/usr/lib/postgresql/19/bin:/usr/libexec/postgresql19:/usr/pgsql-19/bin${PATH+:${PATH}}" && command -v postgres)
          old_bin="${old_bin%/postgres}"
          new_bin=$(PATH="/usr/lib/postgresql/25/bin:/usr/libexec/postgresql25:/usr/pgsql-25/bin${PATH+:${PATH}}" && command -v pg_upgrade)
          new_bin="${new_bin%/pg_upgrade}"
          (set -x && [[ "$("${old_bin}/postgres" --version)" =~ ") ${old_version}"($|[^0-9]) ]])
          (set -x && [[ "$("${new_bin}/initdb" --version)"   =~ ") ${new_version}"($|[^0-9]) ]])
          cd "${data_volume}"
          control=$(LC_ALL=C PGDATA="${old_data}" "${old_bin}/pg_controldata")
          read -r checksums <<< "${control##*page checksum version:}"
          checksums=$(if [[ "${checksums}" -gt 0 ]]; then echo '--data-checksums'; elif [[ "${new_version}" -ge 18 ]]; then echo '--no-data-checksums'; fi)
          section 'Step 3 of 7: Initializing new data directory...'
          PGDATA="${new_data}" "${new_bin}/initdb" --allow-group-access ${checksums}
          section 'Step 4 of 7: Copying shared_preload_libraries parameter...'
          value=$(LC_ALL=C PGDATA="${old_data}" "${old_bin}/postgres" -C shared_preload_libraries)
          echo >> "${new_data}/postgresql.conf" "shared_preload_libraries = '${value//$'\''/$'\'\''}'"
          section 'Step 5 of 7: Checking for potential issues...'
          "${new_bin}/pg_upgrade" --check --link --jobs=1 \
          --old-bindir="${old_bin}" --old-datadir="${old_data}" \
          --new-bindir="${new_bin}" --new-datadir="${new_data}"
          section 'Step 6 of 7: Performing upgrade...'
          (set -x && time "${new_bin}/pg_upgrade" --link --jobs=1 \
          --old-bindir="${old_bin}" --old-datadir="${old_data}" \
          --new-bindir="${new_bin}" --new-datadir="${new_data}")
          section 'Step 7 of 7: Copying Patroni settings...'
          (set -x && cp "${old_data}/patroni.dynamic.json" "${new_data}")
          section 'Success!'
        - upgrade
        - "19"
        - "25"
        image: img4
        name: database
        resources:
          requests:
            cpu: 3140m
        securityContext:
          privileged: false
        volumeMounts:
        - mountPath: /mnt/some/such
          name: vm1
      restartPolicy: Never
      volumes:
      - hostPath:
          path: ""
        name: vol2
status: {}
	`))

	t.Run(feature.PGUpgradeCPUConcurrency+"Enabled", func(t *testing.T) {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.PGUpgradeCPUConcurrency: true,
		}))
		ctx := feature.NewContext(context.Background(), gate)

		job := reconciler.generateUpgradeJob(ctx, upgrade, startup, "")
		assert.Assert(t, cmp.MarshalContains(job, `--jobs=2`))
	})

	tdeJob := reconciler.generateUpgradeJob(ctx, upgrade, startup, "echo testKey")
	assert.Assert(t, cmp.MarshalContains(tdeJob,
		`PGDATA="${new_data}" "${new_bin}/initdb" --allow-group-access ${checksums} --encryption-key-command='echo testKey'`))
}

func TestGenerateRemoveDataJob(t *testing.T) {
	ctx := context.Background()
	reconciler := &PGUpgradeReconciler{}

	upgrade := &v1beta1.PGUpgrade{}
	upgrade.Namespace = "ns1"
	upgrade.Name = "pgu2"
	upgrade.UID = "uid3"
	upgrade.Spec.Image = initialize.Pointer("img4")
	upgrade.Spec.PostgresClusterName = "pg5"
	upgrade.Spec.FromPostgresVersion = 19
	upgrade.Spec.ToPostgresVersion = 25
	upgrade.Spec.Resources.Requests = corev1.ResourceList{
		corev1.ResourceCPU: resource.MustParse("3.14"),
	}

	sts := &appsv1.StatefulSet{}
	sts.Name = "sts"
	sts.Spec.Template.Spec = corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            ContainerDatabase,
			Image:           "img3",
			SecurityContext: &corev1.SecurityContext{Privileged: new(bool)},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "vm1", MountPath: "/mnt/some/such"},
			},
		}},
		Volumes: []corev1.Volume{
			{
				Name: "vol2",
				VolumeSource: corev1.VolumeSource{
					HostPath: new(corev1.HostPathVolumeSource),
				},
			},
		},
	}

	job := reconciler.generateRemoveDataJob(ctx, upgrade, sts)
	assert.Assert(t, cmp.MarshalMatches(job, `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    kubectl.kubernetes.io/default-container: database
  labels:
    postgres-operator.crunchydata.com/cluster: pg5
    postgres-operator.crunchydata.com/pgupgrade: pgu2
    postgres-operator.crunchydata.com/role: removedata
  name: pgu2-sts
  namespace: ns1
  ownerReferences:
  - apiVersion: postgres-operator.crunchydata.com/v1beta1
    blockOwnerDeletion: true
    controller: true
    kind: PGUpgrade
    name: pgu2
    uid: uid3
spec:
  backoffLimit: 0
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: database
      labels:
        postgres-operator.crunchydata.com/cluster: pg5
        postgres-operator.crunchydata.com/pgupgrade: pgu2
        postgres-operator.crunchydata.com/role: removedata
    spec:
      containers:
      - command:
        - bash
        - -c
        - --
        - |-
          shopt -so errexit nounset
          declare -r data_volume='/pgdata' old_version="$1"
          printf 'Removing PostgreSQL %s data...\n\n' "$@"
          delete() (set -x && rm -rf -- "$@")
          old_data="${data_volume}/pg${old_version}"
          control=$(PATH="/usr/lib/postgresql/19/bin:/usr/libexec/postgresql19:/usr/pgsql-19/bin${PATH+:${PATH}}" && LC_ALL=C pg_controldata "${old_data}")
          read -r state <<< "${control##*cluster state:}"
          [[ "${state}" == 'shut down in recovery' ]] || { printf >&2 'Unexpected state! %q\n' "${state}"; exit 1; }
          delete "${old_data}/pg_wal/"
          delete "${old_data}" && echo 'Success!'
        - remove
        - "19"
        image: img4
        name: database
        resources:
          requests:
            cpu: 3140m
        securityContext:
          privileged: false
        volumeMounts:
        - mountPath: /mnt/some/such
          name: vm1
      restartPolicy: Never
      volumes:
      - hostPath:
          path: ""
        name: vol2
status: {}
	`))
}

func TestPGUpgradeContainerImage(t *testing.T) {
	upgrade := &v1beta1.PGUpgrade{}

	t.Setenv("RELATED_IMAGE_PGUPGRADE", "")
	os.Unsetenv("RELATED_IMAGE_PGUPGRADE")
	assert.Equal(t, pgUpgradeContainerImage(upgrade), "")

	t.Setenv("RELATED_IMAGE_PGUPGRADE", "")
	assert.Equal(t, pgUpgradeContainerImage(upgrade), "")

	t.Setenv("RELATED_IMAGE_PGUPGRADE", "env-var-pgbackrest")
	assert.Equal(t, pgUpgradeContainerImage(upgrade), "env-var-pgbackrest")

	require.UnmarshalInto(t, &upgrade.Spec, `{ image: spec-image }`)
	assert.Equal(t, pgUpgradeContainerImage(upgrade), "spec-image")
}

func TestVerifyUpgradeImageValue(t *testing.T) {
	upgrade := &v1beta1.PGUpgrade{}

	t.Run("crunchy-postgres", func(t *testing.T) {
		t.Setenv("RELATED_IMAGE_PGUPGRADE", "")
		os.Unsetenv("RELATED_IMAGE_PGUPGRADE")
		err := verifyUpgradeImageValue(upgrade)
		assert.ErrorContains(t, err, "crunchy-upgrade")
	})

}
