package workflowservice

/*
Copyright 2017-2019 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	"net/http"
)

// ShowWorkflowHandler ...
// returns a ShowWorkflowResponse
func ShowWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	vars := mux.Vars(r)
	log.Debugf("workflowservice.ShowWorkflowHandler %v", vars)

	workflowID := vars["id"]

	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debugf("version parameter is [%s]", clientVersion)
	}

	switch r.Method {
	case "GET":
		log.Debug("workflowservice.ShowWorkflowHandler GET called")
	}

	err = apiserver.Authn(apiserver.SHOW_WORKFLOW_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ShowWorkflowResponse{}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp.Results, err = ShowWorkflow(workflowID)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
		}
	}

	json.NewEncoder(w).Encode(resp)
}
