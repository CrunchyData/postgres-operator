= pgo-sqlrunner (1)
Crunchy Data
January 13, 2019

== NAME
pgo-sqlrunner - Lightweight microservice for loading SQL into a PostgreSQL cluster.

== DESCRIPTION
The pgo-sqlrunner image works in conjunction with Crunchy Data's PostgreSQL Operator to load SQL 
into PostgreSQL clusters using psql.

The container itself consists of:
    - RHEL7 base image
    - PostgreSQL psql tool

Files added to the container during Docker build include: /help.1.

== USAGE
For more information on the PostgreSQL Operator, see the official Crunchy Postgres Operator repository on GitHub.

== LABELS
The starter container includes the following LABEL settings:

That atomic command runs the Docker command set in this label:

`Name=`

The registry location and name of the image. For example, Name="crunchydata/pgo-sqlrunner".

`Version=`

The Red Hat Enterprise Linux version from which the container was built. For example, Version="7.6"

`Release=`

The specific release number of the container. For example, Release="4.0.1"
