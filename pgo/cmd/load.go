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
	"io/ioutil"
	"k8s.io/apimachinery/pkg/labels"
	"os"
)

var LoadConfig string

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Perform a data load",
	Long: `LOAD performs a load. For example:

	pgo load --load-config=./load.json --selector=project=xray`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("load called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: You must specify the cluster to load or a selector flag.`)
		} else {
			if LoadConfig == "" {
				fmt.Println("Error: You must specify the load-config.")
				return
			}

			createLoad(args, Namespace)
		}

	},
}

func init() {
	RootCmd.AddCommand(loadCmd)

	loadCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	loadCmd.Flags().StringVarP(&LoadConfig, "load-config", "", "", "The load configuration to use that defines the load job.")
	loadCmd.Flags().StringVarP(&PoliciesFlag, "policies", "", "", "The policies to apply before loading a file, comma separated.")

}

func createLoad(args []string, ns string) {
	if PoliciesFlag != "" {
		log.Debug("policies=" + PoliciesFlag)
	} else {
		log.Debug("The --policies flag requires a value.")
	}
	if Selector != "" {
		//use the selector instead of an argument list to filter on

		_, err := labels.Parse(Selector)
		if err != nil {
			fmt.Println("Error: Could not parse selector flag.")
			return
		}
	}

	buf, err := ioutil.ReadFile(LoadConfig)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	request := msgs.LoadRequest{}
	request.LoadConfig = string(buf)
	request.Selector = Selector
	request.Namespace = ns
	request.Policies = PoliciesFlag
	request.Args = args
	request.ClientVersion = msgs.PGO_VERSION

	//make the request

	response, err := api.CreateLoad(httpclient, &SessionCredentials, &request)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	//get the response
	if response.Status.Code == msgs.Error {
		fmt.Println("Error: Error in loading...")
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	for i := 0; i < len(response.Results); i++ {
		fmt.Println(response.Results[0])
	}

}
