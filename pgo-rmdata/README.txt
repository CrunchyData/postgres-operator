
you can test this program outside of a container like so:

cd $PGOROOT

go run ./pgo-rmdata/pgo-rmdata.go --kubeconfig=/home/$USER/.kube/config -pg-cluster=mycluster -namespace=mynamespace -remove-data=true -remove-backup=true -is-replica=false -is-backup=false
