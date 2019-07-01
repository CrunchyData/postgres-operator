package apiserver

/*
Copyright 2019 Crunchy Data Solutions, Inc.
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
	"bufio"
	"errors"
	"os"
	"strings"

	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
)

// MISC
const CAT_PERM = "Cat"
const LS_PERM = "Ls"
const APPLY_POLICY_PERM = "ApplyPolicy"
const DF_CLUSTER_PERM = "DfCluster"
const LABEL_PERM = "Label"
const LOAD_PERM = "Load"
const RELOAD_PERM = "Reload"
const RESTORE_PERM = "Restore"
const STATUS_PERM = "Status"
const TEST_CLUSTER_PERM = "TestCluster"
const USER_PERM = "User"
const VERSION_PERM = "Version"

// CREATE
const CREATE_BACKUP_PERM = "CreateBackup"
const CREATE_BENCHMARK_PERM = "CreateBenchmark"
const CREATE_DUMP_PERM = "CreateDump"
const CREATE_CLUSTER_PERM = "CreateCluster"
const CREATE_FAILOVER_PERM = "CreateFailover"
const CREATE_INGEST_PERM = "CreateIngest"
const CREATE_PGBOUNCER_PERM = "CreatePgbouncer"
const CREATE_PGPOOL_PERM = "CreatePgpool"
const CREATE_POLICY_PERM = "CreatePolicy"
const CREATE_SCHEDULE_PERM = "CreateSchedule"
const CREATE_UPGRADE_PERM = "CreateUpgrade"
const CREATE_USER_PERM = "CreateUser"

// RESTORE
const RESTORE_DUMP_PERM = "RestoreDump"
const RESTORE_PGBASEBACKUP_PERM = "RestorePgbasebackup"

// DELETE
const DELETE_BACKUP_PERM = "DeleteBackup"
const DELETE_BENCHMARK_PERM = "DeleteBenchmark"
const DELETE_CLUSTER_PERM = "DeleteCluster"
const DELETE_INGEST_PERM = "DeleteIngest"
const DELETE_PGBOUNCER_PERM = "DeletePgbouncer"
const DELETE_PGPOOL_PERM = "DeletePgpool"
const DELETE_POLICY_PERM = "DeletePolicy"
const DELETE_SCHEDULE_PERM = "DeleteSchedule"
const DELETE_USER_PERM = "DeleteUser"

// SHOW
const SHOW_BACKUP_PERM = "ShowBackup"
const SHOW_BENCHMARK_PERM = "ShowBenchmark"
const SHOW_CLUSTER_PERM = "ShowCluster"
const SHOW_CONFIG_PERM = "ShowConfig"
const SHOW_NAMESPACE_PERM = "ShowNamespace"
const SHOW_INGEST_PERM = "ShowIngest"
const SHOW_POLICY_PERM = "ShowPolicy"
const SHOW_PVC_PERM = "ShowPVC"
const SHOW_WORKFLOW_PERM = "ShowWorkflow"
const SHOW_SCHEDULE_PERM = "ShowSchedule"
const SHOW_SECRETS_PERM = "ShowSecrets"

// UPDATE
const UPDATE_CLUSTER_PERM = "UpdateCluster"

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
	PermMap[USER_PERM] = "yes"
	PermMap[VERSION_PERM] = "yes"
	// Create
	PermMap[CREATE_BACKUP_PERM] = "yes"
	PermMap[CREATE_BENCHMARK_PERM] = "yes"
	PermMap[CREATE_DUMP_PERM] = "yes"
	PermMap[CREATE_CLUSTER_PERM] = "yes"
	PermMap[CREATE_FAILOVER_PERM] = "yes"
	PermMap[CREATE_INGEST_PERM] = "yes"
	PermMap[CREATE_PGBOUNCER_PERM] = "yes"
	PermMap[CREATE_PGPOOL_PERM] = "yes"
	PermMap[CREATE_POLICY_PERM] = "yes"
	PermMap[CREATE_SCHEDULE_PERM] = "yes"
	PermMap[CREATE_UPGRADE_PERM] = "yes"
	PermMap[CREATE_USER_PERM] = "yes"
	// RESTORE
	PermMap[RESTORE_DUMP_PERM] = "yes"
	PermMap[RESTORE_PGBASEBACKUP_PERM] = "yes"
	// Delete
	PermMap[DELETE_BACKUP_PERM] = "yes"
	PermMap[DELETE_BENCHMARK_PERM] = "yes"
	PermMap[DELETE_CLUSTER_PERM] = "yes"
	PermMap[DELETE_INGEST_PERM] = "yes"
	PermMap[DELETE_PGBOUNCER_PERM] = "yes"
	PermMap[DELETE_PGPOOL_PERM] = "yes"
	PermMap[DELETE_POLICY_PERM] = "yes"
	PermMap[DELETE_SCHEDULE_PERM] = "yes"
	PermMap[DELETE_USER_PERM] = "yes"
	// Show
	PermMap[SHOW_BACKUP_PERM] = "yes"
	PermMap[SHOW_BENCHMARK_PERM] = "yes"
	PermMap[SHOW_CLUSTER_PERM] = "yes"
	PermMap[SHOW_CONFIG_PERM] = "yes"
	PermMap[SHOW_NAMESPACE_PERM] = "yes"
	PermMap[SHOW_INGEST_PERM] = "yes"
	PermMap[SHOW_POLICY_PERM] = "yes"
	PermMap[SHOW_PVC_PERM] = "yes"
	PermMap[SHOW_WORKFLOW_PERM] = "yes"
	PermMap[SHOW_SCHEDULE_PERM] = "yes"
	PermMap[SHOW_SECRETS_PERM] = "yes"

	// Scale
	PermMap[SCALE_CLUSTER_PERM] = "yes"

	// Update
	PermMap[UPDATE_CLUSTER_PERM] = "yes"
	log.Infof("loading PermMap with %d Permissions\n", len(PermMap))

	readRoles()
}

func HasPerm(role string, perm string) bool {
	if RoleMap[role][perm] == "yes" {
		return true
	}
	return false
}

func readRoles() {
	var err error
	var lines []string
	var scanner *bufio.Scanner

	cm, found := kubeapi.GetConfigMap(Clientset, config.CustomConfigMapName, PgoNamespace)
	if found {
		log.Infof("Config: %s ConfigMap found in ns %s, using config files from the configmap", config.CustomConfigMapName, PgoNamespace)

		val := cm.Data[pgoroleFile]
		if val == "" {
			log.Infof("could not find %s in ConfigMap", pgoroleFile)
			os.Exit(2)
		}

		log.Infof("Custom %s file found in configmap", pgoroleFile)
		scanner = bufio.NewScanner(strings.NewReader(val))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		err = scanner.Err()
	} else {
		log.Infof("No custom %s file found in configmap, using defaults", pgoroleFile)
		f, err := os.Open(pgorolePath)
		if err != nil {
			log.Error(err)
			os.Exit(2)
		}
		defer f.Close()

		scanner = bufio.NewScanner(f)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		err = scanner.Err()
	}

	if err != nil {
		log.Error(err)
		os.Exit(2)
	}

	for _, line := range lines {
		if len(line) == 0 {

		} else {
			fields := strings.Split(strings.TrimSpace(line), ":")
			if len(fields) != 2 {
				log.Infoln("rolename:permission format not followed")
				log.Error(errors.New("invalid format found in pgorole - rolename:permission format must be followed"))
				log.Errorf("bad line is %s\n", fields)
				os.Exit(2)
			} else {
				roleName := fields[0]
				permsArray := fields[1]
				perms := strings.Split(strings.TrimSpace(permsArray), ",")
				permMap := make(map[string]string)
				for _, v := range perms {
					cleanPerm := strings.TrimSpace(v)
					if PermMap[cleanPerm] == "" {
						log.Errorf(" [%s] not a valid permission for role [%s]", cleanPerm, roleName)
						os.Exit(2)
					}
					permMap[cleanPerm] = "yes"
				}
				RoleMap[roleName] = permMap
				log.Infof("loaded Role [%s] Perms Ct [%d] Perms [%v]", roleName, len(permMap), permMap)
			}
		}
	}

}
