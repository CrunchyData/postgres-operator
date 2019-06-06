// Package cmd provides the command line functions of the crunchy CLI
package cmd

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
	"fmt"
	"os"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var PVCName string

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Perform a Backup",
	Long: `BACKUP performs a Backup, for example:

  pgo backup mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}

		log.Debug("backup called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: You must specify the cluster to backup or a selector flag.`)
		} else {

			exitNow := false // used in switch for early exit.

			switch buSelected := backupType; buSelected {

			case config.LABEL_BACKUP_TYPE_BACKREST:

				// storage config flag invalid for backrest
				if StorageConfig != "" {
					fmt.Println("Error: --storage-config is not allowed when performing a pgbackrest backup.")
					exitNow = true
				}

				if exitNow {
					return
				}

				createBackrestBackup(args, Namespace)

			case "", config.LABEL_BACKUP_TYPE_BASEBACKUP:

				createBackup(args, Namespace)

			case config.LABEL_BACKUP_TYPE_PGDUMP:

				createpgDumpBackup(args, Namespace)

			default:
				fmt.Println("Error: You must specify either pgbasebackup, pgbackrest, or pgdump for the --backup-type.")

			}

		}

	},
}

var backupType string

func init() {
	RootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&BackupOpts, "backup-opts", "", "", "The pgbackup options to pass into pgbasebackup or pgbackrest.")
	backupCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	backupCmd.Flags().StringVarP(&PVCName, "pvc-name", "", "", "The PVC name to use for the backup instead of the default.")
	backupCmd.Flags().StringVarP(&StorageConfig, "storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster storage.  Only applies to pgbasebackup backups.")
	backupCmd.Flags().StringVar(&backupType, "backup-type", "pgbackrest", "The backup type to perform. Default is pgbasebackup. Valid backup types are pgbasebackup, pgbackrest and pgdump.")
	backupCmd.Flags().StringVarP(&BackrestStorageType, "pgbackrest-storage-type", "", "", "The type of storage to use when scheduling pgBackRest backups. Either \"local\", \"s3\" or both, comma separated. (default \"local\")")

}

// showBackup ....
func showBackup(args []string, ns string) {
	log.Debugf("showBackup called %v", args)

	//show pod information for job
	for _, v := range args {
		response, err := api.ShowBackup(httpclient, v, &SessionCredentials, ns)

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

		log.Debugf("response = %v", response)
		log.Debugf("len of items = %d", len(response.BackupList.Items))

		for _, backup := range response.BackupList.Items {
			printBackupCRD(&backup)
		}

	}

}

// printBackupCRD ...
func printBackupCRD(result *crv1.Pgbackup) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgbackup : "+result.Spec.Name)

	fmt.Printf("%s%s\n", TreeBranch, "Namespace:\t"+result.Spec.Namespace)
	fmt.Printf("%s%s\n", TreeBranch, "PVC Name:\t"+result.Spec.StorageSpec.Name)
	fmt.Printf("%s%s\n", TreeBranch, "Access Mode:\t"+result.Spec.StorageSpec.AccessMode)
	fmt.Printf("%s%s\n", TreeBranch, "PVC Size:\t"+result.Spec.StorageSpec.Size)
	fmt.Printf("%s%s\n", TreeBranch, "Creation:\t"+result.ObjectMeta.CreationTimestamp.String())
	fmt.Printf("%s%s\n", TreeBranch, "CCPImageTag:\t"+result.Spec.CCPImageTag)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Status:\t"+result.Spec.BackupStatus)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Host:\t"+result.Spec.BackupHost)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Secret:\t"+result.Spec.BackupUserSecret)
	fmt.Printf("%s%s\n", TreeTrunk, "Backup Port:\t"+result.Spec.BackupPort)

	for _, v := range result.Spec.Toc {
		fmt.Printf("%s%s\n", TreeTrunk, "Backup Path:\t"+v)
	}

}

// deleteBackup ....
func deleteBackup(args []string, ns string) {
	log.Debugf("deleteBackup called %v", args)

	for _, v := range args {
		response, err := api.DeleteBackup(httpclient, v, &SessionCredentials, ns)

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
func createBackup(args []string, ns string) {
	log.Debugf("createBackup called %v BackupOpts %s", args, BackupOpts)

	request := new(msgs.CreateBackupRequest)
	request.Args = args
	request.Namespace = ns
	request.Selector = Selector
	request.PVCName = PVCName
	request.StorageConfig = StorageConfig
	request.BackupOpts = BackupOpts

	response, err := api.CreateBackup(httpclient, &SessionCredentials, request)

	if err != nil {
		fmt.Println("Error: " + err.Error())
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
