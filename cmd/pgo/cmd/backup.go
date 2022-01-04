// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/cmd/pgo/api"
	"github.com/crunchydata/postgres-operator/internal/config"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var PVCName string

// PGDumpDB is used to store the name of the pgDump database when
// performing either a backup or restore
var PGDumpDB string

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

	backupCmd.Flags().StringVarP(&BackupOpts, "backup-opts", "", "", "The options to pass into pgbackrest.")
	backupCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	backupCmd.Flags().StringVarP(&PVCName, "pvc-name", "", "", "The PVC name to use for the backup instead of the default.")
	backupCmd.Flags().StringVarP(&PGDumpDB, "database", "d", "postgres", "The name of the database pgdump will backup.")
	backupCmd.Flags().StringVar(&backupType, "backup-type", "pgbackrest", "The backup type to perform. Default is pgbackrest. Valid backup types are pgbackrest and pgdump.")
	backupCmd.Flags().StringVarP(&BackrestStorageType, "pgbackrest-storage-type", "", "", "The type of storage to use when scheduling pgBackRest backups. Either \"posix\", \"s3\", \"gcs\", \"posix,s3\" or \"posix,gcs\". (default \"posix\")")
}

// deleteBackup ....
func deleteBackup(namespace, clusterName string) {
	request := msgs.DeleteBackrestBackupRequest{
		ClusterName: clusterName,
		Namespace:   namespace,
		Target:      Target,
	}

	// make the request
	response, err := api.DeleteBackup(httpclient, &SessionCredentials, request)

	// if everything is OK, exit early
	if err == nil && response.Status.Code == msgs.Ok {
		return
	}

	// if there is an error, or the response code is not ok, print the error and
	// exit
	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else if response.Status.Code == msgs.Error {
		fmt.Println("Error: " + response.Status.Msg)
	}

	// since we can only have errors at this point, exit with error
	os.Exit(1)
}
