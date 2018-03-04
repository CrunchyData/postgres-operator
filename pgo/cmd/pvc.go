package cmd

/*
 Copyright 2018 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/apiservermsgs"
	"net/http"
)

func showPVC(args []string) {
	log.Debugf("showPVC called %v\n", args)

	//args are a list of pvc names
	for _, arg := range args {
		log.Debug("show pvc called for " + arg)
		printPVC(arg, PVCRoot)

	}

}
func printPVC(pvcName, pvcRoot string) {

	url := APIServerURL + "/pvc/" + pvcName + "?pvcroot=" + pvcRoot
	log.Debug("showPolicy called...[" + url + "]")

	action := "GET"
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		//log.Info("here after new req")
		log.Fatal("NewRequest: ", err)
		return
	}
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)
	resp, err := httpclient.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}
	log.Debugf("%v\n", resp)
	StatusCheck(resp)

	defer resp.Body.Close()
	var response apiservermsgs.ShowPVCResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Error(err)
		log.Println(err)
		return
	}
	if len(response.Results) == 0 {
		fmt.Println("no PVC Results")
		return
	}
	log.Debugf("response = %v\n", response)

	if response.Status.Code == apiservermsgs.Error {
		log.Error(response.Status.Msg)
		return
	}

	for k, v := range response.Results {
		if k == len(response.Results)-1 {
			fmt.Printf("%s%s\n", TreeTrunk, "/"+v)
		} else {
			fmt.Printf("%s%s\n", TreeBranch, "/"+v)
		}
	}

}
