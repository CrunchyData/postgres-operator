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
	"context"
	"encoding/json"
	"net/http"

	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

func CreatePgbouncer(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.CreatePgbouncerRequest) (msgs.CreatePgbouncerResponse, error) {
	var response msgs.CreatePgbouncerResponse

	ctx := context.TODO()
	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgbouncer"
	log.Debugf("createPgbouncer called...[%s]", url)

	action := "POST"
	req, err := http.NewRequestWithContext(ctx, action, url, bytes.NewBuffer(jsonValue))
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

func DeletePgbouncer(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials, request *msgs.DeletePgbouncerRequest) (msgs.DeletePgbouncerResponse, error) {
	var response msgs.DeletePgbouncerResponse

	ctx := context.TODO()
	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgbouncer"
	log.Debugf("deletePgbouncer called...[%s]", url)

	action := "DELETE"
	req, err := http.NewRequestWithContext(ctx, action, url, bytes.NewBuffer(jsonValue))
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

// ShowPgBouncer makes an API call to the "show pgbouncer" apiserver endpoint
// and provides the results either using the ShowPgBouncer response format which
// may include an error
func ShowPgBouncer(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials,
	request msgs.ShowPgBouncerRequest) (msgs.ShowPgBouncerResponse, error) {
	// explicitly set the client version here
	request.ClientVersion = msgs.PGO_VERSION

	log.Debugf("ShowPgBouncer called [%+v]", request)

	// put the request into JSON format and format the URL and HTTP verb
	ctx := context.TODO()
	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgbouncer/show"
	action := "POST"

	// prepare the request!
	httpRequest, err := http.NewRequestWithContext(ctx, action, url, bytes.NewBuffer(jsonValue))
	// if there is an error preparing the request, return here
	if err != nil {
		return msgs.ShowPgBouncerResponse{}, err
	}

	// set the headers around the request, including authentication information
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	// make the request! if there is an error making the request, return

	httpResponse, err := httpclient.Do(httpRequest)
	if err != nil {
		return msgs.ShowPgBouncerResponse{}, err
	}

	defer httpResponse.Body.Close()

	log.Debugf("%+v", httpResponse)

	// check on the HTTP status. If it is not 200, return here
	if err := StatusCheck(httpResponse); err != nil {
		return msgs.ShowPgBouncerResponse{}, err
	}

	// attempt to decode the response into the expected JSON format
	response := msgs.ShowPgBouncerResponse{}

	if err := json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return msgs.ShowPgBouncerResponse{}, err
	}

	// we did it! return the response
	return response, nil
}

// UpdatePgBouncer makes an API call to the "update pgbouncer" apiserver
// endpoint and provides the results
func UpdatePgBouncer(httpclient *http.Client, SessionCredentials *msgs.BasicAuthCredentials,
	request msgs.UpdatePgBouncerRequest) (msgs.UpdatePgBouncerResponse, error) {
	// explicitly set the client version here
	request.ClientVersion = msgs.PGO_VERSION

	log.Debugf("UpdatePgBouncer called [%+v]", request)

	// put the request into JSON format and format the URL and HTTP verb
	ctx := context.TODO()
	jsonValue, _ := json.Marshal(request)
	url := SessionCredentials.APIServerURL + "/pgbouncer"
	action := "PUT"

	// prepare the request!
	httpRequest, err := http.NewRequestWithContext(ctx, action, url, bytes.NewBuffer(jsonValue))
	// if there is an error preparing the request, return here
	if err != nil {
		return msgs.UpdatePgBouncerResponse{}, err
	}

	// set the headers around the request, including authentication information
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.SetBasicAuth(SessionCredentials.Username, SessionCredentials.Password)

	// make the request! if there is an error making the request, return

	httpResponse, err := httpclient.Do(httpRequest)
	if err != nil {
		return msgs.UpdatePgBouncerResponse{}, err
	}

	defer httpResponse.Body.Close()

	log.Debugf("%+v", httpResponse)

	// check on the HTTP status. If it is not 200, return here
	if err := StatusCheck(httpResponse); err != nil {
		return msgs.UpdatePgBouncerResponse{}, err
	}

	// attempt to decode the response into the expected JSON format
	response := msgs.UpdatePgBouncerResponse{}

	if err := json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return msgs.UpdatePgBouncerResponse{}, err
	}

	// we did it! return the response
	return response, nil
}
