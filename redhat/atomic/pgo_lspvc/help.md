= pgo-lspvc (1)
Jeff McCormick
August 17, 2018

== NAME
pgo-lspvc - Display contents of a PVC.

== DESCRIPTION
The pgo-lspvc image works in conjunction with Crunchy Data's PostgreSQL Operator to display the contents of a PVC using the ls command.

The container itself consists of:
    - RHEL7 base image
    - Command executing ls command against the target container

Files added to the container during Docker build include: /help.1.

== USAGE
For more information on the PostgreSQL Operator, see the official Crunchy Postgres Operator repository on GitHub.

== LABELS
The starter container includes the following LABEL settings:

That atomic command runs the Docker command set in this label:

`Name=`

The registry location and name of the image. For example, Name="crunchydata/pgo-lspvc".

`Version=`

The Red Hat Enterprise Linux version from which the container was built. For example, Version="7.5"

`Release=`

The specific release number of the container. For example, Release="3.2.0"
