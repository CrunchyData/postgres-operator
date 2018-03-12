package pvcservice

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	"net/http"
)

// ShowPVCHandler ...
// returns a ShowPVCResponse
func ShowPVCHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	vars := mux.Vars(r)
	log.Debugf("pvcervice.ShowPVCHandler %v\n", vars)

	pvcname := vars["pvcname"]

	pvcroot := r.URL.Query().Get("pvcroot")
	if pvcroot != "" {
		log.Debug("pvcroot param was [" + pvcroot + "]")
	}
	switch r.Method {
	case "GET":
		log.Debug("pvcservice.ShowPVCHandler GET called")
	case "DELETE":
		log.Debug("pvcservice.ShowPVCHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ShowPVCResponse{}
	resp.Results, err = ShowPVC(pvcname, pvcroot)
	if err != nil {
		resp.Status.Code = "error"
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}
