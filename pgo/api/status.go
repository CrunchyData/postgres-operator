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
	"encoding/json"
	"fmt"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func ShowStatus(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, ns string) (msgs.StatusResponse, error) {

	var response msgs.StatusResponse
	url := SessionCredentials.APIServerURL + "/status?version=" + msgs.PGO_VERSION + "&namespace=" + ns
	log.Debug(url)

	action := "GET"
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		return response, err
	}

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
