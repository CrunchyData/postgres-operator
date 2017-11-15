package versionservice

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	"net/http"
)

// VersionHandler ...
// pgo version
func VersionHandler(w http.ResponseWriter, r *http.Request) {

	log.Debug("versionservice.VersionHandler called")

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := Version()

	json.NewEncoder(w).Encode(resp)
}

// VersionHandler ...
// pgo version
func AuthTestHandler(w http.ResponseWriter, r *http.Request) {

	log.Debug("versionservice.AuthTestHandler called")

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	username, password, authOK := r.BasicAuth()
	if authOK == false {
		http.Error(w, "Not authorized", 401)
		return
	}

	log.Debugf("versionservice.AuthTestHandler username=[%s] password=[%s]\n", username, password)

	if !apiserver.BasicAuthCheck(username, password) {
		//if username != "username" || password != "password" {
		http.Error(w, "Not authenticated in apiserver", 401)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := Version()

	json.NewEncoder(w).Encode(resp)
}
