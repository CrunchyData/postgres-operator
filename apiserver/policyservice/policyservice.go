package policyservice

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
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	apiserver "github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	"k8s.io/apimachinery/pkg/util/validation"
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

	err := apiserver.Authn(apiserver.CREATE_POLICY_PERM, w, r)
	if err != nil {
		return
	}

	log.Debug("policyservice.CreatePolicyHandler got request " + request.Name)
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	errs := validation.IsDNS1035Label(request.Name)
	if len(errs) > 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid policy name format " + errs[0]
	} else {

		err := CreatePolicy(apiserver.RESTClient, request.Name, request.URL, request.SQL)
		if err != nil {
			log.Error(err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
		}
	}

	json.NewEncoder(w).Encode(resp)
}

// DeletePolicyHandler ...
// returns a DeletePolicyResponse
func DeletePolicyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("policyservice.DeletePolicyHandler %v\n", vars)

	policyname := vars["name"]
	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	err := apiserver.Authn(apiserver.DELETE_POLICY_PERM, w, r)
	if err != nil {
		return
	}
	log.Debug("policyservice.DeletePolicyHandler GET called")
	resp := msgs.DeletePolicyResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	if clientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp = DeletePolicy(apiserver.RESTClient, policyname)
	}

	json.NewEncoder(w).Encode(resp)

}

// ShowPolicyHandler ...
// returns a ShowPolicyResponse
func ShowPolicyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("policyservice.ShowPolicyHandler %v\n", vars)

	policyname := vars["name"]

	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	err := apiserver.Authn(apiserver.SHOW_POLICY_PERM, w, r)
	if err != nil {
		return
	}
	log.Debug("policyservice.ShowPolicyHandler GET called")
	resp := msgs.ShowPolicyResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	if clientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp.PolicyList = ShowPolicy(apiserver.RESTClient, policyname)
	}

	json.NewEncoder(w).Encode(resp)

}

// ApplyPolicyHandler ...
// pgo apply mypolicy --selector=name=mycluster
func ApplyPolicyHandler(w http.ResponseWriter, r *http.Request) {

	log.Debug("policyservice.ApplyPolicyHandler called")

	var request msgs.ApplyPolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	err := apiserver.Authn(apiserver.APPLY_POLICY_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := ApplyPolicy(&request)

	json.NewEncoder(w).Encode(resp)
}
