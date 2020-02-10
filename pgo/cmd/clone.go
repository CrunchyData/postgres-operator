// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// the source cluster used for the clone, e.g. "oldcluster"
	SourceClusterName string
	// the target/destination cluster used for the clone, e.g. "newcluster"
	TargetClusterName string
	// BackrestStorageSource represents the data source to use (e.g. s3 or local) when both s3
	// and local are enabled in the cluster being cloned
	BackrestStorageSource string
)

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Copies the primary database of an existing cluster to a new cluster",
	Long: `Clone makes a copy of an existing PostgreSQL cluster managed by the Operator and creates a new PostgreSQL cluster managed by the Operator, with the data from the old cluster.

	pgo clone oldcluster newcluster`,
	Run: func(cmd *cobra.Command, args []string) {
		// if the namespace is not specified, default to the PGONamespace specified
		// in the `PGO_NAMESPACE` environmental variable
		if Namespace == "" {
			Namespace = PGONamespace
		}

		log.Debug("clone called")
		// ensure all the required arguments are available
		if len(args) < 1 {
			fmt.Println("Error: You must specifiy a cluster to clone from and a name for a new cluster")
			os.Exit(1)
		}

		if len(args) < 2 {
			fmt.Println("Error: You must specifiy the name of the new cluster")
			os.Exit(1)
		}

		clone(Namespace, args[0], args[1])
	},
}

// init is part of the cobra API
func init() {
	RootCmd.AddCommand(cloneCmd)

	cloneCmd.Flags().StringVarP(&BackrestStorageSource, "pgbackrest-storage-source", "", "",
		"The data source for the clone when both \"local\" and \"s3\" are enabled in the "+
			"source cluster. Either \"local\", \"s3\" or both, comma separated. (default \"local\")")
}

// clone is a helper function to help set up the clone!
func clone(namespace, sourceClusterName, targetClusterName string) {
	log.Debugf("clone called namespace:%s sourceClusterName:%s targetClusterName:%s",
		namespace, sourceClusterName, targetClusterName)

	// set up a request to the clone API sendpoint
	request := new(msgs.CloneRequest)
	request.Namespace = Namespace
	request.SourceClusterName = sourceClusterName
	request.TargetClusterName = targetClusterName
	request.BackrestStorageSource = BackrestStorageSource

	// make a call to the clone API
	response, err := api.Clone(httpclient, &SessionCredentials, request)

	// if there was an error with the API call, print that out here
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// if the response was unsuccesful due to user error, print out the error
	// message here
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// otherwise, print out some feedback:
	fmt.Println("Created clone task for: ", response.TargetClusterName)
	fmt.Println("workflow id is ", response.WorkflowID)
}
