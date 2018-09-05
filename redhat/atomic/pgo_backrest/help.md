= pgo-backrest (1)
Jeff McCormick
August 17, 2018

== NAME
pgo-backrest - performs pgbackrest commands on a database container

== DESCRIPTION
Works in conjunction with Crunchy Data's PostgreSQL Operator to perform pgbackrest commands on a database container

The container itself consists of:
    - RHEL7 base image
    - Bash script that performs the container startup

Files added to the container during Docker build include: /help.1.

== USAGE
For more information on the PostgreSQL Operator, see the official Crunchy Postgres Operator repository on GitHub.

== LABELS
The starter container includes the following LABEL settings:

That atomic command runs the Docker command set in this label:

`Name=`

The registry location and name of the image. For example, Name="crunchydata/pgo-backrest".

`Version=`

The Red Hat Enterprise Linux version from which the container was built. For example, Version="7.5"

`Release=`

The specific release number of the container. For example, Release="3.2.0"
