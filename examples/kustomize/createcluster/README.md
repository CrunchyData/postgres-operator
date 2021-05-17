# create cluster
This is a working example that creates multiple clusters via the crd workflow using
kustomize.

## Prerequisites

### Postgres Operator
This example assumes you have the Crunchy PostgreSQL Operator installed
in a namespace called `pgo`.

### Kustomize
Install the latest [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) version available.  Kustomise is availble in kubectl but it will not be the latest version.

## Documenation
Please see the [documentation](https://access.crunchydata.com/documentation/postgres-operator/latest/custom-resources/) for more guidance using custom resources.

## Example set up and execution
Navigate to the createcluster directory under the examples/kustomize directory
```
cd ./examples/kustomize/createcluster/
```
In the createcluster directory you will see a base directory and an overlay directory. Base will create a simple crunchy data postgreSQL cluster.  There are 3 directories located in the overlay directory, dev, staging and prod.  You can run kustomize against each of those and a Crunchy PostgreSQL cluster will be created for each and each of them are slightly different.

### base
Lets generate the kustomize yaml for the base
```
kustomize build base/
```
If the yaml looks good lets apply it.
```
kustomize build base/ | kubectl apply -f -
```
You will see these items are created after running the above command
```
secret/hippo-hippo-secret created
secret/hippo-postgres-secret created
pgcluster.crunchydata.com/hippo created
```
You may need to wait a few seconds depending on the resources you have allocated to you kubernetes set up for the Crunchy PostgreSQL cluster to become available.

After the cluster is finished creating lets take a look at the cluster with the Crunchy PostgreSQL Operator
```
pgo show cluster hippo -n pgo
```
You will see something like this if successful:
```
cluster : hippo (crunchy-postgres-ha:centos8-13.3-4.7.0)
	pod : hippo-8fb6bd96-j87wq (Running) on gke-xxxx-default-pool-38e946bd-257w (1/1) (primary)
		pvc: hippo (1Gi)
	deployment : hippo
	deployment : hippo-backrest-shared-repo
	service : hippo - ClusterIP (10.0.56.86) - Ports (2022/TCP, 5432/TCP)
	labels : pgo-version=4.7.0 name=hippo crunchy-pgha-scope=hippo deployment-name=hippo pg-cluster=hippo pgouser=admin vendor=crunchydata
```
Feel free to run other pgo cli commands on the hippo cluster

### overlay
As mentioned above there are 3 overlays available in this example, these overlays will modify the common base.
#### development
The development overlay will deploy a simple Crunchy PostgreSQL cluster with pgbouncer

Lets generate the kustomize yaml for the dev overlay
```
kustomize build overlay/dev/
```
The yaml looks good now lets apply it
```
kustomize build overlay/dev/ | kubectl apply -f -
```
You will see these items are created after running the above command
```
secret/dev-hippo-hippo-secret created
secret/dev-hippo-postgres-secret created
pgcluster.crunchydata.com/dev-hippo created
```
After the cluster is finished creating lets take a look at the cluster with the Crunchy PostgreSQL Operator
```
pgo show cluster dev-hippo -n pgo
```
You will see something like this if successful:
```
cluster : dev-hippo (crunchy-postgres-ha:centos8-13.3-4.7.0)
	pod : dev-hippo-588d4cb746-bwrxb (Running) on gke-xxxx-default-pool-95cba91c-0ppp (1/1) (primary)
		pvc: dev-hippo (1Gi)
	deployment : dev-hippo
	deployment : dev-hippo-backrest-shared-repo
	deployment : dev-hippo-pgbouncer
	service : dev-hippo - ClusterIP (10.0.62.87) - Ports (2022/TCP, 5432/TCP)
	service : dev-hippo-pgbouncer - ClusterIP (10.0.48.120) - Ports (5432/TCP)
	labels : crunchy-pgha-scope=dev-hippo name=dev-hippo pg-cluster=dev-hippo vendor=crunchydata deployment-name=dev-hippo environment=development pgo-version=4.7.0 pgouser=admin
```
#### staging
The staging overlay will deploy a crunchy postgreSQL cluster with 2 replica's with annotations added

Lets generate the kustomize yaml for the staging overlay
```
kustomize build overlay/staging/
```
The yaml looks good now lets apply it
```
kustomize build overlay/staging/ | kubectl apply -f -
```
You will see these items are created after running the above command
```
secret/staging-hippo-hippo-secret created
secret/staging-hippo-postgres-secret created
pgcluster.crunchydata.com/staging-hippo created
pgreplica.crunchydata.com/staging-hippo-rpl1 created
```
After the cluster is finished creating lets take a look at the cluster with the crunchy postgreSQL operator
```
pgo show cluster staging-hippo -n pgo
```
You will see something like this if successful, (Notice one of the replicas is a different size):
```
cluster : staging-hippo (crunchy-postgres-ha:centos8-13.3-4.7.0)
	pod : staging-hippo-85cf6dcb65-9h748 (Running) on gke-xxxx-default-pool-95cba91c-0ppp (1/1) (primary)
		pvc: staging-hippo (1Gi)
	pod : staging-hippo-lnxw-cf47d8c8b-6r4wn (Running) on gke-xxxx-default-pool-21b7282d-rqkj (1/1) (replica)
		pvc: staging-hippo-lnxw (1Gi)
	pod : staging-hippo-rpl1-5d89d66f9b-44znd (Running) on gke-xxxx-default-pool-21b7282d-rqkj (1/1) (replica)
		pvc: staging-hippo-rpl1 (2Gi)
	deployment : staging-hippo
	deployment : staging-hippo-backrest-shared-repo
	deployment : staging-hippo-lnxw
	deployment : staging-hippo-rpl1
	service : staging-hippo - ClusterIP (10.0.56.253) - Ports (2022/TCP, 5432/TCP)
	service : staging-hippo-replica - ClusterIP (10.0.56.57) - Ports (2022/TCP, 5432/TCP)
	pgreplica : staging-hippo-lnxw
	pgreplica : staging-hippo-rpl1
	labels : deployment-name=staging-hippo environment=staging name=staging-hippo crunchy-pgha-scope=staging-hippo pg-cluster=staging-hippo pgo-version=4.7.0 pgouser=admin vendor=crunchydata
```

#### production
The production overlay will deploy a crunchy postgreSQL cluster with one replica

Lets generate the kustomize yaml for the prod overlay
```
kustomize build overlay/prod/
```
The yaml looks good now lets apply it
```
kustomize build overlay/prod/ | kubectl apply -f -
```
You will see these items are created after running the above command
```
secret/prod-hippo-hippo-secret created
secret/prod-hippo-postgres-secret created
pgcluster.crunchydata.com/prod-hippo created
```
After the cluster is finished creating lets take a look at the cluster with the crunchy postgreSQL operator
```
pgo show cluster prod-hippo -n pgo
```
You will see something like this if successful, (Notice one of the replicas is a different size):
```
cluster : prod-hippo (crunchy-postgres-ha:centos8-13.3-4.7.0)
	pod : prod-hippo-5d6dd46497-rr67c (Running) on gke-xxxx-default-pool-21b7282d-rqkj (1/1) (primary)
		pvc: prod-hippo (1Gi)
	pod : prod-hippo-flty-84d97c8769-2pzbh (Running) on gke-xxxx-default-pool-95cba91c-0ppp (1/1) (replica)
		pvc: prod-hippo-flty (1Gi)
	deployment : prod-hippo
	deployment : prod-hippo-backrest-shared-repo
	deployment : prod-hippo-flty
	service : prod-hippo - ClusterIP (10.0.56.18) - Ports (2022/TCP, 5432/TCP)
	service : prod-hippo-replica - ClusterIP (10.0.56.101) - Ports (2022/TCP, 5432/TCP)
	pgreplica : prod-hippo-flty
	labels : pgo-version=4.7.0 deployment-name=prod-hippo environment=production pg-cluster=prod-hippo crunchy-pgha-scope=prod-hippo name=prod-hippo pgouser=admin vendor=crunchydata
```
### Delete the clusters
To delete the clusters run the following pgo cli commands

To delete all the clusters in the `pgo` namespace run the following:
```
pgo delete cluster --all -n pgo
```
Or to delete each cluster individually
```
pgo delete cluster hippo -n pgo
pgo delete cluster dev-hippo -n pgo
pgo delete cluster staging-hippo -n pgo
pgo delete cluster prod-hippo -n pgo
```
