
you can test this program outside of a container like so:

cd $PGOROOT

go run ./pgo-rmdata/pgo-rmdata.go -pg-cluster=mycluster -replica-name= -namespace=mynamespace -remove-data=true -remove-backup=true -is-replica=false -is-backup=false
