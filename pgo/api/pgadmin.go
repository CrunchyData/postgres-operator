package api

/*
 Copyright 2018 - 2022 Crunchy Data Solutions, Inc.
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
	"io/ioutil"
	"net/http"

	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
)

func CreatePgAdmin(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.CreatePgAdminRequest) (msgs.CreatePgAdminResponse, error) {
	var response msgs.CreatePgAdminResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgadmin"
	log.Debugf("createPgAdmin called...[%s]", url)

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

	// Read resp.Body so we can log it in the event of error
	body, _ := ioutil.ReadAll(resp.Body)

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("Response body:\n%s\n", string(body))
		log.Println(err)
		return response, err
	}

	return response, err
}

func DeletePgAdmin(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.DeletePgAdminRequest) (msgs.DeletePgAdminResponse, error) {
	var response msgs.DeletePgAdminResponse

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgadmin"
	log.Debugf("deletePgAdmin called...[%s]", url)

	action := "DELETE"
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

	// Read resp.Body so we can log it in the event of error
	body, _ := ioutil.ReadAll(resp.Body)

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("Response body:\n%s\n", string(body))
		log.Println(err)
		return response, err
	}

	return response, err
}

// ShowPgAdmin makes an API call to the "show pgadmin" apiserver endpoint
// and provides the results either using the ShowPgAdmin response format which
// may include an error
func ShowPgAdmin(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials,
	request msgs.ShowPgAdminRequest) (msgs.ShowPgAdminResponse, error) {
	var response msgs.ShowPgAdminResponse

	// explicitly set the client version here
	request.ClientVersion = msgs.PGO_VERSION

	log.Debugf("ShowPgAdmin called [%+v]", request)

	// put the request into JSON format and format the URL and HTTP verb
	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgadmin/show"
	action := "POST"

	// prepare the request!
	httpRequest, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))

	// if there is an error preparing the request, return here
	if err != nil {
		return msgs.ShowPgAdminResponse{}, err
	}

	// set the headers around the request, including authentication information
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	// make the request! if there is an error making the request, return
	resp, err := httpclient.Do(httpRequest)
	if err != nil {
		return msgs.ShowPgAdminResponse{}, err
	}
	defer resp.Body.Close()

	log.Debugf("%+v", resp)

	// check on the HTTP status. If it is not 200, return here
	if err := StatusCheck(resp); err != nil {
		return msgs.ShowPgAdminResponse{}, err
	}

	// Read resp.Body so we can log it in the event of error
	body, _ := ioutil.ReadAll(resp.Body)

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("Response body:\n%s\n", string(body))
		log.Println(err)
		return msgs.ShowPgAdminResponse{}, err
	}

	return response, nil
}
