package upgradeservice

/*
Copyright 2017-2019 Crunchy Data Solutions, Inc.
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
	log "github.com/sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	"net/http"
)

// UpgradeDetail ...
type UpgradeDetail struct {
	Name string
	//deployments
	//replicasets
	//pods
	//services
	//secrets
}

// ShowUpgradeResponse ...
type ShowUpgradeResponse struct {
	Items []UpgradeDetail
}

// CreateUpgradeHandler ...
// pgo upgrade mycluster
// parameters --upgrade-type
// parameters --ccp-image-tag
func CreateUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	log.Debug("upgradeservice.CreateUpgradeHandler called")
	var request msgs.CreateUpgradeRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.CREATE_UPGRADE_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.CreateUpgradeResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreateUpgrade(&request, ns)
	json.NewEncoder(w).Encode(resp)
}

// ShowUpgradeHandler ...
// pgo show upgrade
// pgo delete myupgrade
// parameters showsecrets
// parameters selector
// parameters postgresversion
// returns a ShowUpgradeResponse
func ShowUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	var ns string
	vars := mux.Vars(r)

	upgradename := vars["name"]
	clientVersion := r.URL.Query().Get("version")
	namespace := r.URL.Query().Get("namespace")
	log.Debugf("ShowUpgradeHandler parameters version [%s] namespace [%s]", clientVersion, namespace)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	log.Debug("upgradeservice.ShowUpgradeHandler GET called")

	username, err := apiserver.Authn(apiserver.SHOW_UPGRADE_PERM, w, r)
	if err != nil {
		return
	}

	var resp msgs.ShowUpgradeResponse
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ShowUpgradeResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(username, namespace)
	if err != nil {
		resp = msgs.ShowUpgradeResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowUpgrade(upgradename, ns)
	json.NewEncoder(w).Encode(resp)

}

// DeleteUpgradeHandler ...
// pgo delete upgrade
// returns a ShowUpgradeResponse
func DeleteUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	vars := mux.Vars(r)

	upgradename := vars["name"]
	clientVersion := r.URL.Query().Get("version")
	namespace := r.URL.Query().Get("namespace")
	log.Debugf("DeleteUpgradeHandler parameters version [%s] namespace [%s] name [%s]", clientVersion, namespace, upgradename)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	username, err := apiserver.Authn(apiserver.DELETE_UPGRADE_PERM, w, r)
	if err != nil {
		return
	}

	resp := msgs.DeleteUpgradeResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return

	}

	ns, err = apiserver.GetNamespace(username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeleteUpgrade(upgradename, ns)
	json.NewEncoder(w).Encode(resp)

}
