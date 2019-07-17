package api

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
	"bytes"
	"encoding/json"
	"fmt"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func ShowNamespace(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.ShowNamespaceRequest) (msgs.ShowNamespaceResponse, error) {

	var resp msgs.ShowNamespaceResponse
	resp.Status.Code = msgs.Ok

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/namespace"
	log.Debugf("ShowNamespace called...[%s]", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		resp.Status.Code = msgs.Error
		return resp, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)
	r, err2 := httpclient.Do(req)
	if err2 != nil {
		return resp, err2
	}
	defer r.Body.Close()

	log.Debugf("%v", r)
	err = StatusCheck(r)
	if err != nil {
		return resp, err
	}

	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		log.Printf("%v\n", r.Body)
		fmt.Print("Error: ")
		fmt.Println(err)
		log.Println(err)
		return resp, err
	}

	return resp, err

}

func CreateNamespace(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.CreateNamespaceRequest) (msgs.CreateNamespaceResponse, error) {

	var resp msgs.CreateNamespaceResponse
	resp.Status.Code = msgs.Ok

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/namespacecreate"
	log.Debugf("CreateNamespace called...[%s]", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		resp.Status.Code = msgs.Error
		return resp, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	r, err := httpclient.Do(req)
	if err != nil {
		resp.Status.Code = msgs.Error
		return resp, err
	}
	defer r.Body.Close()

	log.Debugf("%v", r)
	err = StatusCheck(r)
	if err != nil {
		resp.Status.Code = msgs.Error
		return resp, err
	}

	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		log.Printf("%v\n", r.Body)
		log.Println(err)
		resp.Status.Code = msgs.Error
		return resp, err
	}

	log.Debugf("response back from apiserver was %v", resp)
	return resp, err
}

func DeleteNamespace(httpclient *http.Client, request *msgs.DeleteNamespaceRequest, SessionCredentials *msgs.BasicAuthCredentials) (msgs.DeleteNamespaceResponse, error) {

	var response msgs.DeleteNamespaceResponse

	url := SessionCredentials.APIServerURL + "/namespacedelete"

	log.Debugf("DeleteNamespace called [%s]", url)

	jsonValue, _ := json.Marshal(request)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		response.Status.Code = msgs.Error
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
func UpdateNamespace(httpclient *http.Client, request *msgs.UpdateNamespaceRequest, SessionCredentials *msgs.BasicAuthCredentials) (msgs.UpdateNamespaceResponse, error) {

	var response msgs.UpdateNamespaceResponse

	url := SessionCredentials.APIServerURL + "/namespaceupdate"

	log.Debugf("UpdateNamespace called [%s]", url)

	jsonValue, _ := json.Marshal(request)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		response.Status.Code = msgs.Error
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
