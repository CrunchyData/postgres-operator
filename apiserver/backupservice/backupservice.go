package backupservice

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
	"github.com/gorilla/mux"
	"net/http"
)

// BackupDetail ...
type BackupDetail struct {
	Name string
}

// ShowBackupResponse ...
type ShowBackupResponse struct {
	Items []BackupDetail
}

// CreateBackupRequest ...
type CreateBackupRequest struct {
	Name string
}

// CreateBackupHandler ...
func CreateBackupHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("backupservice.CreateBackupHandler called")
	var request CreateBackupRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("backupservice.CreateBackupHandler got request " + request.Name)
}

// ShowBackupHandler ...
func ShowBackupHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("backupservice.ShowBackupHandler called")
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	switch r.Method {
	case "GET":
		log.Infoln("backupservice.ShowBackupHandler GET called")
	case "DELETE":
		log.Infoln("backupservice.ShowBackupHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := new(ShowBackupResponse)
	resp.Items = []BackupDetail{}
	c := BackupDetail{}
	c.Name = "somecluster"
	resp.Items = append(resp.Items, c)

	json.NewEncoder(w).Encode(resp)
}
