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
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var scaledownCmd = &cobra.Command{
	Use:   "scaledown",
	Short: "Scale down a PostgreSQL cluster",
	Long: `The scale command allows you to scale down a Cluster's replica configuration. For example:

	To list targetable replicas:
	pgo scaledown mycluster --query

	To scale down a specific replica:
	pgo scaledown mycluster --target=mycluster-replica-xxxx`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("scaledown called")

		if len(args) == 0 {
			fmt.Println(`Error: You must specify the clusters to scale down.`)
		} else {
			if Query {
				queryCluster(args, Namespace)
			} else {
				if Target == "" {
					fmt.Println(`Error: You must specify --target`)
					os.Exit(2)
				}
				if util.AskForConfirmation(NoPrompt, "") {
				} else {
					fmt.Println("Aborting...")
					os.Exit(2)
				}
				scaleDownCluster(args[0], Namespace)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(scaledownCmd)

	scaledownCmd.Flags().BoolVarP(&Query, "query", "", false, "Prints the list of targetable replica candidates.")
	scaledownCmd.Flags().StringVarP(&Target, "target", "", "", "The replica to target for scaling down")
	scaledownCmd.Flags().BoolVarP(&DeleteData, "delete-data", "d", true,
		"Causes the data for the scaled down replica to be removed permanently.")
	scaledownCmd.Flags().MarkDeprecated("delete-data", "Data is deleted by default.")
	scaledownCmd.Flags().BoolVar(&KeepData, "keep-data", false,
		"Causes data for the scale down replica to *not* be deleted")
	scaledownCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")

}

// queryCluster is a helper function that returns information about the
// available replicas that can be scaled down. This is called when the "--query"
// flag is specified
func queryCluster(args []string, ns string) {

	// iterate through the clusters and output information about each one
	for _, arg := range args {
		// indicate which cluster this is. Put a newline before to put some
		// separation between each line
		fmt.Printf("\nCluster: %s\n", arg)

		// call the API
		response, err := api.ScaleQuery(httpclient, arg, &SessionCredentials, ns)

		// If the API returns in error, just bail out here
		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		// If there is a controlled error, output the message here and continue
		// to iterate through the list
		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			continue
		}

		// If there are no replicas found for this cluster, indicate so, and
		// continue to iterate through the list
		if len(response.Results) == 0 {
			fmt.Println("No replicas found.")
			continue
		}

		// output the information about each instance
		fmt.Printf("%-20s\t%-10s\t%-10s\t%s\n", "REPLICA", "STATUS", "NODE", "REPLICATION LAG")
		for i := 0; i < len(response.Results); i++ {
			instance := response.Results[i]

			log.Debugf("postgresql instance: %v", instance)

			fmt.Printf("%-20s\t%-10s\t%-10s\t%12d MB\n",
				instance.Name, instance.Status, instance.Node, instance.ReplicationLag)
		}
	}
}

func scaleDownCluster(clusterName, ns string) {

	// determine if the data should be deleted. The modern flag for handling this
	// is "KeepData" which defaults to "false". We will honor the "DeleteData"
	// flag (which defaults to "true"), but this will be removed in a future
	// release
	deleteData := !KeepData && DeleteData

	response, err := api.ScaleDownCluster(httpclient, clusterName, Target, deleteData, &SessionCredentials, ns)

	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	if response.Status.Code == msgs.Ok {
		for _, v := range response.Results {
			fmt.Println(v)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
	}

}
