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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

func ScaleCluster(httpclient *http.Client, arg string, ReplicaCount int,
	StorageConfig, NodeLabel, CCPImageTag, ServiceType string,
	SessionCredentials *msgs.BasicAuthCredentials, ns string) (msgs.ClusterScaleResponse, error) {
	var response msgs.ClusterScaleResponse

	url := fmt.Sprintf("%s/clusters/scale/%s", SessionCredentials.APIServerURL, arg)
	log.Debug(url)

	ctx := context.TODO()
	action := "GET"
	req, err := http.NewRequestWithContext(ctx, action, url, nil)
	if err != nil {
		return response, err
	}

	q := req.URL.Query()
	q.Add("replica-count", strconv.Itoa(ReplicaCount))
	q.Add("storage-config", StorageConfig)
	q.Add("node-label", NodeLabel)
	q.Add("version", msgs.PGO_VERSION)
	q.Add("ccp-image-tag", CCPImageTag)
	q.Add("service-type", ServiceType)
	q.Add("namespace", ns)
	req.URL.RawQuery = q.Encode()

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
		fmt.Println("Error: ", err)
		log.Println(err)
		return response, err
	}

	return response, err
}
