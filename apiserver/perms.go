package apiserver

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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

import (
	log "github.com/sirupsen/logrus"
)

// MISC
const CAT_PERM = "Cat"
const LS_PERM = "Ls"
const APPLY_POLICY_PERM = "ApplyPolicy"
const CLONE_PERM = "Clone"
const DF_CLUSTER_PERM = "DfCluster"
const LABEL_PERM = "Label"
const LOAD_PERM = "Load"
const RELOAD_PERM = "Reload"
const RESTORE_PERM = "Restore"
const STATUS_PERM = "Status"
const TEST_CLUSTER_PERM = "TestCluster"
const VERSION_PERM = "Version"

// CREATE
const CREATE_BACKUP_PERM = "CreateBackup"
const CREATE_BENCHMARK_PERM = "CreateBenchmark"
const CREATE_DUMP_PERM = "CreateDump"
const CREATE_CLUSTER_PERM = "CreateCluster"
const CREATE_FAILOVER_PERM = "CreateFailover"
const CREATE_INGEST_PERM = "CreateIngest"
const CREATE_PGBOUNCER_PERM = "CreatePgbouncer"
const CREATE_POLICY_PERM = "CreatePolicy"
const CREATE_SCHEDULE_PERM = "CreateSchedule"
const CREATE_UPGRADE_PERM = "CreateUpgrade"
const CREATE_USER_PERM = "CreateUser"
const CREATE_PGOUSER_PERM = "CreatePgouser"
const CREATE_PGOROLE_PERM = "CreatePgorole"
const CREATE_NAMESPACE_PERM = "CreateNamespace"

// RESTORE
const RESTORE_DUMP_PERM = "RestoreDump"
const RESTORE_PGBASEBACKUP_PERM = "RestorePgbasebackup"

// DELETE
const DELETE_BACKUP_PERM = "DeleteBackup"
const DELETE_BENCHMARK_PERM = "DeleteBenchmark"
const DELETE_CLUSTER_PERM = "DeleteCluster"
const DELETE_INGEST_PERM = "DeleteIngest"
const DELETE_PGBOUNCER_PERM = "DeletePgbouncer"
const DELETE_POLICY_PERM = "DeletePolicy"
const DELETE_SCHEDULE_PERM = "DeleteSchedule"
const DELETE_USER_PERM = "DeleteUser"
const DELETE_PGOUSER_PERM = "DeletePgouser"
const DELETE_PGOROLE_PERM = "DeletePgorole"
const DELETE_NAMESPACE_PERM = "DeleteNamespace"

// SHOW
const SHOW_BACKUP_PERM = "ShowBackup"
const SHOW_BENCHMARK_PERM = "ShowBenchmark"
const SHOW_CLUSTER_PERM = "ShowCluster"
const SHOW_CONFIG_PERM = "ShowConfig"
const SHOW_NAMESPACE_PERM = "ShowNamespace"
const SHOW_INGEST_PERM = "ShowIngest"
const SHOW_POLICY_PERM = "ShowPolicy"
const SHOW_PVC_PERM = "ShowPVC"
const SHOW_USER_PERM = "ShowUser"
const SHOW_WORKFLOW_PERM = "ShowWorkflow"
const SHOW_SCHEDULE_PERM = "ShowSchedule"
const SHOW_SECRETS_PERM = "ShowSecrets"
const SHOW_PGOUSER_PERM = "ShowPgouser"
const SHOW_PGOROLE_PERM = "ShowPgorole"

// UPDATE
const UPDATE_CLUSTER_PERM = "UpdateCluster"
const UPDATE_PGOUSER_PERM = "UpdatePgouser"
const UPDATE_USER_PERM = "UpdateUser"
const UPDATE_PGOROLE_PERM = "UpdatePgorole"
const UPDATE_NAMESPACE_PERM = "UpdateNamespace"

// SCALE
const SCALE_CLUSTER_PERM = "ScaleCluster"

var RoleMap map[string]map[string]string
var PermMap map[string]string

const pgorolePath = "/default-pgo-config/pgorole"
const pgoroleFile = "pgorole"

func InitializePerms() {
	PermMap = make(map[string]string)
	RoleMap = make(map[string]map[string]string)

	// MISC
	PermMap[APPLY_POLICY_PERM] = "yes"
	PermMap[DF_CLUSTER_PERM] = "yes"
	PermMap[LABEL_PERM] = "yes"
	PermMap[LOAD_PERM] = "yes"
	PermMap[CAT_PERM] = "yes"
	PermMap[LS_PERM] = "yes"
	PermMap[RELOAD_PERM] = "yes"
	PermMap[RESTORE_PERM] = "yes"
	PermMap[STATUS_PERM] = "yes"
	PermMap[TEST_CLUSTER_PERM] = "yes"
	PermMap[VERSION_PERM] = "yes"

	// Create
	PermMap[CREATE_BACKUP_PERM] = "yes"
	PermMap[CREATE_BENCHMARK_PERM] = "yes"
	PermMap[CREATE_DUMP_PERM] = "yes"
	PermMap[CREATE_CLUSTER_PERM] = "yes"
	PermMap[CREATE_FAILOVER_PERM] = "yes"
	PermMap[CREATE_INGEST_PERM] = "yes"
	PermMap[CREATE_PGBOUNCER_PERM] = "yes"
	PermMap[CREATE_POLICY_PERM] = "yes"
	PermMap[CREATE_SCHEDULE_PERM] = "yes"
	PermMap[CREATE_UPGRADE_PERM] = "yes"
	PermMap[CREATE_USER_PERM] = "yes"
	PermMap[CREATE_PGOUSER_PERM] = "yes"
	PermMap[CREATE_PGOROLE_PERM] = "yes"
	PermMap[CREATE_NAMESPACE_PERM] = "yes"
	// RESTORE
	PermMap[RESTORE_DUMP_PERM] = "yes"
	PermMap[RESTORE_PGBASEBACKUP_PERM] = "yes"
	// Delete
	PermMap[DELETE_BACKUP_PERM] = "yes"
	PermMap[DELETE_BENCHMARK_PERM] = "yes"
	PermMap[DELETE_CLUSTER_PERM] = "yes"
	PermMap[DELETE_INGEST_PERM] = "yes"
	PermMap[DELETE_PGBOUNCER_PERM] = "yes"
	PermMap[DELETE_POLICY_PERM] = "yes"
	PermMap[DELETE_SCHEDULE_PERM] = "yes"
	PermMap[DELETE_USER_PERM] = "yes"
	PermMap[DELETE_PGOUSER_PERM] = "yes"
	PermMap[DELETE_PGOROLE_PERM] = "yes"
	PermMap[DELETE_NAMESPACE_PERM] = "yes"
	// Show
	PermMap[SHOW_BACKUP_PERM] = "yes"
	PermMap[SHOW_BENCHMARK_PERM] = "yes"
	PermMap[SHOW_CLUSTER_PERM] = "yes"
	PermMap[SHOW_CONFIG_PERM] = "yes"
	PermMap[SHOW_NAMESPACE_PERM] = "yes"
	PermMap[SHOW_INGEST_PERM] = "yes"
	PermMap[SHOW_POLICY_PERM] = "yes"
	PermMap[SHOW_PVC_PERM] = "yes"
	PermMap[SHOW_USER_PERM] = "yes"
	PermMap[SHOW_WORKFLOW_PERM] = "yes"
	PermMap[SHOW_SCHEDULE_PERM] = "yes"
	PermMap[SHOW_SECRETS_PERM] = "yes"
	PermMap[SHOW_PGOUSER_PERM] = "yes"
	PermMap[SHOW_PGOROLE_PERM] = "yes"

	// Scale
	PermMap[SCALE_CLUSTER_PERM] = "yes"

	// Update
	PermMap[UPDATE_CLUSTER_PERM] = "yes"
	PermMap[UPDATE_PGOUSER_PERM] = "yes"
	PermMap[UPDATE_USER_PERM] = "yes"
	PermMap[UPDATE_PGOROLE_PERM] = "yes"
	PermMap[UPDATE_NAMESPACE_PERM] = "yes"

	log.Infof("loading PermMap with %d Permissions\n", len(PermMap))

}

func HasPerm(role string, perm string) bool {
	if RoleMap[role][perm] == "yes" {
		return true
	}
	return false
}
