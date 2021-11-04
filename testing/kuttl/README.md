# Kuttl

## Installing
Docs for install: https://kuttl.dev/docs/cli.html#setup-the-kuttl-kubectl-plugin

Options:
  - Download and install the binary
  - Install the `kubectl krew` [plugin manager](https://github.com/kubernetes-sigs/krew)
    and `kubectl krew install kuttl`
## Cheat sheet

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
