package configservice

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

// ShowConfigHandler ...
// pgo show config
func ShowConfigHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /config configservice config
	/*```
	  Show configuration information for the Operator.
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "version"
	//    description: "Client Version"
	//    in: "path"
	//    type: "string"
	//    required: true
	//  - name: "namespace"
	//    description: "Namespace"
	//    in: "path"
	//    type: "string"
	//    required: true
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowConfigResponse"
	clientVersion := r.URL.Query().Get("version")
	namespace := r.URL.Query().Get("namespace")

	log.Debugf("ShowConfigHandler parameters version [%s] namespace [%s]", clientVersion, namespace)

	username, err := apiserver.Authn(apiserver.SHOW_CONFIG_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.ShowConfigResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return
	}

	_, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowConfig()
	json.NewEncoder(w).Encode(resp)
}
