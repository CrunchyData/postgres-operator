package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/backupservice"
	"github.com/crunchydata/postgres-operator/clusterservice"
	"github.com/crunchydata/postgres-operator/policyservice"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {

	log.Infoln("restserver starts")
	r := mux.NewRouter()
	r.HandleFunc("/policies", policyservice.CreatePolicyHandler)
	r.HandleFunc("/policies/{name}", policyservice.ShowPolicyHandler).Methods("GET", "DELETE")
	r.HandleFunc("/policies/apply/{name}", policyservice.ApplyPolicyHandler)
	r.HandleFunc("/clusters", clusterservice.CreateClusterHandler)
	r.HandleFunc("/clusters/{name}", clusterservice.ShowClusterHandler).Methods("GET", "DELETE")
	r.HandleFunc("/clusters/test/{name}", clusterservice.TestClusterHandler)
	r.HandleFunc("/backups/{name}", backupservice.ShowBackupHandler).Methods("GET", "DELETE")
	log.Fatal(http.ListenAndServe(":8080", r))
}
