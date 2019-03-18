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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

const (
	createBenchmarkURL = "%s/benchmark"
	deleteBenchmarkURL = "%s/benchmarkdelete"
	showBenchmarkURL   = "%s/benchmarkshow"
)

func ShowBenchmark(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.ShowBenchmarkRequest) (msgs.ShowBenchmarkResponse, error) {
	var response msgs.ShowBenchmarkResponse

	jsonValue, _ := json.Marshal(request)
	url := fmt.Sprintf(showBenchmarkURL, SessionCredentials.APIServerURL)

	log.Debugf("show benchmark called [%s]", url)

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

func DeleteBenchmark(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.DeleteBenchmarkRequest) (msgs.DeleteBenchmarkResponse, error) {
	var response msgs.DeleteBenchmarkResponse

	jsonValue, _ := json.Marshal(request)
	url := fmt.Sprintf(deleteBenchmarkURL, SessionCredentials.APIServerURL)

	log.Debugf("delete benchmark called [%s]", url)

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

func CreateBenchmark(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.CreateBenchmarkRequest) (msgs.CreateBenchmarkResponse, error) {
	var response msgs.CreateBenchmarkResponse

	jsonValue, _ := json.Marshal(request)
	url := fmt.Sprintf(createBenchmarkURL, SessionCredentials.APIServerURL)

	log.Debugf("create benchmark called [%s]", url)

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
