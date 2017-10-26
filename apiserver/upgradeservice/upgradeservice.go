package upgradeservice

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

// CreateUpgradeRequest ...
type CreateUpgradeRequest struct {
	Name string
}

// CreateUpgradeHandler ...
// pgo upgrade mycluster
// parameters --upgrade-type
// parameters --ccp-image-tag
func CreateUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("upgradeservice.CreateUpgradeHandler called")
	var request CreateUpgradeRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debug("upgradeservice.CreateUpgradeHandler got request " + request.Name)
}

// ShowUpgradeHandler ...
// pgo show upgrade
// pgo delete myupgrade
// parameters showsecrets
// parameters selector
// parameters namespace
// parameters postgresversion
// returns a ShowUpgradeResponse
func ShowUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("upgradeservice.ShowUpgradeHandler %v\n", vars)

	upgradename := vars["name"]

	namespace := r.URL.Query().Get("namespace")
	if namespace != "" {
		log.Debug("namespace param was [" + namespace + "]")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		log.Debug("upgradeservice.ShowUpgradeHandler GET called")
		resp := ShowUpgrade(namespace, upgradename)
		json.NewEncoder(w).Encode(resp)
	case "DELETE":
		log.Debug("upgradeservice.ShowUpgradeHandler DELETE called")
		resp := DeleteUpgrade(namespace, upgradename)
		json.NewEncoder(w).Encode(resp)
	}

}
