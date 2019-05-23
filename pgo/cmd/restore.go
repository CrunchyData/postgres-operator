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

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/pgo/api"
	pgoutil "github.com/crunchydata/postgres-operator/pgo/util"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var PITRTarget, RestoreBackupPath string
var BackupPath, BackupPVC string
var RestoreToPVC string

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Perform a restore from previous backup",
	Long: `RESTORE performs a restore to a new PostgreSQL cluster. This includes stopping the database and recreating a new primary with the restored data.  Valid backup types to restore from are pgbackrest and pgdump. For example:

	pgo restore mycluster `,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("restore called")
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the cluster name to restore from.`)
		} else {
			if BackupType == "" || BackupType == config.LABEL_BACKUP_TYPE_BACKREST {
				fmt.Println("Warning:  If currently running, the primary database in this cluster will be stopped and recreated as part of this workflow!")
			}
			if pgoutil.AskForConfirmation(NoPrompt, "") {
				restore(args, Namespace)
			} else {
				fmt.Println("Aborting...")
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVarP(&BackupOpts, "backup-opts", "", "", "The restore options for pgbackrest or pgdump.")
	restoreCmd.Flags().StringVarP(&PITRTarget, "pitr-target", "", "", "The PITR target, being a PostgreSQL timestamp such as '2018-08-13 11:25:42.582117-04'.")
	restoreCmd.Flags().StringVarP(&NodeLabel, "node-label", "", "", "The node label (key=value) to use when scheduling "+
		"the restore job, and in the case of a pgBackRest restore, also the new (i.e. restored) primary deployment. If not set, any node is used.")
	restoreCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	restoreCmd.Flags().StringVarP(&BackupPVC, "backup-pvc", "", "", "The PVC containing the pgdump or pgbasebackup backup directory to restore from.")
	restoreCmd.Flags().StringVarP(&BackupType, "backup-type", "", "", "The type of backup to restore from, default is pgbackrest. Valid types are pgbackrest, pgdump or pgbasebackup.")
	restoreCmd.Flags().StringVarP(&RestoreBackupPath, "backup-path", "", "", "The path for the directory containing the pg_basebackup backup to be utilized for the restore. If omitted, defaults to the latest backup.")
	restoreCmd.Flags().StringVarP(&RestoreToPVC, "restore-to-pvc", "", "", "The name of the PVC to restore into when restoring from a pgbasebackup backup.")
	restoreCmd.Flags().StringVarP(&BackrestStorageType, "pgbackrest-storage-type", "", "", "The type of storage to use for a pgBackRest restore. Either \"local\", \"s3\". (default \"local\")")
}

// restore ....
func restore(args []string, ns string) {
	log.Debugf("restore called %v", args)

	var response msgs.RestoreResponse
	var err error

	// use different request message, depending on type.
	if BackupType == "pgdump" {

		request := new(msgs.PgRestoreRequest)
		request.Namespace = ns
		request.FromCluster = args[0]
		request.RestoreOpts = BackupOpts
		request.PITRTarget = PITRTarget
		request.FromPVC = BackupPVC // use PVC specified on command line for pgrestore
		request.NodeLabel = NodeLabel

		response, err = api.RestoreDump(httpclient, &SessionCredentials, request)
	} else if BackupType == config.LABEL_BACKUP_TYPE_BASEBACKUP {

		request := new(msgs.PgbasebackupRestoreRequest)
		request.Namespace = ns
		request.FromCluster = args[0]

		if RestoreToPVC != "" {
			request.ToPVC = RestoreToPVC
		} else {
			request.ToPVC = args[0] + "-" + util.RandStringBytesRmndr(4)
		}

		request.FromPVC = BackupPVC
		request.BackupPath = RestoreBackupPath
		request.NodeLabel = NodeLabel

		response, err = api.RestorePgbasebackup(httpclient, &SessionCredentials, request)
	} else {

		request := new(msgs.RestoreRequest)
		request.Namespace = ns
		request.FromCluster = args[0]
		request.ToPVC = request.FromCluster + "-" + util.RandStringBytesRmndr(4)
		request.RestoreOpts = BackupOpts
		request.PITRTarget = PITRTarget
		request.NodeLabel = NodeLabel
		request.BackrestStorageType = BackrestStorageType

		response, err = api.Restore(httpclient, &SessionCredentials, request)
	}

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
