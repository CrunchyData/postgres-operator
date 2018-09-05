package apiserver

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	"os"
	"strings"
)

const RESTORE_PERM = "Restore"
const SHOW_SECRETS_PERM = "ShowSecrets"
const RELOAD_PERM = "Reload"
const SHOW_CONFIG_PERM = "ShowConfig"
const DF_CLUSTER_PERM = "DfCluster"
const SHOW_CLUSTER_PERM = "ShowCluster"
const CREATE_CLUSTER_PERM = "CreateCluster"
const TEST_CLUSTER_PERM = "TestCluster"
const DELETE_CLUSTER_PERM = "DeleteCluster"
const SHOW_BACKUP_PERM = "ShowBackup"
const CREATE_BACKUP_PERM = "CreateBackup"
const DELETE_BACKUP_PERM = "DeleteBackup"
const LABEL_PERM = "Label"
const LOAD_PERM = "Load"
const CREATE_POLICY_PERM = "CreatePolicy"
const DELETE_POLICY_PERM = "DeletePolicy"
const SHOW_POLICY_PERM = "ShowPolicy"
const APPLY_POLICY_PERM = "ApplyPolicy"
const SHOW_PVC_PERM = "ShowPVC"
const CREATE_UPGRADE_PERM = "CreateUpgrade"
const SHOW_UPGRADE_PERM = "ShowUpgrade"
const DELETE_UPGRADE_PERM = "DeleteUpgrade"
const USER_PERM = "User"
const CREATE_USER_PERM = "CreateUser"
const DELETE_USER_PERM = "DeleteUser"
const VERSION_PERM = "Version"
const CREATE_INGEST_PERM = "CreateIngest"
const SHOW_INGEST_PERM = "ShowIngest"
const DELETE_INGEST_PERM = "DeleteIngest"
const CREATE_FAILOVER_PERM = "CreateFailover"
const STATUS_PERM = "Status"

var RoleMap map[string]map[string]string
var PermMap map[string]string

const pgorolePath = "/config/pgorole"

func InitializePerms() {
	PermMap = make(map[string]string)
	RoleMap = make(map[string]map[string]string)

	PermMap[SHOW_SECRETS_PERM] = "yes"
	PermMap[RELOAD_PERM] = "yes"
	PermMap[SHOW_CONFIG_PERM] = "yes"
	PermMap[STATUS_PERM] = "yes"
	PermMap[DF_CLUSTER_PERM] = "yes"
	PermMap[SHOW_CLUSTER_PERM] = "yes"
	PermMap[CREATE_CLUSTER_PERM] = "yes"
	PermMap[DELETE_CLUSTER_PERM] = "yes"
	PermMap[TEST_CLUSTER_PERM] = "yes"
	PermMap[SHOW_BACKUP_PERM] = "yes"
	PermMap[CREATE_BACKUP_PERM] = "yes"
	PermMap[DELETE_BACKUP_PERM] = "yes"
	PermMap[LABEL_PERM] = "yes"
	PermMap[LOAD_PERM] = "yes"
	PermMap[CREATE_POLICY_PERM] = "yes"
	PermMap[DELETE_POLICY_PERM] = "yes"
	PermMap[SHOW_POLICY_PERM] = "yes"
	PermMap[APPLY_POLICY_PERM] = "yes"
	PermMap[SHOW_PVC_PERM] = "yes"
	PermMap[CREATE_UPGRADE_PERM] = "yes"
	PermMap[SHOW_UPGRADE_PERM] = "yes"
	PermMap[DELETE_UPGRADE_PERM] = "yes"
	PermMap[USER_PERM] = "yes"
	PermMap[CREATE_USER_PERM] = "yes"
	PermMap[DELETE_USER_PERM] = "yes"
	PermMap[VERSION_PERM] = "yes"
	PermMap[CREATE_INGEST_PERM] = "yes"
	PermMap[SHOW_INGEST_PERM] = "yes"
	PermMap[DELETE_INGEST_PERM] = "yes"
	PermMap[CREATE_FAILOVER_PERM] = "yes"
	PermMap[RESTORE_PERM] = "yes"
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

	f, err := os.Open(pgorolePath)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Error(err)
		os.Exit(2)
	}

	for _, line := range lines {
		if len(line) == 0 {

		} else {
			fields := strings.Split(strings.TrimSpace(line), ":")
			if len(fields) != 2 {
				log.Infoln("rolename:perm format not followed")
				log.Error(errors.New("invalid format found in pgorole, role:perm format must be followed"))
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
						log.Errorf(" [%s] not a valid permission for role [%s]\n", cleanPerm, roleName)
						os.Exit(2)
					}
					permMap[cleanPerm] = "yes"
				}
				RoleMap[roleName] = permMap
				log.Infof("loaded Role [%s] Perms Ct [%d] Perms [%v]\n", roleName, len(permMap), permMap)
			}
		}
	}

}
