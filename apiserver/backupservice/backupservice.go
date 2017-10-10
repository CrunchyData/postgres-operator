package backupservice

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

type BackupDetail struct {
	Name string
}
type ShowBackupResponse struct {
	Items []BackupDetail
}

type CreateBackupRequest struct {
	Name string
}

// pgo backup mycluster
// parameters secretfrom
func CreateBackupHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("backupservice.CreateBackupHandler called")
	var request CreateBackupRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("backupservice.CreateBackupHandler got request " + request.Name)
}

// pgo backup mycluster
// pgo delete backup mycluster
// parameters showsecrets
// parameters selector
// parameters namespace
// parameters postgresversion
// returns a ShowClusterResponse
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
