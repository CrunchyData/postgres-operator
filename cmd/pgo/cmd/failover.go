// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/cmd/pgo/util"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var failoverCmd = &cobra.Command{
	Use:   "failover",
	Short: "Performs a manual failover",
	Long: `Performs a manual failover. For example:

	pgo failover mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("failover called")
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the cluster to failover.`)
		} else {
			if Query {
				queryFailover(args, Namespace)
			} else if util.AskForConfirmation(NoPrompt, "") {
				if Target == "" {
					fmt.Println(`Error: The --target flag is required for failover.`)
					return
				}
				createFailover(args, Namespace)
			} else {
				fmt.Println("Aborting...")
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(failoverCmd)

	failoverCmd.Flags().BoolVarP(&Query, "query", "", false, "Prints the list of failover candidates.")
	failoverCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	failoverCmd.Flags().StringVarP(&Target, "target", "", "", "The replica target which the failover will occur on.")

}

// createFailover ....
func createFailover(args []string, ns string) {
	log.Debugf("createFailover called %v", args)

	request := new(msgs.CreateFailoverRequest)
	request.Namespace = ns
	request.ClusterName = args[0]
	request.Target = Target
	request.ClientVersion = msgs.PGO_VERSION

	response, err := api.CreateFailover(httpclient, &SessionCredentials, request)

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

}

// queryFailover is a helper function to return the user information about the
// replicas that can be failed over to for this cluster. This is called when the
// "--query" flag is specified
func queryFailover(args []string, ns string) {
	log.Debugf("queryFailover called %v", args)

	// call the API
	response, err := api.QueryFailover(httpclient, args[0], &SessionCredentials, ns)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	// indicate which cluster this is. Put a newline before to put some
	// separation between each line
	if !response.Standby {
		fmt.Printf("\nCluster: %s\n", args[0])
	} else {
		fmt.Printf("\nCluster (standby): %s\n", args[0])
	}

	// If there is a controlled error, output the message here and continue
	// to iterate through the list
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// If there are no replicas found for this cluster, indicate so, and
	// continue to iterate through the list
	if len(response.Results) == 0 {
		fmt.Println("No replicas found.")
		return
	}

	// output the information about each instance
	fmt.Printf("%-20s\t%-10s\t%-10s\t%-20s\t%s\n", "REPLICA", "STATUS", "NODE", "REPLICATION LAG",
		"PENDING RESTART")

	for i := 0; i < len(response.Results); i++ {
		instance := response.Results[i]

		log.Debugf("postgresql instance: %v", instance)

		fmt.Printf("%-20s\t%-10s\t%-10s\t%12d %-7s\t%15t\n",
			instance.Name, instance.Status, instance.Node, instance.ReplicationLag, "MB",
			instance.PendingRestart)
	}
}
