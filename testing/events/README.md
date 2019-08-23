eventtest requires:

 * pass in the kubeconfig path on the command line using --kubeconfig=/path/to/kubeconfig
 * pass in the cluster name on the command line using -clustername flag
 * pass in the namespace on the command line using -namespace flag
 * pass in the username on the command line using -username flag
 * pass in the rolename on the command line using -rolename flag
 * pass in the endpoint for the event router -event-tcp-address="127.0.0.1:14150"

You can port-forward to the event router as follows:

  kubectl port-forward pod/postgres-operator-79dfddf5bc-6hlqz 14150:4150 -n pgo

This port-forward creates a localhost port at 14150 that maps to the 
pgo-event container port at 4150.


To run a single test:

go test -run TestEventCreate -v --kubeconfig=/home/jeffmc/.kube/config -clustername=foomatic -namespace=pgouser1

To run all tests:

go test ./... -v --kubeconfig=/home/jeffmc/.kube/config -clustername=foomatic -namespace=pgouser1

