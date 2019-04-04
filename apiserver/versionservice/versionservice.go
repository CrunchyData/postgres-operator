package versionservice

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
	log "github.com/sirupsen/logrus"
	"net/http"
)

// VersionHandler ...
// pgo version
func VersionHandler(w http.ResponseWriter, r *http.Request) {

	log.Debug("versionservice.VersionHandler called")

	_, err := apiserver.Authn(apiserver.VERSION_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := Version()

	json.NewEncoder(w).Encode(resp)
}

// HealthHandler ...
func HealthHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := Health()

	json.NewEncoder(w).Encode(resp)
}
