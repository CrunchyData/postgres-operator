package cloneservice

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
	"net/http"

	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

func CloneHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /clone cloneservice clone
	/*```
	  Clone a PostgreSQL cluster into a new deployment
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Clone PostgreSQL Cluster"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CloneRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CloneResponse"
	var err error
	var username, ns string

	log.Debug("cloneservice.CloneHanlder called")

	var request msgs.CloneRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err = apiserver.Authn(apiserver.CLONE_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)

	if err != nil {
		resp := msgs.CloneResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := Clone(&request, ns, username)
	json.NewEncoder(w).Encode(resp)
}
