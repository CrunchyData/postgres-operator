package upgradeservice

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
	log "github.com/Sirupsen/logrus"
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
	log.Debug("upgradeservice.CreateUpgradeHandler called")
	var request msgs.CreateUpgradeRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	err := apiserver.Authn(apiserver.CREATE_UPGRADE_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := CreateUpgrade(&request)

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
	vars := mux.Vars(r)
	log.Debugf("upgradeservice.ShowUpgradeHandler %v\n", vars)

	upgradename := vars["name"]

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		log.Debug("upgradeservice.ShowUpgradeHandler GET called")

		err := apiserver.Authn(apiserver.SHOW_UPGRADE_PERM, w, r)
		if err != nil {
			return
		}

		resp := ShowUpgrade(upgradename)
		json.NewEncoder(w).Encode(resp)
	case "DELETE":
		log.Debug("upgradeservice.ShowUpgradeHandler DELETE called")

		err := apiserver.Authn(apiserver.DELETE_UPGRADE_PERM, w, r)
		if err != nil {
			return
		}

		resp := DeleteUpgrade(upgradename)
		json.NewEncoder(w).Encode(resp)
	}

}
