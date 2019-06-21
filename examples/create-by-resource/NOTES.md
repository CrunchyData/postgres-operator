
# Example of Creating a Postgres Cluster With Resources

Instead of creating a Postgres cluster using the pgo API (REST),
some users require a way to do this with Kube resource definitions
directly using the kubectl or oc commands or APIs.

This example creates a Postgres cluster called *fromcrd* using
a pgcluster CRD plus some Secrets.

## Postgres Secrets

Postgres passwords are not stored in the pgcluster CRD, but rather
secret names for the Postgres users are stored in the CRD.  The
following Secrets are required to be created for any Postgres
cluster used by the Operator:
 * postgres-secret.yaml
 * primaryuser-secret.yaml
 * testuser-secret.yaml

Those 3 Postgres credentials are referenced by the pgcluster CRD
for this cluster.

To use passwords other than the example default passwords, see
the following link https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/#convert-your-secret-data-to-a-base-64-representation

## pgcluster CRD

To cause the Operator to create a Postgres cluster, it is looking
for a pgcluster CRD to be created.  The following file contains
the pgcluster CRD used by this example:
 * fromcrd.json

## Test the Example
You can run the following script to cause the secrets
and CRD to be created:
 * run.sh

The Operator should show the new cluster started:

    jeffmc@~ > pgo show cluster fromcrd

    cluster : fromcrd (centos7-11.4-2.4.1)
    	pod : fromcrd-6b4d69df46-4s7bn (Running) on doppio-kube (1/1) (primary)
	pvc : fromcrd
	resources : CPU Limit= Memory Limit=, CPU Request= Memory Request=
	storage : Primary=1G Replica=1G
	deployment : fromcrd
	service : fromcrd - ClusterIP (10.97.101.79)
	labels : pg-cluster=fromcrd pgo-backrest=false primary=true archive=false deployment-name=fromcrd name=fromcrd current-primary=fromcrd pgo-version=4.0.1 archive-timeout=60 crunchy-pgbadger=false crunchy_collect=false 

Notice the user credentials we created:

    jeffmc@~ > pgo show user fromcrd

    cluster : fromcrd

    secret : fromcrd-postgres-secret
	username: postgres
	password: 3zAyzf18qB

    secret : fromcrd-primaryuser-secret
	username: primaryuser
	password: wFoaiQdXOM

    secret : fromcrd-testuser-secret
	username: testuser
	password: PNq8EEU0qM

Lastly, lets see if the cluster responds:

    jeffmc@~ > pgo test fromcrd

    cluster : fromcrd 
	psql -p 5432 -h 10.97.101.79 -U postgres postgres is Working
	psql -p 5432 -h 10.97.101.79 -U postgres userdb is Working
	psql -p 5432 -h 10.97.101.79 -U primaryuser postgres is Working
	psql -p 5432 -h 10.97.101.79 -U primaryuser userdb is Working
	psql -p 5432 -h 10.97.101.79 -U testuser postgres is Working
	psql -p 5432 -h 10.97.101.79 -U testuser userdb is Working
