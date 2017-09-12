/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

// Package cmd provides the command line functions of the crunchy CLI
package cmd

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/backupservice"
	"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "perform a Backup",
	Long: `BACKUP performs a Backup, for example:
			pgo backup mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("backup called")
		if len(args) == 0 {
			fmt.Println(`You must specify the cluster to backup.`)
		} else {
			createBackup(args)
		}

	},
}

func init() {
	RootCmd.AddCommand(backupCmd)
}

func showBackup(args []string) {
	log.Debugf("showBackup called %v\n", args)

	//show pod information for job
	for _, arg := range args {

		url := "http://localhost:8080/backups/somename?showsecrets=true&other=thing"
		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}
		defer resp.Body.Close()
		var response backupservice.ShowBackupResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Println(err)
		}
		fmt.Println("Name = ", response.Items[0].Name)

		log.Debug("show backup called for " + arg)

	}

}

func createBackup(args []string) {
	log.Debugf("createBackup called %v\n", args)

	//var err error

	for _, arg := range args {
		log.Debug("create backup called for " + arg)

		fmt.Println("created PgBackup " + arg)

	}

}

func deleteBackup(args []string) {
	log.Debugf("deleteBackup called %v\n", args)
	var err error
	backupList := tpr.PgBackupList{}
	err = Tprclient.Get().Resource(tpr.BACKUP_RESOURCE).Do().Into(&backupList)
	if err != nil {
		log.Error("error getting backup list")
		log.Error(err.Error())
		return
	}
	// delete the pgbackup resource instance
	// which will cause the operator to remove the related Job
	for _, arg := range args {
		backupFound := false
		for _, backup := range backupList.Items {
			if arg == "all" || backup.Spec.Name == arg {
				backupFound = true
				err = Tprclient.Delete().
					Resource(tpr.BACKUP_RESOURCE).
					Namespace(Namespace).
					Name(backup.Spec.Name).
					Do().
					Error()
				if err != nil {
					log.Error("error deleting pgbackup " + arg)
					log.Error(err.Error())
				}
				fmt.Println("deleted pgbackup " + backup.Spec.Name)
			}

		}
		if !backupFound {
			fmt.Println("backup " + arg + " not found")
		}

	}

}

func getBackupParams(name string) (*tpr.PgBackup, error) {
	var newInstance *tpr.PgBackup

	storageSpec := tpr.PgStorageSpec{}
	spec := tpr.PgBackupSpec{}
	spec.Name = name
	spec.StorageSpec = storageSpec
	spec.StorageSpec.PvcName = viper.GetString("BACKUP_STORAGE.PVC_NAME")
	spec.StorageSpec.PvcAccessMode = viper.GetString("BACKUP_STORAGE.PVC_ACCESS_MODE")
	spec.StorageSpec.PvcSize = viper.GetString("BACKUP_STORAGE.PVC_SIZE")
	spec.StorageSpec.StorageClass = viper.GetString("BACKUP_STORAGE.STORAGE_CLASS")
	spec.StorageSpec.StorageType = viper.GetString("BACKUP_STORAGE.STORAGE_TYPE")
	spec.StorageSpec.SUPPLEMENTAL_GROUPS = viper.GetString("BACKUP_STORAGE.SUPPLEMENTAL_GROUPS")
	spec.StorageSpec.FSGROUP = viper.GetString("BACKUP_STORAGE.FSGROUP")
	spec.CCP_IMAGE_TAG = viper.GetString("CLUSTER.CCP_IMAGE_TAG")
	spec.BACKUP_STATUS = "initial"
	spec.BACKUP_HOST = "basic"
	spec.BACKUP_USER = "master"
	spec.BACKUP_PASS = "password"
	spec.BACKUP_PORT = "5432"

	cluster := tpr.PgCluster{}
	err := Tprclient.Get().
		Resource(tpr.CLUSTER_RESOURCE).
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&cluster)
	if err == nil {
		spec.BACKUP_HOST = cluster.Spec.Name
		//spec.BACKUP_USER = cluster.Spec.PG_MASTER_USER
		//spec.BACKUP_PASS = cluster.Spec.PG_MASTER_PASSWORD
		spec.BACKUP_PASS = GetMasterSecretPassword(cluster.Spec.Name)
		spec.BACKUP_PORT = cluster.Spec.Port
	} else if errors.IsNotFound(err) {
		log.Debug(name + " is not a cluster")
		return newInstance, err
	} else {
		log.Error("error getting pgcluster " + name)
		log.Error(err.Error())
		return newInstance, err
	}

	newInstance = &tpr.PgBackup{
		Metadata: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance, nil
}

type PodTemplateFields struct {
	Name         string
	CO_IMAGE_TAG string
	BACKUP_ROOT  string
	PVC_NAME     string
}
