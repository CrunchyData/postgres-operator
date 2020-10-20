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

func ShowDf(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request msgs.DfRequest) (msgs.DfResponse, error) {
	var response msgs.DfResponse

	// explicitly set the client version here
	request.ClientVersion = msgs.PGO_VERSION

	log.Debugf("ShowDf called [%+v]", request)

	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/df"

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

	log.Debugf("%+v", resp)

	if err := StatusCheck(resp); err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Print("Error: ")
		fmt.Println(err)
		return response, err
	}

	return response, nil
}
