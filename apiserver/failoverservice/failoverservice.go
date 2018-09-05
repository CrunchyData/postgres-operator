package failoverservice

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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	"net/http"
)

// CreateFailoverHandler ...
// pgo failover mycluster
func CreateFailoverHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Debug("failoverservice.CreateFailoverHandler called")

	var request msgs.CreateFailoverRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	err = apiserver.Authn(apiserver.CREATE_FAILOVER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var resp msgs.CreateFailoverResponse
	if request.ClientVersion != msgs.PGO_VERSION {
		resp = msgs.CreateFailoverResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
	} else {
		resp = CreateFailover(&request)
	}

	json.NewEncoder(w).Encode(resp)
}

// QueryFailoverHandler ...
// pgo failover mycluster --query
func QueryFailoverHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Debug("failoverservice.QueryFailoverHandler called")
	vars := mux.Vars(r)
	name := vars["name"]

	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	err = apiserver.Authn(apiserver.CREATE_FAILOVER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")

	var resp msgs.QueryFailoverResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.QueryFailoverResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
	} else {
		resp = QueryFailover(name)
	}

	json.NewEncoder(w).Encode(resp)
}
