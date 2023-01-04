<!--
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->

PostgreSQL commits transactions by storing changes in its [write-ahead log][WAL].
The contents of the log are applied to data files (containing tables and indexes)
later as part of a checkpoint.

The way WAL files are accessed and utilized often differs from that of data
files. In high-performance situations, it can desirable to put WAL files on
storage with different performance or durability characteristics.

[WAL]: https://www.postgresql.org/docs/current/wal.html


PostgresCluster has a field that specifies how to store PostgreSQL data files
and an optional field for how to store PostgreSQL WAL files. When a WAL volume
is specified the PostgresCluster controller reconciles one "pgwal" PVC per
instance in the instance set.

## Starting with a WAL volume

When a PostgresCluster is created with a WAL volume specified, the `--waldir`
argument to `initdb` ensures that WAL files are written to WAL volume. When
creating a replica (e.g. scaling up) the `--waldir` argument to `pg_basebackup`
does the same. The way pgBackRest handles this depends on the contents of the
backup, but when creating a replica the `--link-map=pg_wal` argument does the
same.

## Adding a WAL volume

It is possible to specify a WAL volume on PostgresCluster after it has already
bootstrapped, has data, etc. In this case, the WAL PVC is reconciled and mounted
as usual and an init container moves any existing WAL files while PostgreSQL is
stopped. These are changes to the instance PodTemplate and go through the normal
rollout procedure.

## Removing a WAL volume

It is possible to remove the specification of a WAL volume on PostgresCluster
after it has already bootstrapped, has data, etc. In this case, a series of
rollouts moves WAL files off the volume then unmounts and deletes the PVC.

First, the command of the init container is adjusted to match the PostgresCluster
spec -- WAL files belong *off* the WAL volume. The WAL PVC continues to exist
and remains mounted in the PodSpec. This change to the PodTemplate is rolled
out, allowing the init container to move WAL files off the WAL volume while
PostgreSQL is stopped.

When the PostgreSQL container of an instance Pod starts running, the
PostgresCluster controller examines the WAL directory inside its volume. When
the WAL files are safely off the WAL volume, it deletes the WAL PVC and removes
it from the PodSpec. This change to the PodTemplate goes through the normal
rollout procedure.

