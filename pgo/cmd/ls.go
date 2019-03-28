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
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "Perform a ls command on a cluster",
	Long: `LS performs a Linux ls command on a cluster directory. For example:

	pgo ls mycluster /pgdata/mycluster/pg_log`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("ls called")
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the cluster`)
		} else {
			ls(args, Namespace)
		}

	},
}

func init() {
	RootCmd.AddCommand(lsCmd)
}

// pgo ls <clustername> <path> <path2>
func ls(args []string, ns string) {
	log.Debugf("ls called %v", args)

	request := new(msgs.LsRequest)
	request.Args = args
	request.Namespace = ns
	response, err := api.Ls(httpclient, &SessionCredentials, request)

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
