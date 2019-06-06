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

var ReplicaCount int

var scaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale a PostgreSQL cluster",
	Long: `The scale command allows you to adjust a Cluster's replica configuration. For example:

	pgo scale mycluster --replica-count=1`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("scale called")

		if len(args) == 0 {
			fmt.Println(`Error: You must specify the clusters to scale.`)
		} else {
			if ReplicaCount < 1 {
				fmt.Println("Error: --replica-count is required to be greater than 0, the default is 1 if not specified")
				return
			}
			if util.AskForConfirmation(NoPrompt, "") {
			} else {
				fmt.Println("Aborting...")
				os.Exit(2)
			}
			scaleCluster(args, Namespace)
		}
	},
}

func init() {
	RootCmd.AddCommand(scaleCmd)

	scaleCmd.Flags().StringVarP(&ServiceType, "service-type", "", "", "The service type to use in the replica Service. If not set, the default in pgo.yaml will be used.")
	scaleCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "", "", "The CCPImageTag to use for cluster creation. If specified, overrides the .pgo.yaml setting.")
	scaleCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	scaleCmd.Flags().IntVarP(&ReplicaCount, "replica-count", "", 1, "The replica count to apply to the clusters.")
	scaleCmd.Flags().StringVarP(&ContainerResources, "resources-config", "", "", "The name of a container resource configuration in pgo.yaml that holds CPU and memory requests and limits.")
	scaleCmd.Flags().StringVarP(&StorageConfig, "storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the replica storage.")
	scaleCmd.Flags().StringVarP(&NodeLabel, "node-label", "", "", "The node label (key) to use in placing the primary database. If not set, any node is used.")

}

func scaleCluster(args []string, ns string) {

	for _, arg := range args {
		log.Debugf(" %s ReplicaCount is %d", arg, ReplicaCount)
		response, err := api.ScaleCluster(httpclient, arg, ReplicaCount, ContainerResources, StorageConfig, NodeLabel, CCPImageTag, ServiceType, &SessionCredentials, ns)

		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code == msgs.Ok {
			for _, v := range response.Results {
				fmt.Println(v)
			}
		} else {
			fmt.Println("Error: " + response.Status.Msg)
		}

	}
}
