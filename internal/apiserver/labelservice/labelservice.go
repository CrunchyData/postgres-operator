package labelservice

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
	"github.com/crunchydata/postgres-operator/internal/apiserver"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// LabelHandler ...
func LabelHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /label labelservice label
	/*```
	LABEL allows you to add a label on a set of clusters.
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Label Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/LabelRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/LabelResponse"
	var ns string

	log.Debug("labelservice.LabelHandler called")

	var request msgs.LabelRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.LABEL_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.LabelResponse{}
	resp.Status = msgs.Status{Msg: "", Code: msgs.Ok}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		resp.Status.Code = msgs.Error
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Msg: err.Error(), Code: msgs.Error}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = Label(&request, ns, username)

	json.NewEncoder(w).Encode(resp)
}

// DeleteLabelHandler ...
func DeleteLabelHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /labeldelete labelservice labeldelete
	/*```
	LABEL allows you to remove a label on a set of clusters.
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Delete Label Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DeleteLabelRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/LabelResponse"
	var ns string

	log.Debug("labelservice.DeleteLabelHandler called")

	var request msgs.DeleteLabelRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.LABEL_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.LabelResponse{}
	resp.Status = msgs.Status{Msg: "", Code: msgs.Ok}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Msg: apiserver.VERSION_MISMATCH_ERROR, Code: msgs.Error}
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Msg: err.Error(), Code: msgs.Error}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeleteLabel(&request, ns)

	json.NewEncoder(w).Encode(resp)
}
