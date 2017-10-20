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
	log.Infoln("upgradeservice.CreateUpgradeHandler called")
	var request CreateUpgradeRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("upgradeservice.CreateUpgradeHandler got request " + request.Name)
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
	log.Infoln("upgradeservice.ShowUpgradeHandler called")
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	upgradename := vars["name"]
	log.Infof(" name arg is %v\n", upgradename)

	namespace := r.URL.Query().Get("namespace")
	if namespace != "" {
		log.Infoln("namespace param was [" + namespace + "]")
	} else {
		log.Infoln("namespace param was null")
	}

	switch r.Method {
	case "GET":
		log.Infoln("upgradeservice.ShowUpgradeHandler GET called")
	case "DELETE":
		log.Infoln("upgradeservice.ShowUpgradeHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := ShowUpgrade(namespace, upgradename)

	json.NewEncoder(w).Encode(resp)
}
