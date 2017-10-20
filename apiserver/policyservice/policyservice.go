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
	log.Infoln("policyservice.CreatePolicyHandler called")
	var request msgs.CreatePolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("policyservice.CreatePolicyHandler got request " + request.Name)

	err := CreatePolicy(apiserver.RESTClient, request.Namespace, request.Name, request.URL, request.SQL)
	if err != nil {
		log.Error(err.Error())
		log.Infoln("error would be reported back to caller!!!!")
	}
}

// ShowPolicyHandler ...
// returns a ShowPolicyResponse
func ShowPolicyHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("policyservice.ShowPolicyHandler called")
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	policyname := vars["name"]
	log.Infof(" name arg is %v\n", policyname)

	namespace := r.URL.Query().Get("namespace")
	if namespace != "" {
		log.Infoln("namespace param was [" + namespace + "]")
	} else {
		log.Infoln("namespace param was null")
	}

	switch r.Method {
	case "GET":
		log.Infoln("policyservice.ShowPolicyHandler GET called")
	case "DELETE":
		log.Infoln("policyservice.ShowPolicyHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ShowPolicyResponse{}
	resp.PolicyList = ShowPolicy(apiserver.RESTClient, namespace, policyname)

	json.NewEncoder(w).Encode(resp)
}

// ApplyPolicyHandler ...
// pgo apply mypolicy --selector=name=mycluster
func ApplyPolicyHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Infoln("policyservice.ApplyPolicyHandler called")

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
