package cmd

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"net/http"
	"os"
)

// deleteCluster ...
func deleteCluster(args []string) {
	log.Debugf("deleteCluster called %v\n", args)

	if Namespace == "" {
		log.Error("Namespace can not be empty")
		return
	}

	for _, arg := range args {
		log.Debug("deleting cluster " + arg)

		url := APIServerURL + "/clusters/" + arg + "?namespace=" + Namespace

		log.Debug("delete cluster called [" + url + "]")

		action := "DELETE"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}

		client := &http.Client{}

		resp, err := client.Do(req)
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
			fmt.Println(GREEN("ok"))
		} else {
			fmt.Println(RED(response.Status.Msg))
		}
		for _, result := range response.Results {
			fmt.Println(result)
		}

	}

}

// showCluster ...
func showCluster(args []string) {

	log.Debugf("showCluster called %v\n", args)
	if Namespace == "" {
		log.Error("Namespace can not be empty")
		return
	}

	log.Debug("selector is " + Labelselector)

	for _, v := range args {

		url := APIServerURL + "/clusters/" + v + "?namespace=" + Namespace + "&selector=" + Labelselector

		log.Debug("show cluster called [" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)

		if err != nil {
			//log.Info("here after new req")
			log.Fatal("NewRequest: ", err)
			return
		}

		client := &http.Client{}

		resp, err := client.Do(req)
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

		if len(response.ClusterList.Items) == 0 {
			fmt.Println("no clusters found")
			return
		}

		log.Debugf("response = %v\n", response)
		log.Debugf("len of items = %d\n", len(response.ClusterList.Items))

		for _, cluster := range response.ClusterList.Items {
			printCluster(&cluster)
		}

	}

}

// printCluster
func printCluster(result *crv1.Pgcluster) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgcluster : "+result.Spec.Name)

}

// createCluster ....
func createCluster(args []string) {
	var err error

	if len(args) == 0 {
		log.Error("cluster name argument is required")
		return
	}

	if Namespace == "" {
		log.Error("Namespace can not be empty")
		return
	}

	r := new(msgs.CreateClusterRequest)
	r.Name = args[0]
	r.Namespace = Namespace
	r.NodeName = NodeName
	r.Password = Password
	r.SecretFrom = SecretFrom
	r.BackupPVC = BackupPVC
	r.UserLabels = UserLabels
	r.BackupPath = BackupPath
	r.Policies = PoliciesFlag
	r.CCPImageTag = CCPImageTag
	r.Series = Series

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

	client := &http.Client{}

	resp, err := client.Do(req)
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
		fmt.Println(GREEN("ok"))

		for _, v := range response.Results {
			fmt.Println(v)
		}
	} else {
		fmt.Println(RED(response.Status.Msg))
		os.Exit(2)
	}

}
