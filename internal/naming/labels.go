/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

import (
	"k8s.io/apimachinery/pkg/labels"
)

const (
	labelPrefix = "postgres-operator.crunchydata.com/"

	LabelCluster     = labelPrefix + "cluster"
	LabelInstance    = labelPrefix + "instance"
	LabelInstanceSet = labelPrefix + "instance-set"

	// LabelRepoName is used to specify the name of a pgBackRest repository
	LabelRepoName = labelPrefix + "name"

	LabelPatroni = labelPrefix + "patroni"
	LabelRole    = labelPrefix + "role"

	// LabelClusterCertificate is used to identify a secret containing a cluster certificate
	LabelClusterCertificate = labelPrefix + "cluster-certificate"

	// LabelPGBackRest is used to indicate that a resource is for pgBackRest
	LabelPGBackRest = labelPrefix + "pgbackrest"

	// LabelPGBackRestBackup is used to indicate that a resource is for a pgBackRest backup
	LabelPGBackRestBackup = labelPrefix + "pgbackrest-backup"

	// LabelPGBackRestConfig is used to indicate that a ConfigMap is for pgBackRest
	LabelPGBackRestConfig = labelPrefix + "pgbackrest-config"

	// LabelPGBackRestDedicated is used to indicate that a ConfigMap is for a pgBackRest dedicated
	// repository host
	LabelPGBackRestDedicated = labelPrefix + "pgbackrest-dedicated"

	// LabelPGBackRestRepo is used to indicate that a Deployment or Pod is for a pgBackRest
	// repository
	LabelPGBackRestRepo = labelPrefix + "pgbackrest-repo"

	// LabelPGBackRestRepoHost is used to indicate that a resource is for a pgBackRest
	// repository host
	LabelPGBackRestRepoHost = labelPrefix + "pgbackrest-host"

	// LabelPGBackRestRepoVolume is used to indicate that a resource for a pgBackRest
	// repository
	LabelPGBackRestRepoVolume = labelPrefix + "pgbackrest-volume"

	LabelPGBackRestCronJob = labelPrefix + "pgbackrest-cronjob"

	// LabelPGBackRestRestore is used to indicate that a Job or Pod is for a pgBackRest restore
	LabelPGBackRestRestore = labelPrefix + "pgbackrest-restore"

	// LabelStartupInstance is used to indicate the startup instance associated with a resource
	LabelStartupInstance = labelPrefix + "startup-instance"

	// LabelUserSecret is used to identify the secret containing the Postgres
	// user connection information
	LabelUserSecret = labelPrefix + "pguser"

	RolePrimary = "primary"
	RoleReplica = "replica"

	// Patroni sets this LabelRole value on the Pod that is currently leader.
	RolePatroniLeader = "master"

	// Patroni sets this LabelRole value on Pods that are following the leader.
	RolePatroniReplica = "replica"

	// RolePGBouncer is the LabelRole applied to PgBouncer objects.
	RolePGBouncer = "pgbouncer"

	// RolePostgresData is the LabelRole applied to PostgreSQL data volumes.
	RolePostgresData = "pgdata"

	// RolePostgresWAL is the LabelRole applied to PostgreSQL WAL volumes.
	RolePostgresWAL = "pgwal"
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

// PGBackRestSelector provides a selector for querying all pgBackRest
// resources
func PGBackRestBackupJobSelector(clusterName, repoName string,
	backupType BackupJobType) labels.Selector {
	return PGBackRestBackupJobLabels(clusterName, repoName, backupType).AsSelector()
}

// PGBackRestRestoreJobLabels provides labels for pgBackRest restore Jobs.
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
	}
	return labels.Merge(commonLabels, cronJobLabels)
}

// PGBackRestDedicatedLabels provides labels for a pgBackRest dedicated repository host
func PGBackRestDedicatedLabels(clusterName string) labels.Set {
	repoLabels := PGBackRestRepoHostLabels(clusterName)
	operatorConfigLabels := map[string]string{
		LabelPGBackRestDedicated: "",
	}
	return labels.Merge(repoLabels, operatorConfigLabels)
}

// PGBackRestDedicatedSelector provides a selector for querying pgBackRest dedicated
// repository host resources
func PGBackRestDedicatedSelector(clusterName string) labels.Selector {
	return PGBackRestDedicatedLabels(clusterName).AsSelector()
}

// PGBackRestRepoHostLabels the labels for a pgBackRest repository host.
func PGBackRestRepoHostLabels(clusterName string) labels.Set {
	commonLabels := PGBackRestLabels(clusterName)
	repoHostLabels := map[string]string{
		LabelPGBackRestRepoHost: "",
	}
	return labels.Merge(commonLabels, repoHostLabels)
}

// PGBackRestRepoVolumeLabels the labels for a pgBackRest repository volume.
func PGBackRestRepoVolumeLabels(clusterName, repoName string) labels.Set {
	repoLabels := PGBackRestRepoLabels(clusterName, repoName)
	repoVolLabels := map[string]string{
		LabelPGBackRestRepoVolume: "",
	}
	return labels.Merge(repoLabels, repoVolLabels)
}
