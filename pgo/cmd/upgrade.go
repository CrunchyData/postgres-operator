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
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Perform an upgrade",
	Long: `UPGRADE performs an upgrade on a PostgreSQL cluster. For example:

  pgo upgrade mycluster

 This upgrade will update the CCPImageTag of the deployment for the primary and all replicas.
 The running containers are upgraded one at a time, sequentially, in the following order: replicas, backrest-repo, then primary.

 Note: If the PostgreSQL Operator is deployed using OLM, the value of the CCPImageTag is overriden by what is in the RELATED_IMAGE_* environmental variables, e.g. for the PostgreSQL container, it would be the value of RELATED_IMAGE_CRUNCHY_POSTGRES_HA`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("upgrade called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: You must specify the cluster to upgrade.`)
		} else {
			createUpgrade(args, Namespace)
		}

	},
}

func init() {
	RootCmd.AddCommand(upgradeCmd)

	upgradeCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "", "", "The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.")

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
	request.CCPImageTag = CCPImageTag
	request.ClientVersion = msgs.PGO_VERSION

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
