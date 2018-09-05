package main

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
	"crypto/tls"
	"crypto/x509"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/backrestservice"
	"github.com/crunchydata/postgres-operator/apiserver/backupservice"
	"github.com/crunchydata/postgres-operator/apiserver/clusterservice"
	"github.com/crunchydata/postgres-operator/apiserver/configservice"
	"github.com/crunchydata/postgres-operator/apiserver/dfservice"
	"github.com/crunchydata/postgres-operator/apiserver/failoverservice"
	"github.com/crunchydata/postgres-operator/apiserver/ingestservice"
	"github.com/crunchydata/postgres-operator/apiserver/labelservice"
	"github.com/crunchydata/postgres-operator/apiserver/loadservice"
	"github.com/crunchydata/postgres-operator/apiserver/policyservice"
	"github.com/crunchydata/postgres-operator/apiserver/pvcservice"
	"github.com/crunchydata/postgres-operator/apiserver/reloadservice"
	"github.com/crunchydata/postgres-operator/apiserver/statusservice"
	"github.com/crunchydata/postgres-operator/apiserver/upgradeservice"
	"github.com/crunchydata/postgres-operator/apiserver/userservice"
	"github.com/crunchydata/postgres-operator/apiserver/versionservice"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

const serverCert = "/config/server.crt"
const serverKey = "/config/server.key"

func main() {

	PORT := "8443"
	tmp := os.Getenv("PORT")
	if tmp != "" {
		PORT = tmp
	}

	debugFlag := os.Getenv("CRUNCHY_DEBUG")
	if debugFlag == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	tmp = os.Getenv("TLS_NO_VERIFY")
	if tmp == "true" {
		log.Debug("TLS_NO_VERIFY set to true")
	} else {
		tmp = "false"
		log.Debug("TLS_NO_VERIFY set to false")
	}
	tlsNoVerify, _ := strconv.ParseBool(tmp)

	log.Infoln("postgres-operator apiserver starts")

	apiserver.Initialize()

	r := mux.NewRouter()
	r.HandleFunc("/version", versionservice.VersionHandler)
	r.HandleFunc("/policies", policyservice.CreatePolicyHandler)
	r.HandleFunc("/policies/{name}", policyservice.ShowPolicyHandler).Methods("GET")
	//here
	r.HandleFunc("/policiesdelete/{name}", policyservice.DeletePolicyHandler).Methods("GET")
	r.HandleFunc("/pvc/{pvcname}", pvcservice.ShowPVCHandler).Methods("GET")
	r.HandleFunc("/policies/apply", policyservice.ApplyPolicyHandler).Methods("POST")
	r.HandleFunc("/ingest", ingestservice.CreateIngestHandler).Methods("POST")
	r.HandleFunc("/ingest/{name}", ingestservice.ShowIngestHandler).Methods("GET")
	//here
	r.HandleFunc("/ingestdelete/{name}", ingestservice.DeleteIngestHandler).Methods("GET")
	r.HandleFunc("/label", labelservice.LabelHandler).Methods("POST")
	r.HandleFunc("/load", loadservice.LoadHandler).Methods("POST")
	r.HandleFunc("/user", userservice.UserHandler).Methods("POST")
	r.HandleFunc("/users", userservice.CreateUserHandler).Methods("POST")
	r.HandleFunc("/users/{name}", userservice.ShowUserHandler).Methods("GET")
	//here
	r.HandleFunc("/usersdelete/{name}", userservice.DeleteUserHandler).Methods("GET")
	r.HandleFunc("/upgrades", upgradeservice.CreateUpgradeHandler).Methods("POST")
	r.HandleFunc("/upgrades/{name}", upgradeservice.ShowUpgradeHandler).Methods("GET")
	//here
	r.HandleFunc("/upgradesdelete/{name}", upgradeservice.DeleteUpgradeHandler).Methods("GET")
	r.HandleFunc("/clusters", clusterservice.CreateClusterHandler).Methods("POST")
	r.HandleFunc("/clusters/{name}", clusterservice.ShowClusterHandler).Methods("GET")
	//here
	r.HandleFunc("/clustersdelete/{name}", clusterservice.DeleteClusterHandler).Methods("GET")
	r.HandleFunc("/clusters/test/{name}", clusterservice.TestClusterHandler)
	r.HandleFunc("/clusters/scale/{name}", clusterservice.ScaleClusterHandler)
	r.HandleFunc("/scale/{name}", clusterservice.ScaleQueryHandler).Methods("GET")
	r.HandleFunc("/scaledown/{name}", clusterservice.ScaleDownHandler).Methods("GET")
	r.HandleFunc("/status", statusservice.StatusHandler)
	r.HandleFunc("/df/{name}", dfservice.DfHandler)
	r.HandleFunc("/config", configservice.ShowConfigHandler)

	r.HandleFunc("/backups/{name}", backupservice.ShowBackupHandler).Methods("GET")
	//here
	r.HandleFunc("/backupsdelete/{name}", backupservice.DeleteBackupHandler).Methods("GET")
	r.HandleFunc("/backups", backupservice.CreateBackupHandler).Methods("POST")
	r.HandleFunc("/backrestbackup", backrestservice.CreateBackupHandler).Methods("POST")
	r.HandleFunc("/backrest/{name}", backrestservice.ShowBackrestHandler).Methods("GET")
	r.HandleFunc("/restore", backrestservice.RestoreHandler).Methods("POST")
	r.HandleFunc("/reload", reloadservice.ReloadHandler).Methods("POST")
	r.HandleFunc("/failover", failoverservice.CreateFailoverHandler).Methods("POST")
	r.HandleFunc("/failover/{name}", failoverservice.QueryFailoverHandler).Methods("GET")

	caCert, err := ioutil.ReadFile(serverCert)
	if err != nil {
		log.Fatal(err)
		log.Error("could not read " + serverCert)
		os.Exit(2)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cfg := &tls.Config{
		//ClientAuth: tls.RequireAndVerifyClientCert,
		//specify pgo-apiserver in the CN....then, add ServerName: "pgo-apiserver",
		ServerName:         "pgo-apiserver",
		InsecureSkipVerify: tlsNoVerify,
		ClientCAs:          caCertPool,
	}

	log.Info("listening on port " + PORT)

	srv := &http.Server{
		Addr:      ":" + PORT,
		Handler:   r,
		TLSConfig: cfg,
	}

	_, err = ioutil.ReadFile(serverKey)
	if err != nil {
		log.Fatal(err)
		log.Error("could not read " + serverKey)
		os.Exit(2)
	}

	log.Fatal(srv.ListenAndServeTLS(serverCert, serverKey))
}
