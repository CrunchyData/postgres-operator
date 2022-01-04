// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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

// IgnoreValidation stores the flag input value that determines whether
// image tag version checking should be done before allowing an upgrade
// to continue
var IgnoreValidation bool

// UpgradeCCPImageTag stores the image tag for the cluster being upgraded.
// This is specifically required when upgrading PostGIS clusters because
// that tag will necessarily differ from the other images tags due to the
// inclusion of the PostGIS version.
var UpgradeCCPImageTag string

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
	UpgradeCmd.Flags().StringVarP(&UpgradeCCPImageTag, "ccp-image-tag", "", "", "The image tag to use for cluster creation. If specified, it overrides the default configuration setting and disables tag validation checking.")
	UpgradeCmd.Flags().BoolVarP(&IgnoreValidation, "ignore-validation", "", false, "Disables version checking against the image tags when performing an cluster upgrade.")
	UpgradeCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
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
	request.UpgradeCCPImageTag = UpgradeCCPImageTag

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
