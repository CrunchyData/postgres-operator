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

// The below constants contains the "apiserver RBAC permissions" -- this was
// reorganized to make it...slightly more organized as we continue to evole
// the system
const (
	// MISC
	APPLY_POLICY_PERM = "ApplyPolicy"
	CAT_PERM          = "Cat"
	DF_CLUSTER_PERM   = "DfCluster"
	LABEL_PERM        = "Label"
	RELOAD_PERM       = "Reload"
	RESTART_PERM      = "Restart"
	RESTORE_PERM      = "Restore"
	STATUS_PERM       = "Status"
	TEST_CLUSTER_PERM = "TestCluster"
	VERSION_PERM      = "Version"

	// CREATE
	CREATE_BACKUP_PERM    = "CreateBackup"
	CREATE_CLUSTER_PERM   = "CreateCluster"
	CREATE_DUMP_PERM      = "CreateDump"
	CREATE_FAILOVER_PERM  = "CreateFailover"
	CREATE_INGEST_PERM    = "CreateIngest"
	CREATE_NAMESPACE_PERM = "CreateNamespace"
	CREATE_PGADMIN_PERM   = "CreatePgAdmin"
	CREATE_PGBOUNCER_PERM = "CreatePgbouncer"
	CREATE_PGOUSER_PERM   = "CreatePgouser"
	CREATE_PGOROLE_PERM   = "CreatePgorole"
	CREATE_POLICY_PERM    = "CreatePolicy"
	CREATE_UPGRADE_PERM   = "CreateUpgrade"
	CREATE_USER_PERM      = "CreateUser"

	// RESTORE
	RESTORE_DUMP_PERM = "RestoreDump"

	// DELETE
	DELETE_BACKUP_PERM    = "DeleteBackup"
	DELETE_CLUSTER_PERM   = "DeleteCluster"
	DELETE_INGEST_PERM    = "DeleteIngest"
	DELETE_NAMESPACE_PERM = "DeleteNamespace"
	DELETE_PGADMIN_PERM   = "DeletePgAdmin"
	DELETE_PGBOUNCER_PERM = "DeletePgbouncer"
	DELETE_PGOROLE_PERM   = "DeletePgorole"
	DELETE_PGOUSER_PERM   = "DeletePgouser"
	DELETE_POLICY_PERM    = "DeletePolicy"
	DELETE_USER_PERM      = "DeleteUser"

	// SHOW
	SHOW_BACKUP_PERM          = "ShowBackup"
	SHOW_CLUSTER_PERM         = "ShowCluster"
	SHOW_CONFIG_PERM          = "ShowConfig"
	SHOW_INGEST_PERM          = "ShowIngest"
	SHOW_NAMESPACE_PERM       = "ShowNamespace"
	SHOW_PGADMIN_PERM         = "ShowPgAdmin"
	SHOW_PGBOUNCER_PERM       = "ShowPgBouncer"
	SHOW_PGOROLE_PERM         = "ShowPgorole"
	SHOW_PGOUSER_PERM         = "ShowPgouser"
	SHOW_POLICY_PERM          = "ShowPolicy"
	SHOW_PVC_PERM             = "ShowPVC"
	SHOW_SECRETS_PERM         = "ShowSecrets"
	SHOW_SYSTEM_ACCOUNTS_PERM = "ShowSystemAccounts"
	SHOW_USER_PERM            = "ShowUser"
	SHOW_WORKFLOW_PERM        = "ShowWorkflow"

	// SCALE
	SCALE_CLUSTER_PERM = "ScaleCluster"

	// UPDATE
	UPDATE_CLUSTER_PERM   = "UpdateCluster"
	UPDATE_NAMESPACE_PERM = "UpdateNamespace"
	UPDATE_PGBOUNCER_PERM = "UpdatePgBouncer"
	UPDATE_PGOROLE_PERM   = "UpdatePgorole"
	UPDATE_PGOUSER_PERM   = "UpdatePgouser"
	UPDATE_USER_PERM      = "UpdateUser"
)

var RoleMap map[string]map[string]string
var PermMap map[string]string

func initializePerms() {
	RoleMap = make(map[string]map[string]string)

	// ...initialize the permission map using most of the legacy method, but make
	// it slightly more organized
	PermMap = map[string]string{
		// MISC
		APPLY_POLICY_PERM: "yes",
		CAT_PERM:          "yes",
		DF_CLUSTER_PERM:   "yes",
		LABEL_PERM:        "yes",
		RELOAD_PERM:       "yes",
		RESTORE_PERM:      "yes",
		STATUS_PERM:       "yes",
		TEST_CLUSTER_PERM: "yes",
		VERSION_PERM:      "yes",

		// CREATE
		CREATE_BACKUP_PERM:    "yes",
		CREATE_DUMP_PERM:      "yes",
		CREATE_CLUSTER_PERM:   "yes",
		CREATE_FAILOVER_PERM:  "yes",
		CREATE_INGEST_PERM:    "yes",
		CREATE_NAMESPACE_PERM: "yes",
		CREATE_PGADMIN_PERM:   "yes",
		CREATE_PGBOUNCER_PERM: "yes",
		CREATE_PGOROLE_PERM:   "yes",
		CREATE_PGOUSER_PERM:   "yes",
		CREATE_POLICY_PERM:    "yes",
		CREATE_UPGRADE_PERM:   "yes",
		CREATE_USER_PERM:      "yes",

		// RESTORE
		RESTORE_DUMP_PERM: "yes",

		// DELETE
		DELETE_BACKUP_PERM:    "yes",
		DELETE_CLUSTER_PERM:   "yes",
		DELETE_INGEST_PERM:    "yes",
		DELETE_NAMESPACE_PERM: "yes",
		DELETE_PGADMIN_PERM:   "yes",
		DELETE_PGBOUNCER_PERM: "yes",
		DELETE_PGOROLE_PERM:   "yes",
		DELETE_PGOUSER_PERM:   "yes",
		DELETE_POLICY_PERM:    "yes",
		DELETE_USER_PERM:      "yes",

		// SHOW
		SHOW_BACKUP_PERM:          "yes",
		SHOW_CLUSTER_PERM:         "yes",
		SHOW_CONFIG_PERM:          "yes",
		SHOW_INGEST_PERM:          "yes",
		SHOW_NAMESPACE_PERM:       "yes",
		SHOW_PGADMIN_PERM:         "yes",
		SHOW_PGBOUNCER_PERM:       "yes",
		SHOW_PGOROLE_PERM:         "yes",
		SHOW_PGOUSER_PERM:         "yes",
		SHOW_POLICY_PERM:          "yes",
		SHOW_PVC_PERM:             "yes",
		SHOW_SECRETS_PERM:         "yes",
		SHOW_SYSTEM_ACCOUNTS_PERM: "yes",
		SHOW_USER_PERM:            "yes",
		SHOW_WORKFLOW_PERM:        "yes",

		// SCALE
		SCALE_CLUSTER_PERM: "yes",

		// UPDATE
		UPDATE_CLUSTER_PERM:   "yes",
		UPDATE_NAMESPACE_PERM: "yes",
		UPDATE_PGBOUNCER_PERM: "yes",
		UPDATE_PGOROLE_PERM:   "yes",
		UPDATE_PGOUSER_PERM:   "yes",
		UPDATE_USER_PERM:      "yes",
	}

	log.Infof("loading PermMap with %d Permissions\n", len(PermMap))

}
