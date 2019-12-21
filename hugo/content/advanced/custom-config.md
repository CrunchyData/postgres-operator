---
title: "Custom Configuration"
date:
draft: false
weight: 4
---

## Custom Postgres Configurations

Users and administrators can specify a
custom set of Postgres configuration files be used when creating
a new Postgres cluster.  The configuration files you can
change include -

 * postgresql.conf
 * pg_hba.conf
 * setup.sql

Different configurations for PostgreSQL might be defined for
the following -

 * OLTP types of databases
 * OLAP types of databases
 * High Memory
 * Minimal Configuration for Development
 * Project Specific configurations
 * Special Security Requirements

#### Global ConfigMap

If you create a *configMap* called *pgo-custom-pg-config* with any
of the above files within it, new clusters will use those configuration
files when setting up a new database instance.  You do *NOT* have to
specify all of the configuration files. It is entirely up to your use case
to determine which to use.

An example set of configuration files and a script to create the
global configMap is found at 
```
$PGOROOT/examples/custom-config
```

If you run the *create.sh* script there, it will create the configMap
that will include the PostgreSQL configuration files within that directory.

#### Config Files Purpose

The *postgresql.conf* file is the main Postgresql configuration file that allows
the definition of a wide variety of tuning parameters and features.

The *pg_hba.conf* file is the way Postgresql secures client access.

The *setup.sql* file is a Crunchy Container Suite configuration
file used to initially populate the database after the initial *initdb*
is run when the database is first created. Changes would be made
to this if you wanted to define which database objects are created by
default.

#### Granular Config Maps

Granular config maps can be defined if it is necessary to use
a different set of configuration files for different clusters
rather than having a single configuration (e.g. Global Config Map).
A specific set of ConfigMaps with their own set of PostgreSQL
configuration files can be created. When creating new clusters, a
`--custom-config` flag can be passed along with the name of the
ConfigMap which will be used for that specific cluster or set of
clusters.

#### Defaults

If there is  no reason to change the default PostgreSQL configuration
files that ship with the Crunchy Postgres container, there is  no
requirement to. In this event, continue using the Operator as usual
and avoid defining a global configMap.



