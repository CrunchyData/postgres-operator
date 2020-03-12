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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	log "github.com/sirupsen/logrus"
)

// below are the tablespace parameters and the expected values of each
const (
	// tablespaceParamName represents the name of the PostgreSQL tablespace
	tablespaceParamName = "name"
	// tablespaceParamPVCSize represents the size of the PVC
	tablespaceParamPVCSize = "pvcsize"
	// tablespaceParamStorageConfig represents the storage config to use for the
	// tablespace
	tablespaceParamStorageConfig = "storageconfig"
)

// availableTablespaceParams is the list of acceptable parameters in the
// --tablespace flag
var availableTablespaceParams = map[string]struct{}{
	tablespaceParamName:          struct{}{},
	tablespaceParamPVCSize:       struct{}{},
	tablespaceParamStorageConfig: struct{}{},
}

// requiredTablespaceParams are the tablespace parameters that are required
var requiredTablespaceParams = []string{
	tablespaceParamName,
	tablespaceParamStorageConfig,
}

// deleteCluster will delete a PostgreSQL cluster that is managed by the
// PostgreSQL Operator
func deleteCluster(args []string, ns string) {
	log.Debugf("deleteCluster called %v", args)

	if AllFlag {
		args = make([]string, 1)
		args[0] = "all"
	}

	r := msgs.DeleteClusterRequest{}
	r.Selector = Selector
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns
	r.DeleteBackups = !KeepBackups
	r.DeleteData = !KeepData

	for _, arg := range args {
		r.Clustername = arg
		response, err := api.DeleteCluster(httpclient, &r, &SessionCredentials)

		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code == msgs.Ok {
			for _, result := range response.Results {
				fmt.Println(result)
			}
		} else {
			fmt.Println("Error: " + response.Status.Msg)
		}

	}

}

// showCluster ...
func showCluster(args []string, ns string) {

	log.Debugf("showCluster called %v", args)

	if OutputFormat != "" {
		if OutputFormat != "json" {
			fmt.Println("Error: ", "json is the only supported --output format value")
			os.Exit(2)
		}
	}

	log.Debugf("selector is %s", Selector)
	if len(args) == 0 && !AllFlag && Selector == "" {
		fmt.Println("Error: ", "--all needs to be set or a cluster name be entered or a --selector be specified")
		os.Exit(2)
	}
	if Selector != "" || AllFlag {
		args = make([]string, 1)
		args[0] = ""
	}

	r := new(msgs.ShowClusterRequest)
	r.Selector = Selector
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION

	for _, v := range args {

		r.Clustername = v
		response, err := api.ShowCluster(httpclient, &SessionCredentials, r)
		if err != nil {
			fmt.Println("Error: ", err.Error())
			os.Exit(2)
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Println(string(b))
			return
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

		if len(response.Results) == 0 {
			fmt.Println("No clusters found.")
			return
		}

		for _, clusterDetail := range response.Results {
			printCluster(&clusterDetail)
		}

	}

}

// printCluster
func printCluster(detail *msgs.ShowClusterDetail) {
	fmt.Println("")
	fmt.Println("cluster : " + detail.Cluster.Spec.Name + " (" + detail.Cluster.Spec.CCPImage + ":" + detail.Cluster.Spec.CCPImageTag + ")")

	for _, pod := range detail.Pods {
		podType := "(" + pod.Type + ")"

		podStr := fmt.Sprintf("%spod : %s (%s) on %s (%s) %s", TreeBranch, pod.Name, string(pod.Phase), pod.NodeName, pod.ReadyStatus, podType)
		fmt.Println(podStr)
		for _, pvc := range pod.PVCName {
			fmt.Println(TreeBranch + "pvc : " + pvc)
		}
	}

	resources := detail.Cluster.Spec.ContainerResources
	resourceStr := fmt.Sprintf("%sresources : CPU Limit=%s Memory Limit=%s, CPU Request=%s Memory Request=%s", TreeBranch, resources.LimitsCPU, resources.LimitsMemory, resources.RequestsCPU, resources.RequestsMemory)
	fmt.Println(resourceStr)

	storageStr := fmt.Sprintf("%sstorage : Primary=%s Replica=%s", TreeBranch, detail.Cluster.Spec.PrimaryStorage.Size, detail.Cluster.Spec.ReplicaStorage.Size)
	fmt.Println(storageStr)

	for _, d := range detail.Deployments {
		fmt.Println(TreeBranch + "deployment : " + d.Name)
	}
	if len(detail.Deployments) > 0 {
		printPolicies(&detail.Deployments[0])
	}

	for _, service := range detail.Services {
		if service.ExternalIP == "" {
			fmt.Println(TreeBranch + "service : " + service.Name + " - ClusterIP (" + service.ClusterIP + ")")
		} else {
			fmt.Println(TreeBranch + "service : " + service.Name + " - ClusterIP (" + service.ClusterIP + ") ExternalIP (" + service.ExternalIP + ")")
		}
	}

	for _, replica := range detail.Replicas {
		fmt.Println(TreeBranch + "pgreplica : " + replica.Name)
	}

	fmt.Printf("%s%s", TreeBranch, "labels : ")
	for k, v := range detail.Cluster.ObjectMeta.Labels {
		fmt.Printf("%s=%s ", k, v)
	}
	fmt.Println("")

}

func printPolicies(d *msgs.ShowClusterDeployment) {
	for _, v := range d.PolicyLabels {
		fmt.Printf("%spolicy: %s\n", TreeBranch, v)
	}
}

// createCluster ....
func createCluster(args []string, ns string, createClusterCmd *cobra.Command) {
	var err error

	if len(args) != 1 {
		fmt.Println("Error: A single Cluster name argument is required.")
		return
	}

	if !util.IsValidForResourceName(args[0]) {
		fmt.Println("Error: Cluster name specified is not valid name - must be lowercase alphanumeric")
		return
	}

	r := new(msgs.CreateClusterRequest)
	r.Name = args[0]
	r.Namespace = ns
	r.ReplicaCount = ClusterReplicaCount
	r.NodeLabel = NodeLabel
	r.Password = Password
	r.PasswordLength = PasswordLength
	r.SecretFrom = SecretFrom
	r.UserLabels = UserLabels
	r.Policies = PoliciesFlag
	r.CCPImageTag = CCPImageTag
	r.CCPImage = CCPImage
	r.MetricsFlag = MetricsFlag
	r.BadgerFlag = BadgerFlag
	r.ServiceType = ServiceType
	r.AutofailFlag = !DisableAutofailFlag
	r.PgbouncerFlag = PgbouncerFlag
	//r.ArchiveFlag = ArchiveFlagg
	r.BackrestStorageType = BackrestStorageType
	r.CustomConfig = CustomConfig
	r.StorageConfig = StorageConfig
	r.ReplicaStorageConfig = ReplicaStorageConfig
	r.ContainerResources = ContainerResources
	r.ClientVersion = msgs.PGO_VERSION
	r.PodAntiAffinity = PodAntiAffinity
	r.PodAntiAffinityPgBackRest = PodAntiAffinityPgBackRest
	r.PodAntiAffinityPgBouncer = PodAntiAffinityPgBouncer
	r.BackrestS3Key = BackrestS3Key
	r.BackrestS3KeySecret = BackrestS3KeySecret
	r.BackrestS3Bucket = BackrestS3Bucket
	r.BackrestS3Region = BackrestS3Region
	r.BackrestS3Endpoint = BackrestS3Endpoint
	r.PVCSize = PVCSize
	r.BackrestPVCSize = BackrestPVCSize
	r.Username = Username
	r.ShowSystemAccounts = ShowSystemAccounts

	// only set SyncReplication in the request if actually provided via the CLI
	if createClusterCmd.Flag("sync-replication").Changed {
		r.SyncReplication = &SyncReplication
	}

	// determine if the user wants to create tablespaces as part of this request,
	// and if so, set the values
	setTablespaces(r)

	response, err := api.CreateCluster(httpclient, &SessionCredentials, r)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(2)
	}

	if response.Status.Code == msgs.Error {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	// print out the legacy cluster information
	fmt.Println("created cluster:", response.Result.Name)
	fmt.Println("workflow id:", response.Result.WorkflowID)
	fmt.Println("users:")

	for _, user := range response.Result.Users {
		fmt.Println("\tusername:", user.Username, "password:", user.Password)
	}
}

// isTablespaceParam returns true if the parameter in question is acceptable for
// using with a tablespace.
func isTablespaceParam(param string) bool {
	_, found := availableTablespaceParams[param]

	return found
}

// setTablespaces determines if there are any Tablespaces that were provided
// via the `--tablespace` CLI flag, and if so, process their values. If
// everything checks out, one or more tablespaces are added to the cluster
// request
func setTablespaces(request *msgs.CreateClusterRequest) {
	// if there are no tablespaces set in the Tablespaces slice, abort
	if len(Tablespaces) == 0 {
		return
	}

	for _, tablespace := range Tablespaces {
		tablespaceDetails := map[string]string{}

		// tablespaces are in the format "name=tsname:storageconfig=nfsstorage",
		// so we need to split this out in order to put that information into the
		// tablespace detail struct
		// we will do the initial split of the string, and then iterate to get the
		// key value map of the parameters, ignoring any ones that do not exist
		for _, tablespaceParamValue := range strings.Split(tablespace, ":") {
			tablespaceDetailParts := strings.Split(tablespaceParamValue, "=")

			// if the split is not 2 items, then abort, as that means this is not
			// a valid key/value pair
			if len(tablespaceDetailParts) != 2 {
				fmt.Println(`Error: Tablespace was not specified in proper format (e.g. "name=tablespacename"), aborting.`)
				os.Exit(1)
			}

			// store the param as lower case
			param := strings.ToLower(tablespaceDetailParts[0])

			// if this is not a tablespace parameter, ignore it
			if !isTablespaceParam(param) {
				continue
			}

			// alright, store this param/value in the map
			tablespaceDetails[param] = tablespaceDetailParts[1]
		}

		// determine if the required parameters are in the map. if they are not,
		// abort
		for _, requiredParam := range requiredTablespaceParams {
			_, found := tablespaceDetails[requiredParam]

			if !found {
				fmt.Printf("Error: Required tablespace parameter \"%s\" is not found, aborting\n", requiredParam)
				os.Exit(1)
			}
		}

		// create the cluster tablespace detail and append it to the slice
		clusterTablespaceDetail := msgs.ClusterTablespaceDetail{
			Name:          tablespaceDetails[tablespaceParamName],
			PVCSize:       tablespaceDetails[tablespaceParamPVCSize],
			StorageConfig: tablespaceDetails[tablespaceParamStorageConfig],
		}

		// append to the tablespaces slice, and continue
		request.Tablespaces = append(request.Tablespaces, clusterTablespaceDetail)
	}
}

// updateCluster ...
func updateCluster(args []string, ns string) {
	log.Debugf("updateCluster called %v", args)

	r := msgs.UpdateClusterRequest{}
	r.Selector = Selector
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.Clustername = args

	// check to see if EnableAutofailFlag or DisableAutofailFlag is set. If so,
	// set a value for Autofail
	if EnableAutofailFlag {
		r.Autofail = msgs.UpdateClusterAutofailEnable
	} else if DisableAutofailFlag {
		r.Autofail = msgs.UpdateClusterAutofailDisable
	}

	response, err := api.UpdateCluster(httpclient, &r, &SessionCredentials)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for _, result := range response.Results {
			fmt.Println(result)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
	}

}
