package ingestservice

/*
Copyright 2018 Crunchy Data Solutions, Inc.
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
	apiserver "github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	//"k8s.io/apimachinery/pkg/util/validation"
	"net/http"
)

// CreateIngestHandler ...
// pgo create ingest
// parameters secretfrom
func CreateIngestHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("ingestservice.CreateIngestHandler called")

	var request msgs.CreateIngestRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	err := apiserver.Authn(apiserver.CREATE_INGEST_PERM, w, r)
	if err != nil {
		return
	}

	log.Debug("ingestservice.CreateIngestHandler got request " + request.Name)
	resp := CreateIngest(apiserver.RESTClient, &request)

	json.NewEncoder(w).Encode(resp)
}

func ShowIngestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("ingestservice.ShowIngestHandler %v\n", vars)

	ingestName := vars["name"]

	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	log.Debug("ingestservice.ShowIngestHandler GET called")

	err := apiserver.Authn(apiserver.SHOW_INGEST_PERM, w, r)
	if err != nil {
		return
	}

	var resp msgs.ShowIngestResponse

	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ShowIngestResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}

	} else {
		resp = ShowIngest(ingestName)
	}
	json.NewEncoder(w).Encode(resp)

}

func DeleteIngestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("ingestservice.DeleteIngestHandler %v\n", vars)

	ingestName := vars["name"]
	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	log.Debug("ingestservice.DeleteIngestHandler called")

	err := apiserver.Authn(apiserver.DELETE_INGEST_PERM, w, r)
	if err != nil {
		return
	}

	var resp msgs.DeleteIngestResponse

	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.DeleteIngestResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}

	} else {

		resp = DeleteIngest(ingestName)
	}
	json.NewEncoder(w).Encode(resp)

}
