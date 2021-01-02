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
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRestart(t *testing.T) {
	t.Parallel()

	var replicaOnce sync.Once
	requireReplica := func(t *testing.T, namespace, cluster string) {
		replicaOnce.Do(func() {
			_, err := pgo("scale", cluster, "--no-prompt", "-n", namespace).Exec(t)
			require.NoError(t, err)
			requireReplicasReady(t, namespace, cluster, 5*time.Minute)
		})
	}

	withNamespace(t, func(namespace func() string) {
		withCluster(t, namespace, func(cluster func() string) {

			t.Run("restart", func(t *testing.T) {

				t.Run("query instances", func(t *testing.T) {
					// require a single replica
					requireReplica(t, namespace(), cluster())

					primaries := primaryPods(t, namespace(), cluster())
					// only a single primary is expected
					require.Len(t, primaries, 1)

					replicas := replicaPods(t, namespace(), cluster())
					require.NotEmpty(t, replicas, "expected replica to exist")
					// only a single replica is expected
					require.Len(t, replicas, 1)

					// query for restart targets,
					output, err := pgo("restart", cluster(), "-n", namespace(),
						"--query",
					).Exec(t)
					require.NoError(t, err)

					require.Contains(t, output, primaries[0].Labels["deployment-name"])
					require.Contains(t, output, replicas[0].Labels["deployment-name"])
				})

				type restartTargetSpec struct {
					Name           string
					PendingRestart bool
				}

				type queryRestartOutput struct {
					Results []restartTargetSpec
				}

				t.Run("apply config changes cluster", func(t *testing.T) {

					// require a single replica
					requireReplica(t, namespace(), cluster())

					// wait for DCS config to populate
					hasDCSConfig := func() bool {
						clusterConf, err := TestContext.Kubernetes.Client.CoreV1().ConfigMaps(namespace()).
							Get(fmt.Sprintf("%s-pgha-config", cluster()), metav1.GetOptions{})
						require.NoError(t, err)
						_, ok := clusterConf.Data[fmt.Sprintf("%s-dcs-config", cluster())]
						return ok
					}
					// wait for the primary and replica to show a pending restart
					requireWaitFor(t, hasDCSConfig, time.Minute, time.Second,
						"timeout waiting for the DCS config to populate in ConfigMap %s-pgha-config",
						cluster())

					restartQueryCMD := []string{"restart", cluster(), "-n", namespace(), "--query",
						"-o", "json"}

					// query for restart targets
					output, err := pgo(restartQueryCMD...).Exec(t)
					require.NoError(t, err)

					queryOutput := queryRestartOutput{}
					err = json.Unmarshal([]byte(output), &queryOutput)
					require.NoError(t, err)

					// should return the primary and replica
					require.NotEmpty(t, queryOutput.Results)

					// check that the primary is accounted for
					for _, queryResult := range queryOutput.Results {
						require.False(t, queryResult.PendingRestart)
					}

					// now update a PG setting
					updatePGConfigDCS(t, cluster(), namespace(),
						map[string]string{"unix_socket_directories": "/tmp,/tmp/e2e"})

					requiresRestartPrimaryReplica := func() bool {
						output, err := pgo(restartQueryCMD...).Exec(t)
						require.NoError(t, err)

						queryOutput := queryRestartOutput{}
						err = json.Unmarshal([]byte(output), &queryOutput)
						require.NoError(t, err)

						for _, queryResult := range queryOutput.Results {
							if queryResult.PendingRestart {
								return true
							}
						}
						return false
					}
					// wait for the primary and replica to show a pending restart
					requireWaitFor(t, requiresRestartPrimaryReplica, time.Minute, time.Second,
						"timeout waiting for all instances in cluster %s in namespace %s "+
							"to show a pending restart", cluster(), namespace())

					// now restart the cluster
					_, err = pgo("restart", cluster(), "-n", namespace(), "--no-prompt").Exec(t)
					require.NoError(t, err)

					output, err = pgo(restartQueryCMD...).Exec(t)
					require.NoError(t, err)

					queryOutput = queryRestartOutput{}
					err = json.Unmarshal([]byte(output), &queryOutput)
					require.NoError(t, err)

					require.NotEmpty(t, queryOutput.Results)

					// ensure pending restarts are no longer required
					for _, queryResult := range queryOutput.Results {
						require.False(t, queryResult.PendingRestart)
					}
				})

				t.Run("apply config changes primary", func(t *testing.T) {

					// require a single replica
					requireReplica(t, namespace(), cluster())

					primaries := primaryPods(t, namespace(), cluster())
					// only a single primary is expected
					require.Len(t, primaries, 1)

					// wait for DCS config to populate
					hasDCSConfig := func() bool {
						clusterConf, err := TestContext.Kubernetes.Client.CoreV1().ConfigMaps(namespace()).
							Get(fmt.Sprintf("%s-pgha-config", cluster()), metav1.GetOptions{})
						require.NoError(t, err)
						_, ok := clusterConf.Data[fmt.Sprintf("%s-dcs-config", cluster())]
						return ok
					}
					// wait for the primary and replica to show a pending restart
					requireWaitFor(t, hasDCSConfig, time.Minute, time.Second,
						"timeout waiting for the DCS config to populate in ConfigMap %s-pgha-config",
						cluster())

					restartQueryCMD := []string{"restart", cluster(), "-n", namespace(), "--query",
						"-o", "json"}

					// query for restart targets
					output, err := pgo(restartQueryCMD...).Exec(t)
					require.NoError(t, err)

					queryOutput := queryRestartOutput{}
					err = json.Unmarshal([]byte(output), &queryOutput)
					require.NoError(t, err)

					// query should return the primary and replica
					require.NotEmpty(t, queryOutput.Results)

					// check that the primary is accounted for
					for _, queryResult := range queryOutput.Results {
						require.False(t, queryResult.PendingRestart)
					}

					// now update a PG setting
					updatePGConfigDCS(t, cluster(), namespace(),
						map[string]string{"max_wal_senders": "8"})

					requiresRestartPrimary := func() bool {
						output, err := pgo(restartQueryCMD...).Exec(t)
						require.NoError(t, err)

						queryOutput := queryRestartOutput{}
						err = json.Unmarshal([]byte(output), &queryOutput)
						require.NoError(t, err)

						for _, queryResult := range queryOutput.Results {
							if queryResult.Name == primaries[0].Labels["deployment-name"] &&
								queryResult.PendingRestart {
								return true
							}
						}
						return false
					}
					// wait for the primary to show a pending restart
					requireWaitFor(t, requiresRestartPrimary, time.Minute, time.Second,
						"timeout waiting for primary in cluster %s  namespace %s) "+
							"to show a pending restart", cluster(), namespace())

					// now restart the cluster
					_, err = pgo("restart", cluster(), "-n", namespace(), "--no-prompt",
						"--target", primaries[0].Labels["deployment-name"]).Exec(t)
					require.NoError(t, err)

					output, err = pgo(restartQueryCMD...).Exec(t)
					require.NoError(t, err)

					queryOutput = queryRestartOutput{}
					err = json.Unmarshal([]byte(output), &queryOutput)
					require.NoError(t, err)

					require.NotEmpty(t, queryOutput.Results)

					// ensure pending restarts are no longer required
					for _, queryResult := range queryOutput.Results {
						if queryResult.Name == primaries[0].Labels["deployment-name"] {
							require.False(t, queryResult.PendingRestart)
						}
					}
				})
			})
		})
	})

}
