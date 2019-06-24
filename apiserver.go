package main

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
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/backrestservice"
	"github.com/crunchydata/postgres-operator/apiserver/backupservice"
	"github.com/crunchydata/postgres-operator/apiserver/benchmarkservice"
	"github.com/crunchydata/postgres-operator/apiserver/catservice"
	"github.com/crunchydata/postgres-operator/apiserver/clusterservice"
	"github.com/crunchydata/postgres-operator/apiserver/configservice"
	"github.com/crunchydata/postgres-operator/apiserver/dfservice"
	"github.com/crunchydata/postgres-operator/apiserver/failoverservice"
	"github.com/crunchydata/postgres-operator/apiserver/labelservice"
	"github.com/crunchydata/postgres-operator/apiserver/loadservice"
	"github.com/crunchydata/postgres-operator/apiserver/lsservice"
	"github.com/crunchydata/postgres-operator/apiserver/namespaceservice"
	"github.com/crunchydata/postgres-operator/apiserver/pgbouncerservice"
	"github.com/crunchydata/postgres-operator/apiserver/pgdumpservice"
	"github.com/crunchydata/postgres-operator/apiserver/pgpoolservice"
	"github.com/crunchydata/postgres-operator/apiserver/policyservice"
	"github.com/crunchydata/postgres-operator/apiserver/pvcservice"
	"github.com/crunchydata/postgres-operator/apiserver/reloadservice"
	"github.com/crunchydata/postgres-operator/apiserver/scheduleservice"
	"github.com/crunchydata/postgres-operator/apiserver/statusservice"
	"github.com/crunchydata/postgres-operator/apiserver/upgradeservice"
	"github.com/crunchydata/postgres-operator/apiserver/userservice"
	"github.com/crunchydata/postgres-operator/apiserver/versionservice"
	"github.com/crunchydata/postgres-operator/apiserver/workflowservice"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const serverCertPath = "/tmp/server.crt"
const serverKeyPath = "/tmp/server.key"

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
	r.HandleFunc("/health", versionservice.HealthHandler)
	r.HandleFunc("/policies", policyservice.CreatePolicyHandler)
	r.HandleFunc("/showpolicies", policyservice.ShowPolicyHandler).Methods("POST")
	//here
	r.HandleFunc("/policiesdelete", policyservice.DeletePolicyHandler).Methods("POST")
	r.HandleFunc("/workflow/{id}", workflowservice.ShowWorkflowHandler).Methods("GET")
	r.HandleFunc("/showpvc", pvcservice.ShowPVCHandler).Methods("POST")
	r.HandleFunc("/policies/apply", policyservice.ApplyPolicyHandler).Methods("POST")
	r.HandleFunc("/label", labelservice.LabelHandler).Methods("POST")
	r.HandleFunc("/labeldelete", labelservice.DeleteLabelHandler).Methods("POST")
	r.HandleFunc("/load", loadservice.LoadHandler).Methods("POST")
	r.HandleFunc("/user", userservice.UserHandler).Methods("POST")
	r.HandleFunc("/users", userservice.CreateUserHandler).Methods("POST")
	r.HandleFunc("/users/{name}", userservice.ShowUserHandler).Methods("GET")
	//here
	r.HandleFunc("/usersdelete/{name}", userservice.DeleteUserHandler).Methods("GET")
	r.HandleFunc("/upgrades", upgradeservice.CreateUpgradeHandler).Methods("POST")
	r.HandleFunc("/clusters", clusterservice.CreateClusterHandler).Methods("POST")
	r.HandleFunc("/showclusters", clusterservice.ShowClusterHandler).Methods("POST")
	//here
	r.HandleFunc("/clustersdelete", clusterservice.DeleteClusterHandler).Methods("POST")
	r.HandleFunc("/clustersupdate/{name}", clusterservice.UpdateClusterHandler).Methods("GET")
	r.HandleFunc("/testclusters", clusterservice.TestClusterHandler).Methods("POST")
	r.HandleFunc("/clusters/scale/{name}", clusterservice.ScaleClusterHandler)
	r.HandleFunc("/scale/{name}", clusterservice.ScaleQueryHandler).Methods("GET")
	r.HandleFunc("/scaledown/{name}", clusterservice.ScaleDownHandler).Methods("GET")
	r.HandleFunc("/status", statusservice.StatusHandler)
	r.HandleFunc("/df/{name}", dfservice.DfHandler)
	r.HandleFunc("/config", configservice.ShowConfigHandler)
	r.HandleFunc("/namespace", namespaceservice.ShowNamespaceHandler)

	// backups / backrest
	r.HandleFunc("/backups/{name}", backupservice.ShowBackupHandler).Methods("GET")
	r.HandleFunc("/backupsdelete/{name}", backupservice.DeleteBackupHandler).Methods("GET")
	r.HandleFunc("/backups", backupservice.CreateBackupHandler).Methods("POST")
	r.HandleFunc("/pgbasebackuprestore", backupservice.RestoreHandler).Methods("POST")
	r.HandleFunc("/backrestbackup", backrestservice.CreateBackupHandler).Methods("POST")
	r.HandleFunc("/backrest/{name}", backrestservice.ShowBackrestHandler).Methods("GET")
	r.HandleFunc("/restore", backrestservice.RestoreHandler).Methods("POST")

	// pgdump
	r.HandleFunc("/pgdumpbackup", pgdumpservice.BackupHandler).Methods("POST")
	r.HandleFunc("/pgdump/{name}", pgdumpservice.ShowDumpHandler).Methods("GET")
	r.HandleFunc("/pgdumprestore", pgdumpservice.RestoreHandler).Methods("POST")

	r.HandleFunc("/reload", reloadservice.ReloadHandler).Methods("POST")
	r.HandleFunc("/ls", lsservice.LsHandler).Methods("POST")
	r.HandleFunc("/cat", catservice.CatHandler).Methods("POST")
	r.HandleFunc("/failover", failoverservice.CreateFailoverHandler).Methods("POST")
	r.HandleFunc("/failover/{name}", failoverservice.QueryFailoverHandler).Methods("GET")
	r.HandleFunc("/pgbouncer", pgbouncerservice.CreatePgbouncerHandler).Methods("POST")
	r.HandleFunc("/pgbouncer", pgbouncerservice.DeletePgbouncerHandler).Methods("DELETE")
	r.HandleFunc("/pgbouncerdelete", pgbouncerservice.DeletePgbouncerHandler).Methods("POST")
	r.HandleFunc("/pgpool", pgpoolservice.CreatePgpoolHandler).Methods("POST")
	r.HandleFunc("/pgpooldelete", pgpoolservice.DeletePgpoolHandler).Methods("POST")

	//schedule
	r.HandleFunc("/schedule", scheduleservice.CreateScheduleHandler).Methods("POST")
	r.HandleFunc("/scheduledelete", scheduleservice.DeleteScheduleHandler).Methods("POST")
	r.HandleFunc("/scheduleshow", scheduleservice.ShowScheduleHandler).Methods("POST")

	//benchmark
	r.HandleFunc("/benchmark", benchmarkservice.CreateBenchmarkHandler).Methods("POST")
	r.HandleFunc("/benchmarkdelete", benchmarkservice.DeleteBenchmarkHandler).Methods("POST")
	r.HandleFunc("/benchmarkshow", benchmarkservice.ShowBenchmarkHandler).Methods("POST")

	err := apiserver.GetTLS(serverCertPath, serverKeyPath)
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}

	var caCert []byte

	caCert, err = ioutil.ReadFile(serverCertPath)
	if err != nil {
		log.Fatal(err)
		log.Error("could not read " + serverCertPath)
		os.Exit(2)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cfg := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		//specify pgo-apiserver in the CN....then, add ServerName: "pgo-apiserver",
		ServerName:         "pgo-apiserver",
		InsecureSkipVerify: tlsNoVerify,
		ClientCAs:          caCertPool,
		MinVersion:         tls.VersionTLS11,
	}

	log.Info("listening on port " + PORT)

	srv := &http.Server{
		Addr:      ":" + PORT,
		Handler:   r,
		TLSConfig: cfg,
	}

	_, err = ioutil.ReadFile(serverKeyPath)
	if err != nil {
		log.Fatal(err)
		log.Error("could not read " + serverKeyPath)
		os.Exit(2)
	}

	log.Fatal(srv.ListenAndServeTLS(serverCertPath, serverKeyPath))
}
