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
	"fmt"
	"net/http"

	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

// Restart POSTs a Restart request to the PostgreSQL Operator "restart" endpoint in order to restart
// a PG cluster or one or more instances within it.
func Restart(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials,
	request *msgs.RestartRequest) (msgs.RestartResponse, error) {

	var response msgs.RestartResponse

	jsonValue, _ := json.Marshal(request)
	url := fmt.Sprintf("%s/%s", SessionCredentials.APIServerURL, "restart")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return response, err
	}

	log.Debugf("restart called [%s]", url)

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	resp, err := httpclient.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	log.Debugf("restart response: %v", resp)

	if err := StatusCheck(resp); err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
		return response, err
	}

	return response, err
}

// QueryRestart sends a GET request to the PostgreSQL Operator "/restart/{clusterName}" endpoint
// in order to obtain information about the various instances available to restart within the
// cluster specified.
func QueryRestart(httpclient *http.Client, clusterName string, SessionCredentials *msgs.BasicAuthCredentials,
	namespace string) (msgs.QueryRestartResponse, error) {

	var response msgs.QueryRestartResponse

	url := fmt.Sprintf("%s/%s/%s", SessionCredentials.APIServerURL, "restart", clusterName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return response, err
	}

	q := req.URL.Query()
	q.Add("version", msgs.PGO_VERSION)
	q.Add("namespace", namespace)
	req.URL.RawQuery = q.Encode()

	log.Debugf("query restart called [%s]", req.URL)

	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	resp, err := httpclient.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	log.Debugf("query restart response: %v", resp)

	if err := StatusCheck(resp); err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
		return response, err
	}

	return response, err
}
