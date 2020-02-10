package pgouserservice

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	apiserver "github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/validation"
	"net/http"
)

func CreatePgouserHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgousercreate pgouserservice pgousercreate
	/*```
	Create a pgouser
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create Pgouser Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreatePgouserRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreatePgouserResponse"
	resp := msgs.CreatePgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	log.Debug("pgouserservice.CreatePgouserHandler called")

	var request msgs.CreatePgouserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.CREATE_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debugf("pgouserservice.CreatePgouserHandler got request %v", request)
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	errs := validation.IsDNS1035Label(request.PgouserName)
	if len(errs) > 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid pgouser name format " + errs[0]
	} else {
		resp = CreatePgouser(apiserver.Clientset, username, &request)
	}

	json.NewEncoder(w).Encode(resp)
}

func DeletePgouserHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgouserdelete pgouserservice pgouserdelete
	/*```
	Delete a pgouser
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Delete Pgouser Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DeletePgouserRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DeletePgouserResponse"
	var request msgs.DeletePgouserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("DeletePgouserHandler parameters [%v]", request)

	username, err := apiserver.Authn(apiserver.DELETE_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.DeletePgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeletePgouser(apiserver.Clientset, username, &request)

	json.NewEncoder(w).Encode(resp)

}

func ShowPgouserHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgousershow pgouserservice pgousershow
	/*```
	Show pgouser information
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Show Pgouser Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/ShowPgouserRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowPgouserResponse"
	var request msgs.ShowPgouserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("ShowPgouserHandler parameters [%v]", request)

	_, err := apiserver.Authn(apiserver.SHOW_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debug("pgouserservice.ShowPgouserHandler POST called")
	resp := msgs.ShowPgouserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowPgouser(apiserver.Clientset, &request)

	json.NewEncoder(w).Encode(resp)

}

func UpdatePgouserHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgouserupdate pgouserservice pgouserupdate
	/*```
	Update a pgouser
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Update Pgouser Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/UpdatePgouserRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/UpdatePgouserResponse"
	log.Debug("pgouserservice.UpdatePgouserHandler called")

	var request msgs.UpdatePgouserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.UPDATE_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.UpdatePgouserResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	resp = UpdatePgouser(apiserver.Clientset, username, &request)
	json.NewEncoder(w).Encode(resp)
}
