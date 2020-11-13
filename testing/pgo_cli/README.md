
This directory contains a suite of basic regression scenarios that exercise the
PostgreSQL Operator through the PostgreSQL Operator Client. It uses the
[`testing` package](https://pkg.go.dev/testing) of Go and can be driven using `go test`.


## Configuration

The environment variables of the `go test` process are passed to `pgo` so it can
be tested under different configurations.

When `PGO_OPERATOR_NAMESPACE` is set, some of the [variables that affect `pgo`][pgo-env]
are given defaults:

- `PGO_CA_CERT`, `PGO_CLIENT_CERT`, and `PGO_CLIENT_KEY` default to paths under
  `~/.pgo/${PGO_OPERATOR_NAMESPACE}/output` which is usually populated by the
  Ansible installer.

- `PGO_APISERVER_URL` defaults to a random local port that forwards to the
  PostgreSQL Operator API using the same mechanism as `kubectl port-forward`.

When `PGO_NAMESPACE` is set, any objects created by tests will appear there.
When it is not set, each test creates a new random namespace, runs there, and
deletes it afterward. These random namespaces all have the `pgo-test` label.

`PGO_TEST_TIMEOUT_SCALE` can be used to adjust the amount of time a test waits
for asynchronous events to take place. A setting of `1.2` waits 20% longer than
usual.

[pgo-env]: ../../docs/content/pgo-client/_index.md#global-environment-variables


### Kubernetes

The suite expects to be able to call the Kubernetes API that is running the
Operator. If it cannot find a [kubeconfig][] file in the typical places, it will
try the [in-cluster API][k8s-in-cluster]. Use the `KUBECONFIG` environment
variable to configure the Kubernetes API client.

[k8s-in-cluster]: https://pkg.go.dev/k8s.io/client-go/rest#InClusterConfig
[kubeconfig]: https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/


## Execution

The suite uses whatever `pgo` executable is on your `PATH`. To use a recently
built one you can set your `PATH` with the following:

```sh
cd testing/pgo_cli
PATH="$(dirname $(dirname $(pwd)))/bin:$PATH"
```

The following will run the entire suite printing the names of tests as it goes.
The results for every test will appear as they finish.

```sh
cd testing/pgo_cli
GO111MODULE=on go test -count=1 -parallel=2 -timeout=30m -v .
```

Use the [`-run` argument][go-test-run] to select some subset of tests:

```sh
go test -run '/failover'
```

[go-test-run]: https://pkg.go.dev/testing#hdr-Subtests_and_Sub_benchmarks


## Test Descriptions

**operator_test.go** executes the pgo version, pgo status and pgo show config commands and verifies the correct responses are returned.

**cluster_create_test.go** executes the pgo create cluster, pgo show workflow, pgo show cluster and pgo show user commands.  The test will create a cluster named mycluster and verify the cluster is created. This test will also verify correct responses are returned for the pgo show workflow command as well as pgo show cluster and pgo show user commands.

**cluster_label_test.go** executes the pgo label and various pgo show cluster commands as well as the pgo delete label command. This test will add a label to a cluster then exercise various ways to show the cluster via the label as well as verify the label was applied to the cluster. Pgo delete label is also executed and verifies the label is successfully removed from the cluster.

**cluster_test_test.go** executes the pgo test command and verifies the cluster services and instances are "UP"

**operator_rbac_test.go** executes various pgo pgouser and pgo pgorole commands. This test verifies operator user creation scenarios which include creating an operator user and assigning roles and namespace access and verifying operator users can only access namespaces they are assigned to as well as being able to run commands that are assigned to them via the pgo pgorole command.

**cluster_user_test.go** executes various pgo create user, pgo show user, pgo update user and pgo delete commands.  This test verifies the operator creates a PostgreSQL user correctly as well as showing the correct user data and updates the user correctly.  This test also verifies a variety of flags that are passed in with the create update and delete user commands.

**cluster_df_test.go** executed the pgo df command and verifies the capacity of a cluster is returned.

**cluster_policy_test.go** executes the policy functionality utilizing the commands pgo create policy, pgo apply policy, and pgo delete policy as well as various flags and will verify the appropriate values are returned and created, updated, applied or deleted.

**cluster_scale_test.go** executes pgo scale which will scale the cluster with an additional replica.  This test will verify the cluster has successfully scaled the cluster up and verify the replica is available and ready.

**cluster_pgbouncer_test.go** executes pgo create pgbouncer to onboard a pgbouncer and verifies pgbouncer has been added and is available and running.  This test also executes the pgo show cluster command as well to verify pgbouncer has been onboarded as well as pgo test to ensure all of the clusters services and instances are "UP" and available.  Lastly, this test will remove the pgbouncer from the cluster by running the pgo delete pgbouncer command and verify pgbouncer was indeed removed from the cluster.

**cluster_delete_test.go** executes the pgo delete command with scenarios such as completely delete the cluster including all of the backups and delete the cluster but keep the backup data and verifies the deletions occurred.

<!-- markdownlint-disable-file MD012 MD041 -->
