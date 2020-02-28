
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

The following will run the entire suite printing the names of tests as it goes.
The results for every test will appear after the very last test finishes.

```sh
cd testing/pgo_cli
GO111MODULE=on go test -count=1 -parallel=2 -timeout=30m -v .
```

Use the [`-run` argument][go-test-run] to select some subset of tests:

```sh
go test -run '///failover'
```

Keep in mind that the suite uses whatever `pgo` executable is on your `PATH`.
To use a recently built one, for example:

```sh
cd testing/pgo_cli
PATH="$(dirname $(dirname $(pwd)))/bin:$PATH"
```

[go-test-run]: https://pkg.go.dev/testing#hdr-Subtests_and_Sub_benchmarks
