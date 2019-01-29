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
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	otherutil "github.com/crunchydata/postgres-operator/util"
	"github.com/spf13/cobra"
	"os"
)

var PITRTarget string

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Perform a pgBackRest restore",
	Long: `RESTORE performs a pgBackRest restore to a new PostgreSQL cluster. This includes stopping the database and recreating a new primary with the restored data.  For example:

	pgo restore mycluster `,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("restore called")
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the cluster name to restore from.`)
		} else {
			fmt.Println("Warning:  stopping this database and creating a new primary is part of the restore workflow!")
			if util.AskForConfirmation(NoPrompt, "") {
				restore(args, Namespace)
			} else {
				fmt.Println("Aborting...")
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVarP(&BackupOpts, "backup-opts", "", "", "The pgbackrest options for the restore.")
	restoreCmd.Flags().StringVarP(&PITRTarget, "pitr-target", "", "", "The PITR target, being a PostgreSQL timestamp such as '2018-08-13 11:25:42.582117-04'.")
	restoreCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "No command line confirmation.")

}

// restore ....
func restore(args []string, ns string) {
	log.Debugf("restore called %v", args)

	request := new(msgs.RestoreRequest)
	request.Namespace = ns
	request.FromCluster = args[0]
	request.ToPVC = request.FromCluster + "-" + otherutil.RandStringBytesRmndr(4)
	request.RestoreOpts = BackupOpts
	request.PITRTarget = PITRTarget

	response, err := api.Restore(httpclient, &SessionCredentials, request)
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
