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
			assert.DeepEqual(t, []string{"bash", "-ceu", "--"}, command[:3])

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
			assert.DeepEqual(t, []string{"bash", "-ceu", "--"}, command[:3])

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
  creationTimestamp: null
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
      creationTimestamp: null
      labels:
        postgres-operator.crunchydata.com/cluster: pg5
        postgres-operator.crunchydata.com/pgupgrade: pgu2
        postgres-operator.crunchydata.com/role: pgupgrade
        postgres-operator.crunchydata.com/version: "25"
    spec:
      containers:
      - command:
        - bash
        - -ceu
        - --
        - |-
          declare -r data_volume='/pgdata' old_version="$1" new_version="$2"
          printf 'Performing PostgreSQL upgrade from version "%s" to "%s" ...\n\n' "$@"
          gid=$(id -G); NSS_WRAPPER_GROUP=$(mktemp)
          (sed "/^postgres:x:/ d; /^[^:]*:x:${gid%% *}:/ d" /etc/group
          echo "postgres:x:${gid%% *}:") > "${NSS_WRAPPER_GROUP}"
          uid=$(id -u); NSS_WRAPPER_PASSWD=$(mktemp)
          (sed "/^postgres:x:/ d; /^[^:]*:x:${uid}:/ d" /etc/passwd
          echo "postgres:x:${uid}:${gid%% *}::${data_volume}:") > "${NSS_WRAPPER_PASSWD}"
          export LD_PRELOAD='libnss_wrapper.so' NSS_WRAPPER_GROUP NSS_WRAPPER_PASSWD
          cd /pgdata || exit
          echo -e "Step 1: Making new pgdata directory...\n"
          mkdir /pgdata/pg"${new_version}"
          echo -e "Step 2: Initializing new pgdata directory...\n"
          /usr/pgsql-"${new_version}"/bin/initdb -k -D /pgdata/pg"${new_version}"
          echo -e "\nStep 3: Setting the expected permissions on the old pgdata directory...\n"
          chmod 750 /pgdata/pg"${old_version}"
          echo -e "Step 4: Copying shared_preload_libraries setting to new postgresql.conf file...\n"
          echo "shared_preload_libraries = '$(/usr/pgsql-"""${old_version}"""/bin/postgres -D \
          /pgdata/pg"""${old_version}""" -C shared_preload_libraries)'" >> /pgdata/pg"${new_version}"/postgresql.conf
          echo -e "Step 5: Running pg_upgrade check...\n"
          time /usr/pgsql-"${new_version}"/bin/pg_upgrade --old-bindir /usr/pgsql-"${old_version}"/bin \
          --new-bindir /usr/pgsql-"${new_version}"/bin --old-datadir /pgdata/pg"${old_version}"\
           --new-datadir /pgdata/pg"${new_version}" --check --link --jobs=1
          echo -e "\nStep 6: Running pg_upgrade...\n"
          time /usr/pgsql-"${new_version}"/bin/pg_upgrade --old-bindir /usr/pgsql-"${old_version}"/bin \
          --new-bindir /usr/pgsql-"${new_version}"/bin --old-datadir /pgdata/pg"${old_version}" \
          --new-datadir /pgdata/pg"${new_version}" --link --jobs=1
          echo -e "\nStep 7: Copying patroni.dynamic.json...\n"
          cp /pgdata/pg"${old_version}"/patroni.dynamic.json /pgdata/pg"${new_version}"
          echo -e "\npg_upgrade Job Complete!"
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
		`/usr/pgsql-"${new_version}"/bin/initdb -k -D /pgdata/pg"${new_version}" --encryption-key-command "echo testKey"`))
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
  creationTimestamp: null
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
      creationTimestamp: null
      labels:
        postgres-operator.crunchydata.com/cluster: pg5
        postgres-operator.crunchydata.com/pgupgrade: pgu2
        postgres-operator.crunchydata.com/role: removedata
    spec:
      containers:
      - command:
        - bash
        - -ceu
        - --
        - |-
          declare -r old_version="$1"
          printf 'Removing PostgreSQL data dir for pg%s...\n\n' "$@"
          echo -e "Checking the directory exists and isn't being used...\n"
          cd /pgdata || exit
          if [ "$(/usr/pgsql-"${old_version}"/bin/pg_controldata /pgdata/pg"${old_version}" | grep -c "shut down in recovery")" -ne 1 ]; then echo -e "Directory in use, cannot remove..."; exit 1; fi
          echo -e "Removing old pgdata directory...\n"
          rm -rf /pgdata/pg"${old_version}" "$(realpath /pgdata/pg${old_version}/pg_wal)"
          echo -e "Remove Data Job Complete!"
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
