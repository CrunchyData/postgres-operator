smoketest requires:

 * pass in the kubeconfig path on the command line using --kubeconfig=/path/to/kubeconfig
 * pass in the cluster name on the command line using -clustername flag
 * pass in the namespace on the command line using -namespace flag

To run a single test:

go test -run TestCreateLabel -v --kubeconfig=/home/jeffmc/.kube/config -clustername=foomatic -namespace=pgouser1

To run all tests:

go test ./... -v --kubeconfig=/home/jeffmc/.kube/config -clustername=foomatic -namespace=pgouser1
