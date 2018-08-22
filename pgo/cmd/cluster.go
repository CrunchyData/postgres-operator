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
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"net/http"
	"os"
	"strconv"
)

// deleteCluster ...
func deleteCluster(args []string) {
	log.Debugf("deleteCluster called %v\n", args)

	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	for _, arg := range args {
		log.Debug("deleting cluster " + arg + " with delete-data " + strconv.FormatBool(DeleteData))

		url := APIServerURL + "/clustersdelete/" + arg + "?selector=" + Selector + "&delete-data=" + strconv.FormatBool(DeleteData) + "&delete-backups=" + strconv.FormatBool(DeleteBackups) + "&version=" + msgs.PGO_VERSION

		log.Debug("delete cluster called [" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			fmt.Println("Error: NewRequest: ", err)
			return
		}

		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			fmt.Println("Error: Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)
		StatusCheck(resp)

		defer resp.Body.Close()
		var response msgs.DeleteClusterResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			fmt.Println("Error: ", err)
			log.Println(err)
			return
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

	log.Debugf("showCluster called %v\n", args)

	log.Debug("selector is " + Selector)
	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	for _, v := range args {

		url := APIServerURL + "/clusters/" + v + "?selector=" + Selector + "&version=" + msgs.PGO_VERSION

		log.Debug("show cluster called [" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)

		if err != nil {
			fmt.Println("Error: NewRequest: ", err)
			return
		}

		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			fmt.Println("Error: Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)
		StatusCheck(resp)

		defer resp.Body.Close()

		var response msgs.ShowClusterResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			fmt.Println("Error: ", err)
			log.Println(err)
			return
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
	fmt.Println("cluster : " + detail.Cluster.Spec.Name + " (" + detail.Cluster.Spec.CCPImageTag + ")")

	for _, pod := range detail.Pods {
		podType := "(" + pod.Type + ")"

		podStr := fmt.Sprintf("%spod : %s (%s) on %s (%s) %s", TreeBranch, pod.Name, string(pod.Phase), pod.NodeName, pod.ReadyStatus, podType)
		fmt.Println(podStr)
		for _, pvc := range pod.PVCName {
			fmt.Println(TreeBranch + "pvc : " + pvc)
		}
	}

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
	r.ArchiveFlag = ArchiveFlag
	r.BackrestFlag = BackrestFlag
	r.PgpoolSecret = PgpoolSecret
	r.CustomConfig = CustomConfig
	r.StorageConfig = StorageConfig
	r.ReplicaStorageConfig = ReplicaStorageConfig
	r.ContainerResources = ContainerResources
	r.ClientVersion = msgs.PGO_VERSION

	jsonValue, _ := json.Marshal(r)
	url := APIServerURL + "/clusters"
	log.Debug("createCluster called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Println("Error: NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		fmt.Println("Error: Do: ", err)
		return
	}

	log.Debugf("%v\n", resp)
	StatusCheck(resp)

	defer resp.Body.Close()

	var response msgs.CreateClusterResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		log.Println(err)
		return
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
