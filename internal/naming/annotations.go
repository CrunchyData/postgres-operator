/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package naming

const (
	annotationPrefix = labelPrefix

	// Finalizer marks an object to be garbage collected by this module.
	Finalizer = annotationPrefix + "finalizer"

	// PatroniSwitchover is the annotation added to a PostgresCluster to initiate a manual
	// Patroni Switchover (or Failover).
	PatroniSwitchover = annotationPrefix + "trigger-switchover"

	// PGBackRestBackup is the annotation that is added to a PostgresCluster to initiate a manual
	// backup.  The value of the annotation will be a unique identifier for a backup Job (e.g. a
	// timestamp), which will be stored in the PostgresCluster status to properly track completion
	// of the Job.  Also used to annotate the backup Job itself as needed to identify the backup
	// ID associated with a specific manual backup Job.
	PGBackRestBackup = annotationPrefix + "pgbackrest-backup"

	// PGBackRestConfigHash is an annotation used to specify the hash value associated with a
	// repo configuration as needed to detect configuration changes that invalidate running Jobs
	// (and therefore must be recreated)
	PGBackRestConfigHash = annotationPrefix + "pgbackrest-hash"

	// PGBackRestCurrentConfig is an annotation used to indicate the name of the pgBackRest
	// configuration associated with a specific Job as determined by either the current primary
	// (if no dedicated repository host is enabled), or the dedicated repository host.  This helps
	// in detecting pgBackRest backup Jobs that no longer mount the proper pgBackRest
	// configuration, e.g. because a failover has occurred, or because dedicated repo host has been
	// enabled or disabled.
	PGBackRestCurrentConfig = annotationPrefix + "pgbackrest-config"

	// PGBackRestRestore is the annotation that is added to a PostgresCluster to initiate an in-place
	// restore.  The value of the annotation will be a unique identfier for a restore Job (e.g. a
	// timestamp), which will be stored in the PostgresCluster status to properly track completion
	// of the Job.
	PGBackRestRestore = annotationPrefix + "pgbackrest-restore"
)
