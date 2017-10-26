package policyservice

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
	log "github.com/Sirupsen/logrus"
	apiserver "github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	"net/http"
)

// CreatePolicyHandler ...
// pgo create policy
// parameters secretfrom
func CreatePolicyHandler(w http.ResponseWriter, r *http.Request) {
	resp := msgs.CreatePolicyResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	log.Debug("policyservice.CreatePolicyHandler called")

	var request msgs.CreatePolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debug("policyservice.CreatePolicyHandler got request " + request.Name)

	err := CreatePolicy(apiserver.RESTClient, request.Namespace, request.Name, request.URL, request.SQL)
	if err != nil {
		log.Error(err.Error())
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}

// ShowPolicyHandler ...
// returns a ShowPolicyResponse
func ShowPolicyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("policyservice.ShowPolicyHandler %v\n", vars)

	policyname := vars["name"]

	namespace := r.URL.Query().Get("namespace")
	if namespace != "" {
		log.Debug("namespace param was [" + namespace + "]")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		log.Debug("policyservice.ShowPolicyHandler GET called")
		resp := msgs.ShowPolicyResponse{}
		resp.PolicyList = ShowPolicy(apiserver.RESTClient, namespace, policyname)

		json.NewEncoder(w).Encode(resp)
	case "DELETE":
		log.Debug("policyservice.ShowPolicyHandler DELETE called")
		resp := DeletePolicy(namespace, apiserver.RESTClient, policyname)
		json.NewEncoder(w).Encode(resp)
	}

}

// ApplyPolicyHandler ...
// pgo apply mypolicy --selector=name=mycluster
func ApplyPolicyHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Debug("policyservice.ApplyPolicyHandler called")

	var request msgs.ApplyPolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ApplyPolicyResponse{}
	resp.Name, err = ApplyPolicy(&request)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}
