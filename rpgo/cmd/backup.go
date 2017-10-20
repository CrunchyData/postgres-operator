// Package cmd provides the command line functions of the crunchy CLI
package cmd

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

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"net/http"
)

// showBackup ....
func showBackup(args []string) {
	log.Debugf("showBackup called %v\n", args)
	if Namespace == "" {
		log.Error("Namespace can not be empty")
		return
	}

	//show pod information for job
	for _, v := range args {
		url := APIServerURL + "/backups/" + v + "?namespace=" + Namespace

		log.Debug("show backup called [" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			//log.Info("here after new req")
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

		var response msgs.ShowBackupResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if len(response.BackupList.Items) == 0 {
			fmt.Println("no backups found")
			return
		}

		log.Debugf("response = %v\n", response)
		log.Debugf("len of items = %d\n", len(response.BackupList.Items))

		for _, backup := range response.BackupList.Items {
			printBackupCRD(&backup)
		}

	}

}

// printBackupCRD ...
func printBackupCRD(result *crv1.Pgbackup) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgbackup : "+result.Spec.Name)

	fmt.Printf("%s%s\n", TreeBranch, "PVC Name:\t"+result.Spec.StorageSpec.Name)
	fmt.Printf("%s%s\n", TreeBranch, "PVC Access Mode:\t"+result.Spec.StorageSpec.AccessMode)
	fmt.Printf("%s%s\n", TreeBranch, "PVC Size:\t\t"+result.Spec.StorageSpec.Size)
	fmt.Printf("%s%s\n", TreeBranch, "CCPImageTag:\t"+result.Spec.CCPImageTag)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Status:\t"+result.Spec.BackupStatus)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Host:\t"+result.Spec.BackupHost)
	fmt.Printf("%s%s\n", TreeBranch, "Backup User:\t"+result.Spec.BackupUser)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Pass:\t"+result.Spec.BackupPass)
	fmt.Printf("%s%s\n", TreeTrunk, "Backup Port:\t"+result.Spec.BackupPort)

}
