// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"fmt"
	"os"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restarts the PostgrSQL database within a PostgreSQL cluster",
	Long: `Restarts one or more PostgreSQL databases within a PostgreSQL cluster.

	For example, to restart the primary and all replicas:
	pgo restart mycluster

	Or target a specific instance within the cluster:
	pgo restart mycluster --target=mycluster-abcd

	And use the 'query' flag obtain a list of all instances within the cluster:
	pgo restart mycluster --query`,
	Run: func(cmd *cobra.Command, args []string) {

		if OutputFormat != "" {
			if OutputFormat != "json" {
				fmt.Println("Error: ", "json is the only supported --output format value")
				os.Exit(2)
			}
		}

		if Namespace == "" {
			Namespace = PGONamespace
		}

		if len(args) == 0 {
			fmt.Println(`Error: You must specify the cluster to restart.`)
		} else {
			switch {
			case Query:
				queryRestart(args, Namespace)
			case len(args) > 1:
				fmt.Println("Error: a single cluster must be specified when performing a restart")
			case util.AskForConfirmation(NoPrompt, ""):
				restart(args[0], Namespace)
			default:
				fmt.Println("Aborting...")
			}
		}
	},
}

func init() {

	RootCmd.AddCommand(restartCmd)

	restartCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	restartCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", `The output format. Supported types are: "json"`)
	restartCmd.Flags().BoolVarP(&Query, "query", "", false, "Prints the list of instances that can be restarted.")
	restartCmd.Flags().StringArrayVarP(&Targets, "target", "", []string{}, "The instance that will be restarted.")
}

// restart sends a request to restart a PG cluster or one or more instances within it.
func restart(clusterName, namespace string) {

	log.Debugf("restart called %v", clusterName)

	request := new(msgs.RestartRequest)
	request.Namespace = namespace
	request.ClusterName = clusterName
	request.Targets = Targets
	request.ClientVersion = msgs.PGO_VERSION

	response, err := api.Restart(httpclient, &SessionCredentials, request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	if OutputFormat == "json" {
		b, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println(string(b))

		if response.Status.Code != msgs.Ok {
			os.Exit(1)
		}
		return
	}

	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	for _, instance := range response.Result.Instances {
		if instance.Error {
			fmt.Printf("Error restarting instance %s: %s\n", instance.InstanceName, instance.ErrorMessage)
			continue
		}
		fmt.Printf("Successfully restarted instance %s\n", instance.InstanceName)
	}
}

// queryRestart  is called when the "--query" flag is specified, and displays a list of all
// instances (the primary and all replicas) within a cluster.  This is useful when the user
// would like to specify one or more instances for a restart using the "--target" flag.
func queryRestart(args []string, namespace string) {

	log.Debugf("queryRestart called %v", args)

	for _, clusterName := range args {
		response, err := api.QueryRestart(httpclient, clusterName, &SessionCredentials, namespace)
		if err != nil {
			fmt.Println("\nError: " + err.Error())
			continue
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("\nError: " + response.Status.Msg)
			continue
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Println(string(b))
			return
		}

		// indicate whether or not a standby cluster
		if !response.Standby {
			fmt.Printf("\nCluster: %s\n", clusterName)
		} else {
			fmt.Printf("\nCluster (standby): %s\n", clusterName)
		}

		// output the information about each instance
		fmt.Printf("%-20s\t%-10s\t%-10s\t%-10s\t%-20s\t%s\n", "INSTANCE", "ROLE", "STATUS", "NODE",
			"REPLICATION LAG", "PENDING RESTART")

		for i := 0; i < len(response.Results); i++ {
			instance := response.Results[i]

			log.Debugf("postgresql instance: %v", instance)

			if instance.ReplicationLag != -1 {
				fmt.Printf("%-20s\t%-10s\t%-10s\t%-10s\t%12d %-7s\t%15t\n",
					instance.Name, instance.Role, instance.Status, instance.Node, instance.ReplicationLag, "MB",
					instance.PendingRestart)
			} else {
				fmt.Printf("%-20s\t%-10s\t%-10s\t%-10s\t%15s\t%23t\n",
					instance.Name, instance.Role, instance.Status, instance.Node, "unknown",
					instance.PendingRestart)
			}
		}
	}
}
