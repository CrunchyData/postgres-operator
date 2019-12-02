package routing

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
	"github.com/crunchydata/postgres-operator/apiserver/backrestservice"
	"github.com/crunchydata/postgres-operator/apiserver/backupservice"
	"github.com/crunchydata/postgres-operator/apiserver/benchmarkservice"
	"github.com/crunchydata/postgres-operator/apiserver/catservice"
	"github.com/crunchydata/postgres-operator/apiserver/cloneservice"
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
	"github.com/crunchydata/postgres-operator/apiserver/pgoroleservice"
	"github.com/crunchydata/postgres-operator/apiserver/pgouserservice"
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
)

// RegisterAllRoutes adds all routes supported by the apiserver to the
// provided router
func RegisterAllRoutes(r *mux.Router) {
	r.HandleFunc("/version", versionservice.VersionHandler)
	r.HandleFunc("/health", versionservice.HealthHandler)
	r.HandleFunc("/policies", policyservice.CreatePolicyHandler)
	r.HandleFunc("/showpolicies", policyservice.ShowPolicyHandler).Methods("POST")
	r.HandleFunc("/policiesdelete", policyservice.DeletePolicyHandler).Methods("POST")
	r.HandleFunc("/workflow/{id}", workflowservice.ShowWorkflowHandler).Methods("GET")
	r.HandleFunc("/showpvc", pvcservice.ShowPVCHandler).Methods("POST")
	r.HandleFunc("/pgouserupdate", pgouserservice.UpdatePgouserHandler).Methods("POST")
	r.HandleFunc("/pgouserdelete", pgouserservice.DeletePgouserHandler).Methods("POST")
	r.HandleFunc("/pgousercreate", pgouserservice.CreatePgouserHandler).Methods("POST")
	r.HandleFunc("/pgousershow", pgouserservice.ShowPgouserHandler).Methods("POST")
	r.HandleFunc("/pgoroleupdate", pgoroleservice.UpdatePgoroleHandler).Methods("POST")
	r.HandleFunc("/pgoroledelete", pgoroleservice.DeletePgoroleHandler).Methods("POST")
	r.HandleFunc("/pgorolecreate", pgoroleservice.CreatePgoroleHandler).Methods("POST")
	r.HandleFunc("/pgoroleshow", pgoroleservice.ShowPgoroleHandler).Methods("POST")
	r.HandleFunc("/policies/apply", policyservice.ApplyPolicyHandler).Methods("POST")
	r.HandleFunc("/label", labelservice.LabelHandler).Methods("POST")
	r.HandleFunc("/labeldelete", labelservice.DeleteLabelHandler).Methods("POST")
	r.HandleFunc("/load", loadservice.LoadHandler).Methods("POST")

	r.HandleFunc("/userupdate", userservice.UpdateUserHandler).Methods("POST")
	r.HandleFunc("/usercreate", userservice.CreateUserHandler).Methods("POST")
	r.HandleFunc("/usershow", userservice.ShowUserHandler).Methods("POST")
	r.HandleFunc("/userdelete", userservice.DeleteUserHandler).Methods("POST")

	r.HandleFunc("/upgrades", upgradeservice.CreateUpgradeHandler).Methods("POST")
	r.HandleFunc("/clusters", clusterservice.CreateClusterHandler).Methods("POST")
	r.HandleFunc("/showclusters", clusterservice.ShowClusterHandler).Methods("POST")
	r.HandleFunc("/clustersdelete", clusterservice.DeleteClusterHandler).Methods("POST")
	r.HandleFunc("/clustersupdate", clusterservice.UpdateClusterHandler).Methods("POST")
	r.HandleFunc("/testclusters", clusterservice.TestClusterHandler).Methods("POST")
	r.HandleFunc("/clusters/scale/{name}", clusterservice.ScaleClusterHandler)
	r.HandleFunc("/scale/{name}", clusterservice.ScaleQueryHandler).Methods("GET")
	r.HandleFunc("/scaledown/{name}", clusterservice.ScaleDownHandler).Methods("GET")
	r.HandleFunc("/status", statusservice.StatusHandler)
	r.HandleFunc("/df/{name}", dfservice.DfHandler)
	r.HandleFunc("/config", configservice.ShowConfigHandler)
	r.HandleFunc("/namespace", namespaceservice.ShowNamespaceHandler).Methods("POST")
	r.HandleFunc("/namespacedelete", namespaceservice.DeleteNamespaceHandler).Methods("POST")
	r.HandleFunc("/namespacecreate", namespaceservice.CreateNamespaceHandler).Methods("POST")
	r.HandleFunc("/namespaceupdate", namespaceservice.UpdateNamespaceHandler).Methods("POST")

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

	//schedule
	r.HandleFunc("/schedule", scheduleservice.CreateScheduleHandler).Methods("POST")
	r.HandleFunc("/scheduledelete", scheduleservice.DeleteScheduleHandler).Methods("POST")
	r.HandleFunc("/scheduleshow", scheduleservice.ShowScheduleHandler).Methods("POST")

	//benchmark
	r.HandleFunc("/benchmark", benchmarkservice.CreateBenchmarkHandler).Methods("POST")
	r.HandleFunc("/benchmarkdelete", benchmarkservice.DeleteBenchmarkHandler).Methods("POST")
	r.HandleFunc("/benchmarkshow", benchmarkservice.ShowBenchmarkHandler).Methods("POST")

	//clone
	r.HandleFunc("/clone", cloneservice.CloneHandler).Methods("POST")}
