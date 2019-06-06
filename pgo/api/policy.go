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

func ShowPolicy(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.ShowPolicyRequest) (msgs.ShowPolicyResponse, error) {

	var response msgs.ShowPolicyResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/showpolicies"
	log.Debugf("showPolicy called...[%s]", url)

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
		log.Println(err)
		return response, err
	}

	return response, err

}
func CreatePolicy(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.CreatePolicyRequest) (msgs.CreatePolicyResponse, error) {

	var resp msgs.CreatePolicyResponse
	resp.Status.Code = msgs.Ok

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/policies"
	log.Debugf("createPolicy called...[%s]", url)

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
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

func DeletePolicy(httpclient *http.Client, request *msgs.DeletePolicyRequest, SessionCredentials *msgs.BasicAuthCredentials) (msgs.DeletePolicyResponse, error) {

	var response msgs.DeletePolicyResponse

	url := SessionCredentials.APIServerURL + "/policiesdelete"

	log.Debugf("delete policy called [%s]", url)

	action := "POST"
	jsonValue, _ := json.Marshal(request)
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
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

func ApplyPolicy(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.ApplyPolicyRequest) (msgs.ApplyPolicyResponse, error) {

	var response msgs.ApplyPolicyResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/policies/apply"
	log.Debugf("applyPolicy called...[%s]", url)

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
