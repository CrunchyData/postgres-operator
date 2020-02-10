package pgoroleservice

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

func CreatePgoroleHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgorolecreate pgoroleservice pgorolecreate
	/*```
	Create a pgorole
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create pgorole Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreatePgoroleRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreatePgoroleResponse"
	resp := msgs.CreatePgoroleResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	log.Debug("pgoroleservice.CreatePgoroleHandler called")

	var request msgs.CreatePgoroleRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	rolename, err := apiserver.Authn(apiserver.CREATE_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debugf("pgoroleservice.CreatePgoroleHandler got request %v", request)
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	errs := validation.IsDNS1035Label(request.PgoroleName)
	if len(errs) > 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "invalid pgorole name format " + errs[0]
	} else {
		resp = CreatePgorole(apiserver.Clientset, rolename, &request)
	}

	json.NewEncoder(w).Encode(resp)
}

func DeletePgoroleHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgoroledelete pgoroleservice pgoroledelete
	/*```
	Delete a pgorole
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Delete pgorole Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DeletePgoroleRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DeletePgoroleResponse"
	var request msgs.DeletePgoroleRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("DeletePgoroleHandler parameters [%v]", request)

	rolename, err := apiserver.Authn(apiserver.DELETE_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.DeletePgoroleResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeletePgorole(apiserver.Clientset, rolename, &request)

	json.NewEncoder(w).Encode(resp)

}

func ShowPgoroleHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgoroleshow pgoroleservice pgoroleshow
	/*```
	Show pgorole information
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Show pgorole Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/ShowPgoroleRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowPgoroleResponse"
	var request msgs.ShowPgoroleRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("ShowPgoroleHandler parameters [%v]", request)

	_, err := apiserver.Authn(apiserver.SHOW_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debug("pgoroleservice.ShowPgoroleHandler POST called")
	resp := msgs.ShowPgoroleResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowPgorole(apiserver.Clientset, &request)

	json.NewEncoder(w).Encode(resp)

}

func UpdatePgoroleHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgoroleupdate pgoroleservice pgoroleupdate
	/*```
	Delete a pgorole
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Update pgorole Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/UpdatePgoroleRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/UpdatePgoroleResponse"
	log.Debug("pgoroleservice.UpdatePgoroleHandler called")

	var request msgs.UpdatePgoroleRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	rolename, err := apiserver.Authn(apiserver.UPDATE_PGOUSER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.UpdatePgoroleResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	resp = UpdatePgorole(apiserver.Clientset, rolename, &request)
	json.NewEncoder(w).Encode(resp)
}
