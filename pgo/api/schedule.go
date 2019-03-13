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
	"net/http"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

const (
	createScheduleURL = "%s/schedule"
	deleteScheduleURL = "%s/scheduledelete"
	showScheduleURL   = "%s/scheduleshow"
)

func CreateSchedule(client *http.Client, SessionCredentials *msgs.BasicAuthCredentials, r *msgs.CreateScheduleRequest) (msgs.CreateScheduleResponse, error) {
	var response msgs.CreateScheduleResponse

	jsonValue, _ := json.Marshal(r)
	url := fmt.Sprintf(createScheduleURL, SessionCredentials.APIServerURL)

	log.Debugf("create schedule called [%s]", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return response, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	log.Debugf("%v", resp)
	if err := StatusCheck(resp); err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Println(err)
		return response, err
	}

	return response, err
}

func DeleteSchedule(client *http.Client, SessionCredentials *msgs.BasicAuthCredentials, r *msgs.DeleteScheduleRequest) (msgs.DeleteScheduleResponse, error) {
	var response msgs.DeleteScheduleResponse

	jsonValue, _ := json.Marshal(r)
	url := fmt.Sprintf(deleteScheduleURL, SessionCredentials.APIServerURL)

	log.Debugf("delete schedule called [%s]", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return response, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()

	log.Debugf("%v", resp)
	if err := StatusCheck(resp); err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Println(err)
		return response, err
	}

	return response, err
}

func ShowSchedule(client *http.Client, SessionCredentials *msgs.BasicAuthCredentials, r *msgs.ShowScheduleRequest) (msgs.ShowScheduleResponse, error) {
	var response msgs.ShowScheduleResponse

	jsonValue, _ := json.Marshal(r)
	url := fmt.Sprintf(showScheduleURL, SessionCredentials.APIServerURL)
	log.Debugf("show schedule called [%s]", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return response, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}

	defer resp.Body.Close()

	log.Debugf("%v", resp)
	if err := StatusCheck(resp); err != nil {
		return response, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Println(err)
		return response, err
	}

	return response, err
}
