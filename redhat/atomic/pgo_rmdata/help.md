= pgo-rmdata (1)
Jeff McCormick
August 17, 2018

== NAME
pgo-rmdata - Removes the contents of the PostgreSQL data directory.

== DESCRIPTION
The pgo-rmdata image works in conjunction with Crunchy Data's PostgreSQL Operator to remove the contents of the PostgreSQL data directory.

The container itself consists of:
    - RHEL7 base image
    - Command that executes the forced recursive removal of data in the PGDATA directory

Files added to the container during Docker build include: /help.1.

== USAGE
For more information on the PostgreSQL Operator, see the official Crunchy Postgres Operator repository on GitHub.

== LABELS
The starter container includes the following LABEL settings:

That atomic command runs the Docker command set in this label:

`Name=`

The registry location and name of the image. For example, Name="crunchydata/pgo-rmdata".

`Version=`

The Red Hat Enterprise Linux version from which the container was built. For example, Version="7.6"

`Release=`

The specific release number of the container. For example, Release="4.0.1"
