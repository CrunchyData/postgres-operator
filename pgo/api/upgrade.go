package api

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"net/http"
)

func ShowUpgrade(httpclient *http.Client, APIServerURL, arg, BasicAuthUsername, BasicAuthPassword string) (msgs.ShowUpgradeResponse, error) {

	var response msgs.ShowUpgradeResponse

	url := APIServerURL + "/upgrades/" + arg + "?version=" + msgs.PGO_VERSION
	log.Debug("showUpgrade called...[" + url + "]")

	action := "GET"
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		return response, err
	}

	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)
	resp, err := httpclient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println("Error: Do: ", err)
		return response, err
	}
	log.Debugf("%v\n", resp)
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

func CreateUpgrade(httpclient *http.Client, APIServerURL, BasicAuthUsername, BasicAuthPassword string, request *msgs.CreateUpgradeRequest) (msgs.CreateUpgradeResponse, error) {

	var response msgs.CreateUpgradeResponse

	jsonValue, _ := json.Marshal(request)
	url := APIServerURL + "/upgrades"
	log.Debug("CreateUpgrade called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return response, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return response, err
	}

	log.Debugf("%v\n", resp)
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

func DeleteUpgrade(httpclient *http.Client, APIServerURL, v, BasicAuthUsername, BasicAuthPassword string) (msgs.DeleteUpgradeResponse, error) {

	var response msgs.DeleteUpgradeResponse

	url := APIServerURL + "/upgradesdelete/" + v + "?version=" + msgs.PGO_VERSION
	log.Debug("deleteUpgrade called...[" + url + "]")

	action := "GET"

	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		return response, err
	}

	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()
	log.Debugf("%v\n", resp)
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
