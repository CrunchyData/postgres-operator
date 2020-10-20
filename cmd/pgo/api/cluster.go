package api

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"net/http"

	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

const (
	createClusterURL = "%s/clusters"
	deleteClusterURL = "%s/clustersdelete"
	updateClusterURL = "%s/clustersupdate"
	showClusterURL   = "%s/showclusters"
)

func ShowCluster(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.ShowClusterRequest) (msgs.ShowClusterResponse, error) {

	var response msgs.ShowClusterResponse

	jsonValue, _ := json.Marshal(request)
	url := fmt.Sprintf(showClusterURL, SessionCredentials.APIServerURL)
	log.Debugf("showCluster called...[%s]", url)

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return response, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	resp, err := httpclient.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	log.Debugf("%v", resp)
	err = StatusCheck(resp)
	if err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		log.Println(err)
		return response, err
	}

	return response, err

}

func DeleteCluster(httpclient *http.Client, request *msgs.DeleteClusterRequest, SessionCredentials *msgs.BasicAuthCredentials) (msgs.DeleteClusterResponse, error) {

	var response msgs.DeleteClusterResponse

	jsonValue, _ := json.Marshal(request)
	url := fmt.Sprintf(deleteClusterURL, SessionCredentials.APIServerURL)

	log.Debugf("delete cluster called %s", url)

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		response.Status.Code = msgs.Error
		return response, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	resp, err := httpclient.Do(req)
	if err != nil {
		fmt.Println("Error: Do: ", err)
		return response, err
	}
	defer resp.Body.Close()
	log.Debugf("%v", resp)
	err = StatusCheck(resp)
	if err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		log.Println(err)
		return response, err
	}

	return response, err

}

func CreateCluster(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.CreateClusterRequest) (msgs.CreateClusterResponse, error) {

	var response msgs.CreateClusterResponse

	jsonValue, _ := json.Marshal(request)
	url := fmt.Sprintf(createClusterURL, SessionCredentials.APIServerURL)
	log.Debugf("createCluster called...[%s]", url)

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return response, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	resp, err := httpclient.Do(req)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()

	log.Debugf("%v", resp)
	err = StatusCheck(resp)
	if err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Println(err)
		return response, err
	}

	return response, err
}

func UpdateCluster(httpclient *http.Client, request *msgs.UpdateClusterRequest, SessionCredentials *msgs.BasicAuthCredentials) (msgs.UpdateClusterResponse, error) {
	//func UpdateCluster(httpclient *http.Client, arg, selector string, SessionCredentials *msgs.BasicAuthCredentials, autofailFlag, ns string) (msgs.UpdateClusterResponse, error) {

	var response msgs.UpdateClusterResponse
	jsonValue, _ := json.Marshal(request)

	url := fmt.Sprintf(updateClusterURL, SessionCredentials.APIServerURL)
	log.Debugf("update cluster called %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		response.Status.Code = msgs.Error
		return response, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	resp, err := httpclient.Do(req)
	if err != nil {
		fmt.Println("Error: Do: ", err)
		return response, err
	}
	defer resp.Body.Close()
	log.Debugf("%v", resp)
	err = StatusCheck(resp)
	if err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		log.Println(err)
		return response, err
	}

	return response, err

}
