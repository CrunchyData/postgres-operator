// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2019 - 2022 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/pgo/api"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
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
	Use:        "clone",
	Deprecated: `Use "pgo create cluster newcluster --restore-from=oldcluster" instead. "pgo clone" will be removed in a future release.`,
	Short:      "Copies the primary database of an existing cluster to a new cluster",
	Long: `Clone makes a copy of an existing PostgreSQL cluster managed by the Operator and creates a new PostgreSQL cluster managed by the Operator, with the data from the old cluster.

	pgo create cluster newcluster --restore-from=oldcluster
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
			fmt.Println("Error: You must specify a cluster to clone from and a name for a new cluster")
			os.Exit(1)
		}

		if len(args) < 2 {
			fmt.Println("Error: You must specify the name of the new cluster")
			os.Exit(1)
		}

		clone(Namespace, args[0], args[1])
	},
}

// init is part of the cobra API
func init() {
	RootCmd.AddCommand(cloneCmd)

	cloneCmd.Flags().StringVarP(&BackrestPVCSize, "pgbackrest-pvc-size", "", "",
		`The size of the PVC capacity for the pgBackRest repository. Overrides the value set in the storage class. This is ignored if the storage type of "local" is not used. Must follow the standard Kubernetes format, e.g. "10.1Gi"`)
	cloneCmd.Flags().StringVarP(&BackrestStorageSource, "pgbackrest-storage-source", "", "",
		"The data source for the clone when both \"local\" and \"s3\" are enabled in the "+
			"source cluster. Either \"local\", \"s3\" or both, comma separated. (default \"local\")")
	cloneCmd.Flags().BoolVar(&MetricsFlag, "enable-metrics", false, `If sets, enables metrics collection on the newly cloned cluster`)
	cloneCmd.Flags().StringVarP(&PVCSize, "pvc-size", "", "",
		`The size of the PVC capacity for primary and replica PostgreSQL instances. Overrides the value set in the storage class. Must follow the standard Kubernetes format, e.g. "10.1Gi"`)
}

// clone is a helper function to help set up the clone!
func clone(namespace, sourceClusterName, targetClusterName string) {
	log.Debugf("clone called namespace:%s sourceClusterName:%s targetClusterName:%s",
		namespace, sourceClusterName, targetClusterName)

	// set up a request to the clone API sendpoint
	request := msgs.CloneRequest{
		BackrestStorageSource: BackrestStorageSource,
		BackrestPVCSize:       BackrestPVCSize,
		EnableMetrics:         MetricsFlag,
		Namespace:             Namespace,
		PVCSize:               PVCSize,
		SourceClusterName:     sourceClusterName,
		TargetClusterName:     targetClusterName,
	}

	// make a call to the clone API
	response, err := api.Clone(httpclient, &SessionCredentials, &request)

	// if there was an error with the API call, print that out here
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// if the response was unsuccessful due to user error, print out the error
	// message here
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// otherwise, print out some feedback:
	fmt.Println("Created clone task for: ", response.TargetClusterName)
	fmt.Println("workflow id is ", response.WorkflowID)
}
