= pgo-apiserver (1)
Jeff McCormick
August 17, 2018

== NAME
pgo-apiserver - The REST API container image for Crunchy Data's PostgreSQL Operator

== DESCRIPTION
The Postgres Operator is a controller, written in Golang, that uses the Kubernetes API and CustomResourceDefinition concepts to offer users a CLI which enables them to create and manage PostgreSQL databases and clusters running on a Kubernetes cluster.

The apiserver REST API allows users to interact freely with the Operator using a command line interface.

The container itself consists of:
    - RHEL7 base image
    - Bash script that performs the container startup
    - PostgreSQL binary packages

Files added to the container during Docker build include: /help.1.

== USAGE
For more information on the PostgreSQL Operator, see the official Crunchy Postgres Operator repository on GitHub.

== LABELS
The starter container includes the following LABEL settings:

That atomic command runs the Docker command set in this label:

`Name=`

The registry location and name of the image. For example, Name="crunchydata/pgo-apiserver".

`Version=`

The Red Hat Enterprise Linux version from which the container was built. For example, Version="7.7"

`Release=`

The specific release number of the container. For example, Release="4.1.0"
