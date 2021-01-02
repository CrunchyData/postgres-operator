package dfservice

/*
Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"net/http"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
)

// CreateErrorResponse creates an error response message
func CreateErrorResponse(errorMessage string) msgs.DfResponse {
	return msgs.DfResponse{
		Status: msgs.Status{
			Code: msgs.Error,
			Msg:  errorMessage,
		},
	}
}

// StatusHandler ...
// pgo df mycluster
// pgo df --selector=env=research
func DfHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /df/{name} dfservice df-name
	/*```
	  Displays the disk status for PostgreSQL clusters.
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "PostgreSQL Cluster Disk Utilization"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DfRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DfResponse"
	log.Debug("dfservice.DFHandler called")

	// first, check that the requesting user is authorized to make this request
	username, err := apiserver.Authn(apiserver.DF_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	// decode the request paramaeters
	var request msgs.DfRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := CreateErrorResponse(err.Error())
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Debugf("DfHandler parameters [%+v]", request)

	// set some of the header...though we really should not be setting the HTTP
	// Status upfront, but whatever
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// check that the client versions match. If they don't, error out
	if request.ClientVersion != msgs.PGO_VERSION {
		response := CreateErrorResponse(apiserver.VERSION_MISMATCH_ERROR)
		json.NewEncoder(w).Encode(response)
		return
	}

	// ensure that the user has access to this namespace. if not, error out
	if _, err := apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace); err != nil {
		response := CreateErrorResponse(err.Error())
		json.NewEncoder(w).Encode(response)
		return
	}

	// process the request
	response := DfCluster(request)

	// turn the response into JSON
	json.NewEncoder(w).Encode(response)
}
