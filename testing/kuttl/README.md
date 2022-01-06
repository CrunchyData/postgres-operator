# Kuttl

## Installing
Docs for install: https://kuttl.dev/docs/cli.html#setup-the-kuttl-kubectl-plugin

Options:
  - Download and install the binary
  - Install the `kubectl krew` [plugin manager](https://github.com/kubernetes-sigs/krew)
    and `kubectl krew install kuttl`
## Cheat sheet

### Suppressing Noisy Logs

KUTTL gives you the option to suppress events from the test logging output. To enable this feature
update the `kuttl` parameter when calling the `make` target

```
KUTTL_TEST='kuttl test --suppress-log=events' make check-kuttl
```

To suppress the events permanently, you can add the following to the KUTTL config (kuttl-test.yaml)
```
suppress: 
- events
```

### Run test suite

Make sure that the operator is running in your kubernetes environment and that your `kubeconfig` is
set up. Then run the make target:

```
make check-kuttl
```

### Running a single test
A single test is considered to be one directory under `kuttl/e2e`, for example
`kuttl/e2e/restore` would run the `restore` test.

There are two ways to run a single test in isolation: 
- using an env var with the make target: `KUTTL_TEST='kuttl test --test <test-name>' make check-kuttl`
- using `kubectl kuttl --test` flag: `kubectl kuttl test testing/kuttl/e2e --test <test-name>`

### Writing additional tests

To make it easier to read tests, we want to put our `assert.yaml`/`errors.yaml` files after the
files that create/update the objects for a step. To achieve this, infix an extra `-` between the
step number and the object/step name.

For example, if the `00` test step wants to create a cluster and then assert that the cluster is ready,
the files would be named

```console
00--cluster.yaml # note the extra `-` to ensure that it sorts above the following file
00-assert.yaml
```