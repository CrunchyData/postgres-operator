= pgo-load (1)
Jeff McCormick
August 17, 2018

== NAME
pgo-load - Loads a CSV or JSON file into a PostgreSQL database.

== DESCRIPTION
Works in conjunction with Crunchy Data's PostgreSQL Operator to load a CSV or JSON file into the database.

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

The registry location and name of the image. For example, Name="crunchydata/pgo-load".

`Version=`

The Red Hat Enterprise Linux version from which the container was built. For example, Version="7.6"

`Release=`

The specific release number of the container. For example, Release="4.0.1"
