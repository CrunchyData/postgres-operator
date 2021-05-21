package pgo_cli_test

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

import (
	"strings"
	"testing"
	"time"

	"github.com/crunchydata/postgres-operator/testing/kubeapi"
	"github.com/stretchr/testify/require"
)

// TC60 ✓
// TC122 ✓
// TC130 ✓
func TestClusterBackup(t *testing.T) {
	t.Parallel()

	teardownSchedule := func(t *testing.T, namespace, schedule string) {
		pgo("delete", "schedule", "-n", namespace, "--no-prompt", "--schedule-name="+schedule).Exec(t)
	}

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {
			t.Run("show backup", func(t *testing.T) {
				t.Run("shows something", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)

					// BUG(cbandy): cannot check too soon.
					waitFor(t, func() bool { return false }, 5*time.Second, time.Second)

					output, err := pgo("show", "backup", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotEmpty(t, output)
				})
			})

			t.Run("backup", func(t *testing.T) {
				t.Run("creates an incremental backup", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requireStanzaExists(t, namespace(), cluster(), 2*time.Minute)

					// BUG(cbandy): cannot create a backup too soon.
					waitFor(t, func() bool { return false }, 5*time.Second, time.Second)

					output, err := pgo("backup", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "created")

					exists := func() bool {
						output, err := pgo("show", "backup", cluster(), "-n", namespace()).Exec(t)
						require.NoError(t, err)
						return strings.Contains(output, "incr backup")
					}
					requireWaitFor(t, exists, time.Minute, time.Second,
						"timeout waiting for backup of %q in %q", cluster(), namespace())
				})

				t.Run("accepts options", func(t *testing.T) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requireStanzaExists(t, namespace(), cluster(), 2*time.Minute)

					// BUG(cbandy): cannot create a backup too soon.
					waitFor(t, func() bool { return false }, 5*time.Second, time.Second)

					output, err := pgo("backup", cluster(), "-n", namespace(),
						"--backup-opts=--type=diff",
					).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "created")

					exists := func() bool {
						output, err := pgo("show", "backup", cluster(), "-n", namespace()).Exec(t)
						require.NoError(t, err)
						return strings.Contains(output, "diff backup")
					}
					requireWaitFor(t, exists, time.Minute, time.Second,
						"timeout waiting for backup of %q in %q", cluster(), namespace())
				})
			})

			t.Run("create schedule", func(t *testing.T) {
				t.Run("creates a backup", func(t *testing.T) {
					output, err := pgo("create", "schedule", "--selector=name="+cluster(), "-n", namespace(),
						"--schedule-type=pgbackrest", "--schedule=* * * * *", "--pgbackrest-backup-type=full",
					).Exec(t)
					defer teardownSchedule(t, namespace(), cluster()+"-pgbackrest-full")
					require.NoError(t, err)
					require.Contains(t, output, "created")

					output, err = pgo("show", "schedule", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "pgbackrest-full")

					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requireStanzaExists(t, namespace(), cluster(), 2*time.Minute)

					output, err = pgo("show", "backup", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					before := strings.Count(output, "full backup")

					more := func() bool {
						output, err := pgo("show", "backup", cluster(), "-n", namespace()).Exec(t)
						require.NoError(t, err)
						return strings.Count(output, "full backup") > before
					}
					requireWaitFor(t, more, 75*time.Second, time.Second,
						"timeout waiting for backup to execute on %q in %q", cluster(), namespace())
				})
				t.Run("create backup noselector", func(t *testing.T) {
					output, err := pgo("create", "schedule", cluster(), "-n", namespace(),
						"--schedule-type=pgbackrest", "--schedule=* * * * *", "--pgbackrest-backup-type=full",
					).Exec(t)
					defer teardownSchedule(t, namespace(), cluster()+"-pgbackrest-full")
					require.NoError(t, err)
					require.Contains(t, output, "created")

					output, err = pgo("show", "schedule", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "pgbackrest-full")

					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requireStanzaExists(t, namespace(), cluster(), 2*time.Minute)

					output, err = pgo("show", "backup", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					before := strings.Count(output, "full backup")

					more := func() bool {
						output, err := pgo("show", "backup", cluster(), "-n", namespace()).Exec(t)
						require.NoError(t, err)
						return strings.Count(output, "full backup") > before
					}
					requireWaitFor(t, more, 75*time.Second, time.Second,
						"timeout waiting for backup to execute on %q in %q", cluster(), namespace())
				})
			})

			t.Run("delete schedule", func(t *testing.T) {
				requireSchedule := func(t *testing.T, kind string) {
					_, err := pgo("create", "schedule", "--selector=name="+cluster(), "-n", namespace(),
						"--schedule-type=pgbackrest", "--schedule=* * * * *", "--pgbackrest-backup-type="+kind,
					).Exec(t)
					require.NoError(t, err)
				}

				t.Run("removes all schedules", func(t *testing.T) {
					requireSchedule(t, "diff")
					requireSchedule(t, "full")

					output, err := pgo("delete", "schedule", cluster(), "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "deleted")
					require.Contains(t, output, "pgbackrest-diff")
					require.Contains(t, output, "pgbackrest-full")

					output, err = pgo("show", "schedule", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotContains(t, output, "pgbackrest-diff")
					require.NotContains(t, output, "pgbackrest-full")
				})

				t.Run("accepts schedule name", func(t *testing.T) {
					requireSchedule(t, "diff")
					requireSchedule(t, "full")

					output, err := pgo("delete", "schedule", "-n", namespace(),
						"--schedule-name="+cluster()+"-pgbackrest-diff", "--no-prompt",
					).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "deleted")
					require.Contains(t, output, "pgbackrest-diff")
					require.NotContains(t, output, "pgbackrest-full")

					output, err = pgo("show", "schedule", cluster(), "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.NotContains(t, output, "pgbackrest-diff")
					require.Contains(t, output, "pgbackrest-full")
				})
			})
		})

		t.Run("restore", func(t *testing.T) {
			t.Run("keeps existing pvc", func(t *testing.T) {
				t.Parallel()
				withCluster(t, namespace, func(cluster func() string) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requireStanzaExists(t, namespace(), cluster(), 2*time.Minute)

					before := clusterPVCs(t, namespace(), cluster())
					require.NotEmpty(t, before, "expected volumes to exist")

					// find the creation timestamp for the primary PVC, which wll have the same
					// name as the cluster
					var primaryPVCCreationTimestamp time.Time
					for _, pvc := range before {
						if pvc.GetName() == cluster() {
							primaryPVCCreationTimestamp = pvc.GetCreationTimestamp().Time
						}
					}

					output, err := pgo("restore", cluster(), "--no-prompt", "-n", namespace()).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "restore request")

					// wait for primary PVC to be recreated
					more := func() bool {
						after := clusterPVCs(t, namespace(), cluster())
						for _, pvc := range after {
							// check to see if the PVC for the primary is bound, and has a timestamp
							// equal to the original timestamp for the primary PVC timestamp captured above,
							// indicating that it has not been re-created
							if pvc.GetName() == cluster() && kubeapi.IsPVCBound(pvc) &&
								pvc.GetCreationTimestamp().Time.Equal(primaryPVCCreationTimestamp) {
								return true
							}
						}
						return false
					}
					requireWaitFor(t, more, time.Minute, time.Second,
						"timeout waiting for restore to begin on %q in %q", cluster(), namespace())

					requireClusterReady(t, namespace(), cluster(), 2*time.Minute)
				})
			})

			t.Run("accepts point-in-time options", func(t *testing.T) {
				t.Parallel()
				withCluster(t, namespace, func(cluster func() string) {
					requireClusterReady(t, namespace(), cluster(), time.Minute)
					requireStanzaExists(t, namespace(), cluster(), 2*time.Minute)

					// data that will need to be restored
					_, stderr := clusterPSQL(t, namespace(), cluster(),
						`CREATE TABLE important (data) AS VALUES ('treasure')`)
					require.Empty(t, stderr)

					// point to at which to restore
					recoveryObjective, stderr := clusterPSQL(t, namespace(), cluster(), `
						\set QUIET yes
						\pset format unaligned
						\pset tuples_only yes
						SELECT clock_timestamp()`)
					recoveryObjective = strings.TrimSpace(recoveryObjective)
					require.Empty(t, stderr)

					// a reason to restore followed by a WAL flush
					_, stderr = clusterPSQL(t, namespace(), cluster(), `
						DROP TABLE important;
						DO $$ BEGIN IF current_setting('server_version_num')::int > 100000
							THEN PERFORM pg_switch_wal();
							ELSE PERFORM pg_switch_xlog();
						END IF; END $$`)
					require.Empty(t, stderr)

					output, err := pgo("restore", cluster(), "-n", namespace(),
						"--backup-opts=--type=time",
						"--pitr-target="+recoveryObjective,
						"--no-prompt",
					).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, recoveryObjective)

					restored := func() bool {
						pods, err := TestContext.Kubernetes.ListPods(
							namespace(), map[string]string{
								"pg-cluster":      cluster(),
								"pgo-pg-database": "true",
							})

						if err != nil || len(pods) == 0 {
							return false
						}

						stdout, stderr, err := TestContext.Kubernetes.PodExec(
							pods[0].Namespace, pods[0].Name,
							strings.NewReader(`TABLE important`),
							"psql", "-U", "postgres", "-f-")

						return err == nil && len(stderr) == 0 &&
							strings.Contains(stdout, "(1 row)")
					}
					requireWaitFor(t, restored, 2*time.Minute, time.Second,
						"timeout waiting for restore to finish on %q in %q", cluster(), namespace())

					requireClusterReady(t, namespace(), cluster(), time.Minute)
				})
			})
		})
	})
}
