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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClusterAnnotation(t *testing.T) {
	t.Parallel()

	withNamespace(t, func(namespace func() string) {
		t.Run("on create", func(t *testing.T) {
			t.Parallel()
			tests := []struct {
				testName     string
				annotations  map[string]string
				clusterFlags []string
				addFlags     []string
				removeFlags  []string
				deployments  []string
			}{
				{
					testName: "create-global",
					annotations: map[string]string{
						"global":  "here",
						"global2": "foo",
					},
					clusterFlags: []string{"--pgbouncer"},
					addFlags:     []string{"--annotation=global=here", "--annotation=global2=foo"},
					removeFlags:  []string{"--annotation=global-", "--annotation=global2-"},
					deployments:  []string{"create-global", "create-global-backrest-shared-repo", "create-global-pgbouncer"},
				}, {
					testName: "create-postgres",
					annotations: map[string]string{
						"postgres": "present",
					},
					clusterFlags: []string{},
					addFlags:     []string{"--annotation-postgres=postgres=present"},
					removeFlags:  []string{"--annotation-postgres=postgres-"},
					deployments:  []string{"create-postgres"},
				}, {
					testName: "create-pgbackrest",
					annotations: map[string]string{
						"pgbackrest": "what",
					},
					clusterFlags: []string{},
					addFlags:     []string{"--annotation-pgbackrest=pgbackrest=what"},
					removeFlags:  []string{"--annotation-pgbackrest=pgbackrest-"},
					deployments:  []string{"create-pgbackrest-backrest-shared-repo"},
				}, {
					testName: "create-pgbouncer",
					annotations: map[string]string{
						"pgbouncer": "aqui",
					},
					clusterFlags: []string{"--pgbouncer"},
					addFlags:     []string{"--annotation-pgbouncer=pgbouncer=aqui"},
					removeFlags:  []string{"--annotation-pgbouncer=pgbouncer-"},
					deployments:  []string{"create-pgbouncer-pgbouncer"},
				}, {
					testName: "remove-one",
					annotations: map[string]string{
						"leave": "me",
					},
					clusterFlags: []string{"--pgbouncer"},
					addFlags:     []string{"--annotation=remove=me", "--annotation=leave=me"},
					removeFlags:  []string{"--annotation=remove-"},
					deployments:  []string{"remove-one", "remove-one-backrest-shared-repo", "remove-one-pgbouncer"},
				},
			}

			for _, test := range tests {
				test := test // lock test variable in for each run since it changes across parallel loops
				t.Run(test.testName, func(t *testing.T) {
					t.Parallel()
					createCMD := []string{"create", "cluster", test.testName, "-n", namespace()}
					createCMD = append(createCMD, test.clusterFlags...)
					createCMD = append(createCMD, test.addFlags...)
					output, err := pgo(createCMD...).Exec(t)
					t.Cleanup(func() {
						teardownCluster(t, namespace(), test.testName, time.Now())
					})
					require.NoError(t, err)
					require.Contains(t, output, "created cluster:")

					requireClusterReady(t, namespace(), test.testName, (2 * time.Minute))
					if contains(test.clusterFlags, "--pgbouncer") {
						requirePgBouncerReady(t, namespace(), test.testName, (2 * time.Minute))
					}

					t.Run("add", func(t *testing.T) {
						for _, deploymentName := range test.deployments {
							for expectedKey, expectedValue := range test.annotations {
								hasAnnotation := func() bool {
									actualAnnotations := TestContext.Kubernetes.GetDeployment(namespace(), deploymentName).Spec.Template.ObjectMeta.GetAnnotations()
									actualValue := actualAnnotations[expectedKey]
									if actualValue == expectedValue {
										return true
									}

									return false
								}

								requireWaitFor(t, hasAnnotation, time.Minute, time.Second,
									"timeout waiting for deployment \"%q\" to have annotation \"%s: %s\"", deploymentName, expectedKey, expectedValue)
							}
						}
					})

					t.Run("remove", func(t *testing.T) {
						t.Skip("Bug: annotation in not removed on update")
						updateCMD := []string{"update", "cluster", test.testName, "-n", namespace(), "--no-prompt"}
						updateCMD = append(updateCMD, test.removeFlags...)
						output, err := pgo(updateCMD...).Exec(t)
						require.NoError(t, err)
						require.Contains(t, output, "updated pgcluster")

						for _, deploymentName := range test.deployments {
							for expectedKey, _ := range test.annotations {
								notHasAnnotation := func() bool {
									actualAnnotations := TestContext.Kubernetes.GetDeployment(namespace(), deploymentName).Spec.Template.ObjectMeta.GetAnnotations()
									actualValue := actualAnnotations[expectedKey]
									if actualValue != "" {
										return false
									}

									return true
								}

								requireWaitFor(t, notHasAnnotation, time.Minute, time.Second,
									"timeout waiting for annotation key \"%s\" to be removed from deployment \"%s\"", expectedKey, deploymentName)
							}
						}
					})
				})
			}
		})

		t.Run("on update", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				testName     string
				annotations  map[string]string
				clusterFlags []string
				addFlags     []string
				deployments  []string
			}{
				{
					testName: "update-global",
					annotations: map[string]string{
						"global":  "here",
						"global2": "foo",
					},
					clusterFlags: []string{"--pgbouncer"},
					addFlags:     []string{"--annotation=global=here", "--annotation=global2=foo"},
					deployments:  []string{"update-global", "update-global-backrest-shared-repo", "update-global-pgbouncer"},
				}, {
					testName: "update-postgres",
					annotations: map[string]string{
						"postgres": "present",
					},
					clusterFlags: []string{},
					addFlags:     []string{"--annotation-postgres=postgres=present"},
					deployments:  []string{"update-postgres"},
				}, {
					testName: "update-pgbackrest",
					annotations: map[string]string{
						"pgbackrest": "what",
					},
					clusterFlags: []string{},
					addFlags:     []string{"--annotation-pgbackrest=pgbackrest=what"},
					deployments:  []string{"update-pgbackrest-backrest-shared-repo"},
				}, {
					testName: "update-pgbouncer",
					annotations: map[string]string{
						"pgbouncer": "aqui",
					},
					clusterFlags: []string{"--pgbouncer"},
					addFlags:     []string{"--annotation-pgbouncer=pgbouncer=aqui"},
					deployments:  []string{"update-pgbouncer-pgbouncer"},
				},
			}

			for _, test := range tests {
				test := test // lock test variable in for each run since it changes across parallel loops
				t.Run(test.testName, func(t *testing.T) {
					t.Parallel()
					createCMD := []string{"create", "cluster", test.testName, "-n", namespace()}
					createCMD = append(createCMD, test.clusterFlags...)
					output, err := pgo(createCMD...).Exec(t)
					t.Cleanup(func() {
						teardownCluster(t, namespace(), test.testName, time.Now())
					})
					require.NoError(t, err)
					require.Contains(t, output, "created cluster:")

					requireClusterReady(t, namespace(), test.testName, (2 * time.Minute))
					if contains(test.clusterFlags, "--pgbouncer") {
						requirePgBouncerReady(t, namespace(), test.testName, (2 * time.Minute))
					}

					updateCMD := []string{"update", "cluster", test.testName, "-n", namespace(), "--no-prompt"}
					updateCMD = append(updateCMD, test.addFlags...)
					output, err = pgo(updateCMD...).Exec(t)
					require.NoError(t, err)
					require.Contains(t, output, "updated pgcluster")

					t.Run("add", func(t *testing.T) {
						for _, deploymentName := range test.deployments {
							for expectedKey, expectedValue := range test.annotations {
								hasAnnotation := func() bool {
									actualAnnotations := TestContext.Kubernetes.GetDeployment(namespace(), deploymentName).Spec.Template.ObjectMeta.GetAnnotations()
									actualValue := actualAnnotations[expectedKey]
									if actualValue == expectedValue {
										return true
									}

									return false
								}

								requireWaitFor(t, hasAnnotation, time.Minute, time.Second,
									"timeout waiting for deployment \"%q\" to have annotation \"%s: %s\"", deploymentName, expectedKey, expectedValue)
							}
						}
					})
				})
			}
		})
	})
}
