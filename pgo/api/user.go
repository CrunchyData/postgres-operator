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

func ShowUser(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.ShowUserRequest) (msgs.ShowUserResponse, error) {

	var response msgs.ShowUserResponse
	response.Status.Code = msgs.Ok

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/usershow"
	log.Debugf("ShowUser called...[%s]", url)

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
func CreateUser(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.CreateUserRequest) (msgs.CreateUserResponse, error) {

	var response msgs.CreateUserResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/usercreate"
	log.Debugf("createUsers called...[%s]", url)

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

func DeleteUser(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.DeleteUserRequest) (msgs.DeleteUserResponse, error) {

	var response msgs.DeleteUserResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/userdelete"
	log.Debugf("deleteUser called...[%s]", url)

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

func UpdateUser(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.UpdateUserRequest) (msgs.UpdateUserResponse, error) {

	var response msgs.UpdateUserResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/userupdate"
	log.Debugf("UpdateUser called...[%s]", url)

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
