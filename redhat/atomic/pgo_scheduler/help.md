= pgo-scheduler (1)
Crunchy Data
January 13, 2019

== NAME
pgo-scheduler - Lightweight cronservice for scheduling automated tasks such as backups and SQL 
jobs.

== DESCRIPTION
The pgo-scheduler image works in conjunction with Crunchy Data's PostgreSQL Operator to execute cron 
based tasks such as PostgreSQL backups or SQL policies against PostgreSQL clusters.

The container itself consists of:
    - RHEL7 base image
    - Scheduler application

Files added to the container during Docker build include: /help.1.

== USAGE
For more information on the PostgreSQL Operator, see the official Crunchy Postgres Operator repository on GitHub.

== LABELS
The starter container includes the following LABEL settings:

That atomic command runs the Docker command set in this label:

`Name=`

The registry location and name of the image. For example, Name="crunchydata/pgo-scheduler".

`Version=`

The Red Hat Enterprise Linux version from which the container was built. For example, Version="7.6"

`Release=`

The specific release number of the container. For example, Release="4.0.1"
