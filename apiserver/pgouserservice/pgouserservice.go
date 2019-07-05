package pgouserservice

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

func CreatePgouserHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	resp := msgs.CreatePgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	log.Debug("pgouserservice.CreatePgouserHandler called")

	var request msgs.CreatePgouserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.CREATE_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	log.Debugf("pgouserservice.CreatePgouserHandler got request %s", request.Name)
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
		resp.Status.Msg = "invalid pgouser name format " + errs[0]
	} else {

		err := CreatePgouser(apiserver.RESTClient, &request)
		if err != nil {
			log.Error(err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
		}
		if found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "pgouser already exists with that name"
		}
	}

	json.NewEncoder(w).Encode(resp)
}

func DeletePgouserHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	var request msgs.DeletePgouserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("DeletePgouserHandler parameters [%v]", request)

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	username, err := apiserver.Authn(apiserver.DELETE_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}
	resp := msgs.DeletePgouserResponse{}
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

	resp = DeletePgouser(apiserver.RESTClient, &request)

	json.NewEncoder(w).Encode(resp)

}

func ShowPgouserHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	var request msgs.ShowPgouserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("ShowPgouserHandler parameters [%v]", request)

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	username, err := apiserver.Authn(apiserver.SHOW_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	log.Debug("pgouserservice.ShowPgouserHandler POST called")
	resp := msgs.ShowPgouserResponse{}
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

	resp = ShowPgouser(apiserver.RESTClient, &request)

	json.NewEncoder(w).Encode(resp)

}

func UpdatePgouserHandler(w http.ResponseWriter, r *http.Request) {

	var ns string
	log.Debug("pgouserservice.UpdatePgouserHandler called")

	var request msgs.UpdatePgouserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.UPDATE_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.ApplyPgouserResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ApplyPgouser(&request)
	json.NewEncoder(w).Encode(resp)
}
