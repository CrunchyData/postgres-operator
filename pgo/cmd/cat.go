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

var catCmd = &cobra.Command{
	Use:   "cat",
	Short: "Perform a cat command on a cluster",
	Long: `CAT performs a Linux cat command on a cluster file. For example:

	pgo cat mycluster /pgdata/mycluster/postgresql.conf`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("cat called")
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the cluster`)
		} else {
			cat(args, Namespace)
		}

	},
}

func init() {
	RootCmd.AddCommand(catCmd)
}

// pgo cat <clustername> <path> <path2>
func cat(args []string, ns string) {
	log.Debugf("cat called %v", args)

	request := new(msgs.CatRequest)
	request.Args = args
	request.Namespace = ns
	response, err := api.Cat(httpclient, &SessionCredentials, request)

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
