// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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

			case config.LABEL_BACKUP_TYPE_PGDUMP:

				createpgDumpBackup(args, Namespace)

			default:
				fmt.Println("Error: You must specify either pgbackrest or pgdump for the --backup-type.")

			}

		}

	},
}

var backupType string

func init() {
	RootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&BackupOpts, "backup-opts", "", "", "The pgbackup options to pass into pgbackrest.")
	backupCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	backupCmd.Flags().StringVarP(&PVCName, "pvc-name", "", "", "The PVC name to use for the backup instead of the default.")
	backupCmd.Flags().StringVar(&backupType, "backup-type", "pgbackrest", "The backup type to perform. Default is pgbackrest. Valid backup types are pgbackrest and pgdump.")
	backupCmd.Flags().StringVarP(&BackrestStorageType, "pgbackrest-storage-type", "", "", "The type of storage to use when scheduling pgBackRest backups. Either \"local\", \"s3\" or both, comma separated. (default \"local\")")

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
