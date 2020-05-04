// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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

	msgs "github.com/crunchydata/postgres-operator/internal/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// IgnoreValidation stores the flag input value that determines whether
// image tag version checking should be done before allowing an upgrade
// to continue
var IgnoreValidation bool

var UpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Perform a cluster upgrade.",
	Long: `UPGRADE allows you to perform a comprehensive PGCluster upgrade 
	(for use after performing a Postgres Operator upgrade). 
	For example:
	
	pgo upgrade mycluster
	Upgrades the cluster for use with the upgraded Postgres Operator version.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("cluster upgrade called")
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: You must specify the cluster to upgrade.`)
		} else {
			fmt.Println("All active replicas will be scaled down and the primary database in this cluster will be stopped and recreated as part of this workflow!")
			if util.AskForConfirmation(NoPrompt, "") {
				createUpgrade(args, Namespace)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(UpgradeCmd)

	// flags for "pgo upgrade"
	UpgradeCmd.Flags().BoolVarP(&IgnoreValidation, "ignore-validation", "", false, "Disables version checking against the image tags when performing an cluster upgrade.")
}

func createUpgrade(args []string, ns string) {
	log.Debugf("createUpgrade called %v", args)

	if len(args) == 0 && Selector == "" {
		fmt.Println("Error: Cluster name(s) or a selector flag is required.")
		os.Exit(2)
	}

	request := msgs.CreateUpgradeRequest{}
	request.Args = args
	request.Namespace = ns
	request.Selector = Selector
	request.ClientVersion = msgs.PGO_VERSION
	request.IgnoreValidation = IgnoreValidation

	response, err := api.CreateUpgrade(httpclient, &SessionCredentials, &request)

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
