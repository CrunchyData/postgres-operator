package pvcservice

/*
Copyright 2019 Crunchy Data Solutions, Inc.
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

// ShowPVCHandler ...
// returns a ShowPVCResponse
func ShowPVCHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var username, ns string

	var request msgs.ShowPVCRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	pvcname := request.PVCName
	pvcroot := request.PVCRoot
	clientVersion := request.ClientVersion
	nodeLabel := request.NodeLabel
	namespace := request.Namespace

	log.Debugf("ShowPVCHandler parameters pvcroot [%s],  version [%s] namespace [%s] pvcname [%s] nodeLabel [%]", pvcroot, clientVersion, namespace, pvcname, nodeLabel)

	switch r.Method {
	case "GET":
		log.Debug("pvcservice.ShowPVCHandler GET called")
	case "DELETE":
		log.Debug("pvcservice.ShowPVCHandler DELETE called")
	}

	username, err = apiserver.Authn(apiserver.SHOW_PVC_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.ShowPVCResponse{}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp.Results, err = ShowPVC(request.AllFlag, nodeLabel, pvcname, pvcroot, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}
