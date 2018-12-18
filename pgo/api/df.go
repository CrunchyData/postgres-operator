package api

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
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"net/http"
)

func ShowDf(httpclient *http.Client, arg, selector string, SessionCredentials *msgs.BasicAuthCredentials) (msgs.DfResponse, error) {

	var response msgs.DfResponse

	log.Debugf("ShowDf called %s with selector=%s", arg, selector)

	url := SessionCredentials.APIServerURL + "/df/" + arg + "?selector=" + selector + "&version=" + msgs.PGO_VERSION
	log.Debug(url)

	req, err := http.NewRequest("GET", url, nil)
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
		fmt.Print("Error: ")
		fmt.Println(err)
		log.Println(err)
		return response, err
	}

	return response, err

}
