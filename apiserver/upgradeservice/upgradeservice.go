package upgradeservice

/*
Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// CreateUpgradeHandler ...
// pgo upgrade mycluster
// parameters --upgrade-type
// parameters --ccp-image-tag
func CreateUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /upgrades upgradeservice upgrades
	/*```
	UPGRADE performs an upgrade on a PostgreSQL cluster. For example:



	This upgrade will update the CCPImageTag of the deployment for the following: primary, replicas, and backrest-repo.
	The running containers are upgraded one at a time, sequentially, in the following order: replicas, backrest-repo, then primary.
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create Upgrade Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreateUpgradeRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreateUpgradeResponse"
	var ns string

	log.Debug("upgradeservice.CreateUpgradeHandler called")
	var request msgs.CreateUpgradeRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.CREATE_UPGRADE_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.CreateUpgradeResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreateUpgrade(&request, ns)
	json.NewEncoder(w).Encode(resp)
}
