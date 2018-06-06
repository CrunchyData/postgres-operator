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

		url := APIServerURL + "/clustersdelete/" + arg + "?selector=" + Selector + "&delete-data=" + strconv.FormatBool(DeleteData) + "&delete-backups=" + strconv.FormatBool(DeleteBackups) + "&version=" + ClientVersion

		log.Debug("delete cluster called [" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}

		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)
		StatusCheck(resp)

		defer resp.Body.Close()
		var response msgs.DeleteClusterResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if response.Status.Code == msgs.Ok {
			for _, result := range response.Results {
				fmt.Println(result)
			}
		} else {
			log.Error(RED(response.Status.Msg))
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

		url := APIServerURL + "/clusters/" + v + "?selector=" + Selector + "&version=" + ClientVersion

		log.Debug("show cluster called [" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)

		if err != nil {
			//log.Info("here after new req")
			log.Fatal("NewRequest: ", err)
			return
		}

		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)
		StatusCheck(resp)

		defer resp.Body.Close()

		var response msgs.ShowClusterResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Println("error:", err)
			}
			fmt.Println(string(b))
			return
		}

		if response.Status.Code != msgs.Ok {
			log.Error(RED(response.Status.Msg))
			os.Exit(2)
		}

		if len(response.Results) == 0 {
			fmt.Println("no clusters found")
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

	var primaryStr string
	for _, pod := range detail.Pods {
		if pod.Primary {
			primaryStr = "(primary)"
		} else {
			primaryStr = ""
		}
		podStr := fmt.Sprintf("%spod : %s (%s) on %s (%s) %s", TreeBranch, pod.Name, string(pod.Phase), pod.NodeName, pod.ReadyStatus, primaryStr)
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

	for i, service := range detail.Services {
		if i == len(detail.Services)-1 {
			fmt.Println(TreeBranch + "service : " + service.Name + " (" + service.ClusterIP + ")")
		} else {
			fmt.Println(TreeBranch + "service : " + service.Name + " (" + service.ClusterIP + ")")
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

	if ShowSecrets {
		for _, s := range detail.Secrets {
			fmt.Println("")
			fmt.Println("secret : " + s.Name)
			fmt.Println(TreeBranch + "username: " + s.Username)
			fmt.Println(TreeTrunk + "password: " + s.Password)
		}
	}

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
		log.Error("cluster name argument is required")
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
	r.AutofailFlag = AutofailFlag
	r.PgpoolFlag = PgpoolFlag
	r.ArchiveFlag = ArchiveFlag
	r.PgpoolSecret = PgpoolSecret
	r.CustomConfig = CustomConfig
	r.StorageConfig = StorageConfig
	r.ReplicaStorageConfig = ReplicaStorageConfig
	r.ContainerResources = ContainerResources
	r.ClientVersion = ClientVersion

	jsonValue, _ := json.Marshal(r)
	url := APIServerURL + "/clusters"
	log.Debug("createCluster called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	log.Debugf("%v\n", resp)
	StatusCheck(resp)

	defer resp.Body.Close()

	var response msgs.CreateClusterResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Error(err)
		log.Println(err)
		return
	}

	if response.Status.Code == msgs.Ok {
		for _, v := range response.Results {
			fmt.Println(v)
		}
	} else {
		log.Error(RED(response.Status.Msg))
		os.Exit(2)
	}

}
