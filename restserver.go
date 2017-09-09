package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/clusterservice"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {

	log.Infoln("restserver starts")
	r := mux.NewRouter()
	r.HandleFunc("/clusters", clusterservice.CreateClusterHandler)
	r.HandleFunc("/clusters/{name}", clusterservice.ShowClusterHandler)
	r.HandleFunc("/clusters/test/{name}", clusterservice.TestClusterHandler)
	log.Fatal(http.ListenAndServe(":8080", r))
}
