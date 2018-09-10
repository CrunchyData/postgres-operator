// Package cmd provides the command line functions of the crunchy CLI
package cmd

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
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	labelutil "github.com/crunchydata/postgres-operator/util"
	"github.com/spf13/cobra"
	"os"
)

var PVCName string

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Perform a Backup",
	Long: `BACKUP performs a Backup, for example:

  pgo backup mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("backup called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: You must specify the cluster to backup or a selector flag.`)
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				if BackupType == labelutil.LABEL_BACKUP_TYPE_BACKREST {
					createBackrestBackup(args, BackrestOpts)
				} else if BackupType == labelutil.LABEL_BACKUP_TYPE_BASEBACKUP {
					createBackup(args)
				} else {
					fmt.Println("Error: You must specify either pgbasebackup or pgbackrest for the --backup-type.")
				}
			} else {
				fmt.Println("Aborting...")
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	backupCmd.Flags().StringVarP(&PVCName, "pvc-name", "", "", "The PVC name to use for the backup instead of the default.")
	backupCmd.Flags().StringVarP(&StorageConfig, "storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster storage.")
	backupCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "No command line confirmation.")
	backupCmd.Flags().StringVarP(&BackupType, "backup-type", "", "", "The backup type to perform. Default is pgbasebackup, and both pgbasebackup and pgbackrest are valid backup types.")
	backupCmd.Flags().StringVarP(&BackrestOpts, "pgbackrest-opts", "", "", "The pgbackrest backup options to pass.")

}

// showBackup ....
func showBackup(args []string) {
	log.Debugf("showBackup called %v\n", args)

	//show pod information for job
	for _, v := range args {
		response, err := api.ShowBackup(httpclient, v, &SessionCredentials)

		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

		if len(response.BackupList.Items) == 0 {
			fmt.Println("No backups found.")
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
	fmt.Printf("%s%s\n", TreeBranch, "Access Mode:\t"+result.Spec.StorageSpec.AccessMode)
	fmt.Printf("%s%s\n", TreeBranch, "PVC Size:\t"+result.Spec.StorageSpec.Size)
	fmt.Printf("%s%s\n", TreeBranch, "Creation:\t"+result.ObjectMeta.CreationTimestamp.String())
	fmt.Printf("%s%s\n", TreeBranch, "CCPImageTag:\t"+result.Spec.CCPImageTag)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Status:\t"+result.Spec.BackupStatus)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Host:\t"+result.Spec.BackupHost)
	fmt.Printf("%s%s\n", TreeBranch, "Backup User:\t"+result.Spec.BackupUser)
	fmt.Printf("%s%s\n", TreeTrunk, "Backup Port:\t"+result.Spec.BackupPort)

}

// deleteBackup ....
func deleteBackup(args []string) {
	log.Debugf("deleteBackup called %v\n", args)

	for _, v := range args {
		response, err := api.DeleteBackup(httpclient, v, &SessionCredentials)

		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code == msgs.Ok {
			if len(response.Results) == 0 {
				fmt.Println("No backups found.")
				return
			}
			for k := range response.Results {
				fmt.Println("Deleted backup " + response.Results[k])
			}
		} else {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

	}

}

// createBackup ....
func createBackup(args []string) {
	log.Debugf("createBackup called %v\n", args)

	request := new(msgs.CreateBackupRequest)
	request.Args = args
	request.Selector = Selector
	request.PVCName = PVCName
	request.StorageConfig = StorageConfig

	response, err := api.CreateBackup(httpclient, &SessionCredentials, request)

	if err != nil {
		fmt.Println("Error: " + response.Status.Msg)
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
