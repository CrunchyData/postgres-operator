package reloadservice

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
	//"github.com/gorilla/mux"
	"net/http"
)

// ReloadHandler ...
// pgo reload all
// pgo reload --selector=name=mycluster
// pgo reload mycluster
func ReloadHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Debug("reloadservice.ReloadHandler called")

	var request msgs.ReloadRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	err = apiserver.Authn(apiserver.RELOAD_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := Reload(&request)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}
