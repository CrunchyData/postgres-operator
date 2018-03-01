package cmd

/*
 Copyright 2018 Crunchy Data Solutions, Inc.
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

		url := APIServerURL + "/clusters/" + arg + "?selector=" + Selector + "&delete-data=" + strconv.FormatBool(DeleteData) + "&delete-backups=" + strconv.FormatBool(DeleteBackups)

		log.Debug("delete cluster called [" + url + "]")

		action := "DELETE"
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
			fmt.Println(RED(response.Status.Msg))
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

		url := APIServerURL + "/clusters/" + v + "?selector=" + Selector

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

		defer resp.Body.Close()

		var response msgs.ShowClusterResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
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

	for _, pod := range detail.Pods {
		fmt.Println(TreeBranch + "pod : " + pod.Name + " (" + string(pod.Phase) + " on " + pod.NodeName + ") (" + pod.ReadyStatus + ")")
		fmt.Println(TreeBranch + "pvc : " + pod.PVCName)
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
	r.NodeName = NodeName
	r.Password = Password
	r.SecretFrom = SecretFrom
	r.BackupPVC = BackupPVC
	r.UserLabels = UserLabels
	r.BackupPath = BackupPath
	r.Policies = PoliciesFlag
	r.CCPImageTag = CCPImageTag
	r.Series = Series
	r.MetricsFlag = MetricsFlag
	r.CustomConfig = CustomConfig
	r.StorageConfig = StorageConfig
	r.ReplicaStorageConfig = ReplicaStorageConfig

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
		fmt.Println(RED(response.Status.Msg))
		os.Exit(2)
	}

}
