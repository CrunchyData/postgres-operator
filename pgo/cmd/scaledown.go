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
	scaledownCmd.Flags().BoolVarP(&DeleteData, "delete-data", "d", false, "Causes the data for the scaled down replica to be removed permanently.")
	scaledownCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")

}

func queryCluster(args []string, ns string) {

	for _, arg := range args {
		response, err := api.ScaleQuery(httpclient, arg, &SessionCredentials, ns)

		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code == msgs.Ok {
			if len(response.Targets) > 0 {
				fmt.Println("Replica targets include:")
				for i := 0; i < len(response.Targets); i++ {
					printScaleTarget(response.Targets[i])
				}
			}

			for _, v := range response.Results {
				fmt.Println(v)
			}
		} else {
			fmt.Println("Error: " + response.Status.Msg)
		}

	}
}

func scaleDownCluster(clusterName, ns string) {

	response, err := api.ScaleDownCluster(httpclient, clusterName, Target, DeleteData, &SessionCredentials, ns)

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

func printScaleTarget(target msgs.ScaleQueryTargetSpec) {
	fmt.Printf("\t%s (%s) (%s) (%s)\n", target.Name, target.ReadyStatus, target.Node, target.RepStatus)
}
