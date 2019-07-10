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

func ShowPgorole(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.ShowPgoroleRequest) (msgs.ShowPgoroleResponse, error) {

	var response msgs.ShowPgoroleResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgoroleshow"
	log.Debugf("ShowPgorole called...[%s]", url)

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
		log.Println(err)
		return response, err
	}

	return response, err

}
func CreatePgorole(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.CreatePgoroleRequest) (msgs.CreatePgoroleResponse, error) {

	var resp msgs.CreatePgoroleResponse
	resp.Status.Code = msgs.Ok

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgorolecreate"
	log.Debugf("CreatePgorole called...[%s]", url)

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

func DeletePgorole(httpclient *http.Client, request *msgs.DeletePgoroleRequest, SessionCredentials *msgs.BasicAuthCredentials) (msgs.DeletePgoroleResponse, error) {

	var response msgs.DeletePgoroleResponse

	url := SessionCredentials.APIServerURL + "/pgoroledelete"

	log.Debugf("DeletePgorole called [%s]", url)

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

func UpdatePgorole(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.UpdatePgoroleRequest) (msgs.UpdatePgoroleResponse, error) {

	var response msgs.UpdatePgoroleResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgoroleupdate"
	log.Debugf("UpdatePgorole called...[%s]", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
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
