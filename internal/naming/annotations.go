// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

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

	// PGBackRestBackupJobCompletion is the annotation that is added to restore jobs, pvcs, and
	// VolumeSnapshots that are involved in the volume snapshot creation process. The annotation
	// holds a RFC3339 formatted timestamp that corresponds to the completion time of the associated
	// backup job.
	PGBackRestBackupJobCompletion = annotationPrefix + "pgbackrest-backup-job-completion"

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
	// restore.  The value of the annotation will be a unique identifier for a restore Job (e.g. a
	// timestamp), which will be stored in the PostgresCluster status to properly track completion
	// of the Job.
	PGBackRestRestore = annotationPrefix + "pgbackrest-restore"

	// PGBackRestIPVersion is an annotation used to indicate whether an IPv6 wildcard address should be
	// used for the pgBackRest "tls-server-address" or not. If the user wants to use IPv6, the value
	// should be "IPv6". As of right now, if the annotation is not present or if the annotation's value
	// is anything other than "IPv6", the "tls-server-address" will default to IPv4 (0.0.0.0). The need
	// for this annotation is due to an issue in pgBackRest (#1841) where using a wildcard address to
	// bind all addresses does not work in certain IPv6 environments.
	PGBackRestIPVersion = annotationPrefix + "pgbackrest-ip-version"

	// PostgresExporterCollectorsAnnotation is an annotation used to allow users to control whether or
	// not postgres_exporter default metrics, settings, and collectors are enabled. The value "None"
	// disables all postgres_exporter defaults. Disabling the defaults may cause errors in dashboards.
	PostgresExporterCollectorsAnnotation = annotationPrefix + "postgres-exporter-collectors"

	// CrunchyBridgeClusterAdoptionAnnotation is an annotation used to allow users to "adopt" or take
	// control over an existing Bridge Cluster with a CrunchyBridgeCluster CR. Essentially, if a
	// CrunchyBridgeCluster CR does not have a status.ID, but the name matches the name of an existing
	// bridge cluster, the user must add this annotation to the CR to allow the CR to take control of
	// the Bridge Cluster. The Value assigned to the annotation must be the ID of existing cluster.
	CrunchyBridgeClusterAdoptionAnnotation = annotationPrefix + "adopt-bridge-cluster"

	// AutoCreateUserSchemaAnnotation is an annotation used to allow users to control whether the cluster
	// has schemas automatically created for the users defined in `spec.users` for all of the databases
	// listed for that user.
	AutoCreateUserSchemaAnnotation = annotationPrefix + "autoCreateUserSchema"

	// AuthorizeBackupRemovalAnnotation is an annotation used to allow users
	// to delete PVC-based backups when changing from a cluster with backups
	// to a cluster without backups. As usual with the operator, we do not
	// touch cloud-based backups.
	AuthorizeBackupRemovalAnnotation = annotationPrefix + "authorizeBackupRemoval"

	// Used from Kubernetes v1.21+ to define a default container used when the
	// `-c` flag is not passed.
	// --https://kubernetes.io/docs/reference/labels-annotations-taints/#kubectl-kubernetes-io-default-container
	DefaultContainerAnnotation = "kubectl.kubernetes.io/default-container"
)
