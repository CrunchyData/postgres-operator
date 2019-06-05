package policyservice

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
	"encoding/json"
	apiserver "github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/validation"
	"net/http"
)

// CreatePolicyHandler ...
// pgo create policy
// parameters secretfrom
func CreatePolicyHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	resp := msgs.CreatePolicyResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	log.Debug("policyservice.CreatePolicyHandler called")

	var request msgs.CreatePolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.CREATE_POLICY_PERM, w, r)
	if err != nil {
		return
	}

	log.Debugf("policyservice.CreatePolicyHandler got request %s", request.Name)
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	errs := validation.IsDNS1035Label(request.Name)
	if len(errs) > 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid policy name format " + errs[0]
	} else {

		found, err := CreatePolicy(apiserver.RESTClient, request.Name, request.URL, request.SQL, ns)
		if err != nil {
			log.Error(err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
		}
		if found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "policy already exists with that name"
		}
	}

	json.NewEncoder(w).Encode(resp)
}

// DeletePolicyHandler ...
// returns a DeletePolicyResponse
func DeletePolicyHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	var request msgs.DeletePolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	policyname := request.PolicyName
	clientVersion := request.ClientVersion
	namespace := request.Namespace

	log.Debugf("DeletePolicyHandler parameters version [%s] name [%s] namespace [%s]", clientVersion, policyname, namespace)

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	username, err := apiserver.Authn(apiserver.DELETE_POLICY_PERM, w, r)
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
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeletePolicy(apiserver.RESTClient, policyname, ns)

	json.NewEncoder(w).Encode(resp)

}

// ShowPolicyHandler ...
// returns a ShowPolicyResponse
func ShowPolicyHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	var request msgs.ShowPolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	policyname := request.Policyname

	clientVersion := request.ClientVersion
	namespace := request.Namespace

	log.Debugf("ShowPolicyHandler parameters version [%s] namespace [%s] name [%s]", clientVersion, namespace, policyname)

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	username, err := apiserver.Authn(apiserver.SHOW_POLICY_PERM, w, r)
	if err != nil {
		return
	}

	log.Debug("policyservice.ShowPolicyHandler POST called")
	resp := msgs.ShowPolicyResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	if clientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp.PolicyList = ShowPolicy(apiserver.RESTClient, policyname, request.AllFlag, ns)

	json.NewEncoder(w).Encode(resp)

}

// ApplyPolicyHandler ...
// pgo apply mypolicy --selector=name=mycluster
func ApplyPolicyHandler(w http.ResponseWriter, r *http.Request) {

	var ns string
	log.Debug("policyservice.ApplyPolicyHandler called")

	var request msgs.ApplyPolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.APPLY_POLICY_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.ApplyPolicyResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ApplyPolicy(&request, ns)
	json.NewEncoder(w).Encode(resp)
}
