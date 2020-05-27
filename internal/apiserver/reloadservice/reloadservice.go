package reloadservice

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/internal/apiserver"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// ReloadHandler ...
// pgo reload all
// pgo reload --selector=name=mycluster
// pgo reload mycluster
func ReloadHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /reload reloadservice reload
	/*```
	RELOAD performs a PostgreSQL reload on a cluster or set of clusters.
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Reload Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/ReloadRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ReloadResponse"
	var err error
	var username, ns string

	log.Debug("reloadservice.ReloadHandler called")

	var request msgs.ReloadRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err = apiserver.Authn(apiserver.RELOAD_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp := msgs.ReloadResponse{}
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	reloadResponse := Reload(&request, ns, username)
	if err != nil {
		resp := msgs.ReloadResponse{}
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	json.NewEncoder(w).Encode(reloadResponse)
}
