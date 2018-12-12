package cmd

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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

	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
)

// deleteCluster ...
func deleteCluster(args []string) {
	log.Debugf("deleteCluster called %v", args)

	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	for _, arg := range args {
		response, err := api.DeleteCluster(httpclient, arg, Selector, &SessionCredentials, DeleteData, DeleteBackups, DeleteConfigMaps)
		//var response msgs.DeleteClusterResponse

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
func showCluster(args []string) {

	log.Debugf("showCluster called %v", args)

	if OutputFormat != "" {
		if OutputFormat != "json" {
			fmt.Println("Error: ", "json is the only supported --output format value")
			os.Exit(2)
		}
	}

	log.Debugf("selector is %s", Selector)
	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	for _, v := range args {

		response, err := api.ShowCluster(httpclient, v, Selector, CCPImageTag, &SessionCredentials)
		if err != nil {
			fmt.Println("Error: ", err.Error())
			os.Exit(2)
		}

		//var response msgs.ShowClusterResponse

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
	fmt.Println("cluster : " + detail.Cluster.Spec.Name + " (" + detail.Cluster.Spec.CCPImageTag + ")")

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
		fmt.Println(TreeBranch + "replica : " + replica.Name)
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
func createCluster(args []string) {
	var err error

	if len(args) == 0 {
		fmt.Println("Error: Cluster name argument is required.")
		return
	}

	r := new(msgs.CreateClusterRequest)
	r.Name = args[0]
	r.ReplicaCount = ClusterReplicaCount
	r.NodeLabel = NodeLabel
	r.Password = Password
	r.SecretFrom = SecretFrom
	r.BackupPVC = BackupPVC
	r.UserLabels = UserLabels
	r.BackupPath = BackupPath
	r.Policies = PoliciesFlag
	r.CCPImageTag = CCPImageTag
	r.Series = Series
	r.MetricsFlag = MetricsFlag
	r.BadgerFlag = BadgerFlag
	r.ServiceType = ServiceType
	r.AutofailFlag = AutofailFlag
	r.PgpoolFlag = PgpoolFlag
	r.PgbouncerFlag = PgbouncerFlag
	r.ArchiveFlag = ArchiveFlag
	r.BackrestFlag = BackrestFlag
	r.BackrestRestoreFrom = BackrestRestoreFrom
	r.PgpoolSecret = PgpoolSecret
	r.CustomConfig = CustomConfig
	r.StorageConfig = StorageConfig
	r.ReplicaStorageConfig = ReplicaStorageConfig
	r.ContainerResources = ContainerResources
	r.ClientVersion = msgs.PGO_VERSION

	response, err := api.CreateCluster(httpclient, &SessionCredentials, r)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for _, v := range response.Results {
			fmt.Println(v)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

// updateCluster ...
func updateCluster(args []string) {
	log.Debugf("updateCluster called %v", args)

	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	for _, arg := range args {
		response, err := api.UpdateCluster(httpclient, arg, Selector, &SessionCredentials, AutofailStringFlag)

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
