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
	"github.com/crunchydata/postgres-operator/pgo/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var AutofailReplaceReplica string

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
	failoverCmd.Flags().StringVarP(&AutofailReplaceReplica, "autofail-replace-replica", "", "", "If 'true', causes a replica to be created to replace the promoted replica.  If 'false', causes a replica to not be created, if not set, the pgo.yaml AutofailReplaceReplica setting is used.")

}

// createFailover ....
func createFailover(args []string, ns string) {
	log.Debugf("createFailover called %v", args)

	request := new(msgs.CreateFailoverRequest)
	request.Namespace = ns
	request.ClusterName = args[0]
	request.Target = Target
	request.AutofailReplaceReplica = AutofailReplaceReplica
	if AutofailReplaceReplica != "" {
		if AutofailReplaceReplica == "true" || AutofailReplaceReplica == "false" {
		} else {
			fmt.Println("Error: --autofail-replace-replica if specified is required to be either true or false")
			os.Exit(2)
		}
	}
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

// queryFailover ....
func queryFailover(args []string, ns string) {
	log.Debugf("queryFailover called %v", args)

	response, err := api.QueryFailover(httpclient, args[0], &SessionCredentials, ns)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		fmt.Println(response.Status.Msg)
		if len(response.Targets) > 0 {
			fmt.Println("Failover targets include:")
			for i := 0; i < len(response.Targets); i++ {
				printTarget(response.Targets[i])
			}
		}
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

func printTarget(target msgs.FailoverTargetSpec) {
	var preferredNode string

	if target.PreferredNode {
		preferredNode = "(preferred node)"
	}
	fmt.Printf("\t%s (%s) (%s) %s ReceiveLoc (%d) ReplayLoc (%d)\n", target.Name, target.ReadyStatus, target.Node, preferredNode, target.ReceiveLocation, target.ReplayLocation)
}
