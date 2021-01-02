// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/cmd/pgo/api"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

// createBackrestBackup ....
func createBackrestBackup(args []string, ns string) {
	log.Debugf("createBackrestBackup called %v %s", args, BackupOpts)

	request := new(msgs.CreateBackrestBackupRequest)
	request.Namespace = ns
	request.Args = args
	request.Selector = Selector
	request.BackupOpts = BackupOpts
	request.BackrestStorageType = BackrestStorageType

	response, err := api.CreateBackrestBackup(httpclient, &SessionCredentials, request)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.Results) == 0 {
		fmt.Println("No clusters found.")
		return
	}
}

// showBackrest ....
func showBackrest(args []string, ns string) {
	log.Debugf("showBackrest called %v", args)

	for _, v := range args {
		response, err := api.ShowBackrest(httpclient, v, Selector, &SessionCredentials, ns)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

		if len(response.Items) == 0 {
			fmt.Println("No pgBackRest found.")
			return
		}

		log.Debugf("response = %v", response)
		log.Debugf("len of items = %d", len(response.Items))

		for i := range response.Items {
			printBackrest(&response.Items[i])
		}
	}
}

// printBackrest
func printBackrest(result *msgs.ShowBackrestDetail) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("cluster: %s\n", result.Name)
	fmt.Printf("storage type: %s\n\n", result.StorageType)

	for _, info := range result.Info {
		fmt.Printf("stanza: %s\n", info.Name)
		fmt.Printf("    status: %s\n", info.Status.Message)
		fmt.Printf("    cipher: %s\n\n", info.Cipher)

		for _, archive := range info.Archives {
			// this is the quick way of getting the name...alternatively we could look
			// it up by ID
			fmt.Printf("    %s (current)\n", info.Name)
			fmt.Printf("        wal archive min/max (%s)\n\n", archive.ID)

			// iterate trhough the the backups and list out all the information
			for _, backup := range info.Backups {
				databaseSize, databaseUnit := getSizeAndUnit(backup.Info.Size)
				databaseBackupSize, databaseBackupUnit := getSizeAndUnit(backup.Info.Delta)
				repositorySize, repositoryUnit := getSizeAndUnit(backup.Info.Repository.Size)
				repositoryBackupSize, repositoryBackupUnit := getSizeAndUnit(backup.Info.Repository.Delta)

				// this matches the output format of pgbackrest info
				fmt.Printf("        %s backup: %s\n", backup.Type, backup.Label)
				fmt.Printf("            timestamp start/stop: %s / %s\n",
					time.Unix(backup.Timestamp.Start, 0),
					time.Unix(backup.Timestamp.Stop, 0))
				fmt.Printf("            wal start/stop: %s / %s\n",
					backup.Archive.Start, backup.Archive.Stop)
				fmt.Printf("            database size: %.1f%s, backup size: %.1f%s\n",
					databaseSize, getUnitString(databaseUnit),
					databaseBackupSize, getUnitString(databaseBackupUnit))
				fmt.Printf("            repository size: %.1f%s, repository backup size: %.1f%s\n",
					repositorySize, getUnitString(repositoryUnit),
					repositoryBackupSize, getUnitString(repositoryBackupUnit))
				fmt.Printf("            backup reference list: %s\n\n",
					strings.Join(backup.Reference, ", "))
			}
		}
	}
}
