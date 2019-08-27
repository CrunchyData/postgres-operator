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

func ShowPgouser(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.ShowPgouserRequest) (msgs.ShowPgouserResponse, error) {

	var response msgs.ShowPgouserResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgousershow"
	log.Debugf("ShowPgouser called...[%s]", url)

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
func CreatePgouser(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.CreatePgouserRequest) (msgs.CreatePgouserResponse, error) {

	var resp msgs.CreatePgouserResponse
	resp.Status.Code = msgs.Ok

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgousercreate"
	log.Debugf("CreatePgouser called...[%s]", url)

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

func DeletePgouser(httpclient *http.Client, request *msgs.DeletePgouserRequest, SessionCredentials *msgs.BasicAuthCredentials) (msgs.DeletePgouserResponse, error) {

	var response msgs.DeletePgouserResponse

	url := SessionCredentials.APIServerURL + "/pgouserdelete"

	log.Debugf("DeletePgouser called [%s]", url)

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

func UpdatePgouser(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.UpdatePgouserRequest) (msgs.UpdatePgouserResponse, error) {

	var response msgs.UpdatePgouserResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgouserupdate"
	log.Debugf("UpdatePgouser called...[%s]", url)

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
