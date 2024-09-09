// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"k8s.io/apimachinery/pkg/labels"
)

const (
	labelPrefix = "postgres-operator.crunchydata.com/"

	// LabelCluster et al. provides the fundamental labels for Postgres instances
	LabelCluster     = labelPrefix + "cluster"
	LabelInstance    = labelPrefix + "instance"
	LabelInstanceSet = labelPrefix + "instance-set"

	// LabelRepoName is used to specify the name of a pgBackRest repository
	LabelRepoName = labelPrefix + "name"

	LabelPatroni = labelPrefix + "patroni"
	LabelRole    = labelPrefix + "role"

	// LabelClusterCertificate is used to identify a secret containing a cluster certificate
	LabelClusterCertificate = labelPrefix + "cluster-certificate"

	// LabelData is used to identify Pods and Volumes store Postgres data.
	LabelData = labelPrefix + "data"

	// LabelMoveJob is used to identify a directory move Job.
	LabelMoveJob = labelPrefix + "move-job"

	// LabelMovePGBackRestRepoDir is used to identify the Job that moves an existing pgBackRest repo directory.
	LabelMovePGBackRestRepoDir = labelPrefix + "move-pgbackrest-repo-dir"

	// LabelMovePGDataDir is used to identify the Job that moves an existing pgData directory.
	LabelMovePGDataDir = labelPrefix + "move-pgdata-dir"

	// LabelMovePGWalDir is used to identify the Job that moves an existing pg_wal directory.
	LabelMovePGWalDir = labelPrefix + "move-pgwal-dir"

	// LabelPGBackRest is used to indicate that a resource is for pgBackRest
	LabelPGBackRest = labelPrefix + "pgbackrest"

	// LabelPGBackRestBackup is used to indicate that a resource is for a pgBackRest backup
	LabelPGBackRestBackup = labelPrefix + "pgbackrest-backup"

	// LabelPGBackRestConfig is used to indicate that a ConfigMap or Secret is for pgBackRest
	LabelPGBackRestConfig = labelPrefix + "pgbackrest-config"

	// LabelPGBackRestDedicated is used to indicate that a ConfigMap is for a pgBackRest dedicated
	// repository host
	LabelPGBackRestDedicated = labelPrefix + "pgbackrest-dedicated"

	// LabelPGBackRestRepo is used to indicate that a Deployment or Pod is for a pgBackRest
	// repository
	LabelPGBackRestRepo = labelPrefix + "pgbackrest-repo"

	// LabelPGBackRestRepoVolume is used to indicate that a resource for a pgBackRest
	// repository
	LabelPGBackRestRepoVolume = labelPrefix + "pgbackrest-volume"

	LabelPGBackRestCronJob = labelPrefix + "pgbackrest-cronjob"

	// LabelPGBackRestRestore is used to indicate that a Job or Pod is for a pgBackRest restore
	LabelPGBackRestRestore = labelPrefix + "pgbackrest-restore"

	// LabelPGBackRestRestoreConfig is used to indicate that a configuration
	// resource (e.g. a ConfigMap or Secret) is for a pgBackRest restore
	LabelPGBackRestRestoreConfig = labelPrefix + "pgbackrest-restore-config"

	// LabelPGMonitorDiscovery is the label added to Pods running the "exporter" container to
	// support discovery by Prometheus according to pgMonitor configuration
	LabelPGMonitorDiscovery = labelPrefix + "crunchy-postgres-exporter"

	// LabelPostgresUser identifies the PostgreSQL user an object is for or about.
	LabelPostgresUser = labelPrefix + "pguser"

	// LabelStartupInstance is used to indicate the startup instance associated with a resource
	LabelStartupInstance = labelPrefix + "startup-instance"

	RolePrimary = "primary"
	RoleReplica = "replica"

	// RolePatroniLeader is the LabelRole that Patroni sets on the Pod that is
	// currently the leader.
	RolePatroniLeader = "master"

	// RolePatroniReplica is a LabelRole value that Patroni sets on Pods that are
	// following the leader.
	RolePatroniReplica = "replica"

	// RolePGBouncer is the LabelRole applied to PgBouncer objects.
	RolePGBouncer = "pgbouncer"

	// RolePGAdmin is the LabelRole applied to pgAdmin objects.
	RolePGAdmin = "pgadmin"

	// RolePostgresData is the LabelRole applied to PostgreSQL data volumes.
	RolePostgresData = "pgdata"

	// RolePostgresUser is the LabelRole applied to PostgreSQL user secrets.
	RolePostgresUser = "pguser"

	// RolePostgresWAL is the LabelRole applied to PostgreSQL WAL volumes.
	RolePostgresWAL = "pgwal"

	// RoleMonitoring is the LabelRole applied to Monitoring resources
	RoleMonitoring = "monitoring"
)

const (
	// LabelCrunchyBridgeClusterPostgresRole identifies the PostgreSQL user an object is for or about.
	LabelCrunchyBridgeClusterPostgresRole = labelPrefix + "cbc-pgrole"

	// RoleCrunchyBridgeClusterPostgresRole is the LabelRole applied to CBC PostgreSQL role secrets.
	RoleCrunchyBridgeClusterPostgresRole = "cbc-pgrole"
)

const (
	// DataPGAdmin is a LabelData value that indicates the object has pgAdmin data.
	DataPGAdmin = "pgadmin"

	// DataPGBackRest is a LabelData value that indicates the object has pgBackRest data.
	DataPGBackRest = "pgbackrest"

	// DataPostgres is a LabelData value that indicates the object has PostgreSQL data.
	DataPostgres = "postgres"
)

// BackupJobType represents different types of backups (e.g. ad-hoc backups, scheduled backups,
// the backup for pgBackRest replica creation, etc.)
type BackupJobType string

const (
	// BackupManual is the backup type utilized for manual backups
	BackupManual BackupJobType = "manual"

	// BackupReplicaCreate is the backup type for the backup taken to enable pgBackRest replica
	// creation
	BackupReplicaCreate BackupJobType = "replica-create"

	// BackupScheduled is the backup type utilized for scheduled backups
	BackupScheduled BackupJobType = "scheduled"
)

const (

	// LabelStandalonePGAdmin is used to indicate a resource for a standalone-pgadmin instance.
	LabelStandalonePGAdmin = labelPrefix + "pgadmin"
)

// Merge takes sets of labels and merges them. The last set
// provided will win in case of conflicts.
func Merge(sets ...map[string]string) labels.Set {
	merged := labels.Set{}
	for _, set := range sets {
		merged = labels.Merge(merged, set)
	}
	return merged
}

// DirectoryMoveJobLabels provides labels for PVC move Jobs.
func DirectoryMoveJobLabels(clusterName string) labels.Set {
	jobLabels := map[string]string{
		LabelCluster: clusterName,
		LabelMoveJob: "",
	}
	return jobLabels
}

// PGBackRestLabels provides common labels for pgBackRest resources.
func PGBackRestLabels(clusterName string) labels.Set {
	return map[string]string{
		LabelCluster:    clusterName,
		LabelPGBackRest: "",
	}
}

// PGBackRestBackupJobLabels provides labels for pgBackRest backup Jobs.
func PGBackRestBackupJobLabels(clusterName, repoName string,
	backupType BackupJobType) labels.Set {
	repoLabels := PGBackRestLabels(clusterName)
	jobLabels := map[string]string{
		LabelPGBackRestRepo:   repoName,
		LabelPGBackRestBackup: string(backupType),
	}
	return labels.Merge(jobLabels, repoLabels)
}

// PGBackRestBackupJobSelector provides a selector for querying all pgBackRest
// resources
func PGBackRestBackupJobSelector(clusterName, repoName string,
	backupType BackupJobType) labels.Selector {
	return PGBackRestBackupJobLabels(clusterName, repoName, backupType).AsSelector()
}

// PGBackRestRestoreConfigLabels provides labels for configuration (e.g. ConfigMaps and Secrets)
// generated to perform a pgBackRest restore.
//
// Deprecated: Store restore data in the pgBackRest ConfigMap and Secret,
// [PGBackRestConfig] and [PGBackRestSecret].
func PGBackRestRestoreConfigLabels(clusterName string) labels.Set {
	commonLabels := PGBackRestLabels(clusterName)
	jobLabels := map[string]string{
		LabelPGBackRestRestoreConfig: "",
	}
	return labels.Merge(jobLabels, commonLabels)
}

// PGBackRestRestoreConfigSelector provides selector for querying pgBackRest restore config
// resources.
func PGBackRestRestoreConfigSelector(clusterName string) labels.Selector {
	return PGBackRestRestoreConfigLabels(clusterName).AsSelector()
}

// PGBackRestRestoreJobLabels provides labels for pgBackRest restore Jobs and
// associated configuration ConfigMaps and Secrets.
func PGBackRestRestoreJobLabels(clusterName string) labels.Set {
	commonLabels := PGBackRestLabels(clusterName)
	jobLabels := map[string]string{
		LabelPGBackRestRestore: "",
	}
	return labels.Merge(jobLabels, commonLabels)
}

// PGBackRestRestoreJobSelector provides selector for querying pgBackRest restore Jobs.
func PGBackRestRestoreJobSelector(clusterName string) labels.Selector {
	return PGBackRestRestoreJobLabels(clusterName).AsSelector()
}

// PGBackRestRepoLabels provides common labels for pgBackRest repository
// resources.
func PGBackRestRepoLabels(clusterName, repoName string) labels.Set {
	commonLabels := PGBackRestLabels(clusterName)
	repoLabels := map[string]string{
		LabelPGBackRestRepo: repoName,
	}
	return labels.Merge(commonLabels, repoLabels)
}

// PGBackRestSelector provides a selector for querying all pgBackRest
// resources
func PGBackRestSelector(clusterName string) labels.Selector {
	return PGBackRestLabels(clusterName).AsSelector()
}

// PGBackRestConfigLabels provides labels for the pgBackRest configuration created and used by
// the PostgreSQL Operator
func PGBackRestConfigLabels(clusterName string) labels.Set {
	repoLabels := PGBackRestLabels(clusterName)
	operatorConfigLabels := map[string]string{
		LabelPGBackRestConfig: "",
	}
	return labels.Merge(repoLabels, operatorConfigLabels)
}

// PGBackRestCronJobLabels provides common labels for pgBackRest CronJobs
func PGBackRestCronJobLabels(clusterName, repoName, backupType string) labels.Set {
	commonLabels := PGBackRestLabels(clusterName)
	cronJobLabels := map[string]string{
		LabelPGBackRestRepo:    repoName,
		LabelPGBackRestCronJob: backupType,
		LabelPGBackRestBackup:  string(BackupScheduled),
	}
	return labels.Merge(commonLabels, cronJobLabels)
}

// PGBackRestDedicatedLabels provides labels for a pgBackRest dedicated repository host
func PGBackRestDedicatedLabels(clusterName string) labels.Set {
	commonLabels := PGBackRestLabels(clusterName)
	operatorConfigLabels := map[string]string{
		LabelPGBackRestDedicated: "",
	}
	return labels.Merge(commonLabels, operatorConfigLabels)
}

// PGBackRestDedicatedSelector provides a selector for querying pgBackRest dedicated
// repository host resources
func PGBackRestDedicatedSelector(clusterName string) labels.Selector {
	return PGBackRestDedicatedLabels(clusterName).AsSelector()
}

// PGBackRestRepoVolumeLabels the labels for a pgBackRest repository volume.
func PGBackRestRepoVolumeLabels(clusterName, repoName string) labels.Set {
	repoLabels := PGBackRestRepoLabels(clusterName, repoName)
	repoVolLabels := map[string]string{
		LabelPGBackRestRepoVolume: "",
		LabelData:                 DataPGBackRest,
	}
	return labels.Merge(repoLabels, repoVolLabels)
}

// StandalonePGAdminLabels return labels for standalone pgAdmin resources
func StandalonePGAdminLabels(pgAdminName string) labels.Set {
	return map[string]string{
		LabelStandalonePGAdmin: pgAdminName,
		LabelRole:              RolePGAdmin,
	}
}

// StandalonePGAdminSelector provides a selector for standalone pgAdmin resources
func StandalonePGAdminSelector(pgAdminName string) labels.Selector {
	return StandalonePGAdminLabels(pgAdminName).AsSelector()
}

// StandalonePGAdminDataLabels returns the labels for standalone pgAdmin resources
// that contain or mount data
func StandalonePGAdminDataLabels(pgAdminName string) labels.Set {
	return labels.Merge(
		StandalonePGAdminLabels(pgAdminName),
		map[string]string{
			LabelData: DataPGAdmin,
		},
	)
}

// StandalonePGAdminDataSelector returns a selector for standalone pgAdmin resources
// that contain or mount data
func StandalonePGAdminDataSelector(pgAdmiName string) labels.Selector {
	return StandalonePGAdminDataLabels(pgAdmiName).AsSelector()
}
