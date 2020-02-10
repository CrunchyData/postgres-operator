package routing

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	RegisterBackrestSvcRoutes(r)
	RegisterBackupSvcRoutes(r)
	RegisterBenchmarkSvcRoutes(r)
	RegisterCatSvcRoutes(r)
	RegisterCloneSvcRoutes(r)
	RegisterClusterSvcRoutes(r)
	RegisterConfigSvcRoutes(r)
	RegisterDfSvcRoutes(r)
	RegisterFailoverSvcRoutes(r)
	RegisterLabelSvcRoutes(r)
	RegisterLoadSvcRoutes(r)
	RegisterLsSvcRoutes(r)
	RegisterNamespaceSvcRoutes(r)
	RegisterPGBouncerSvcRoutes(r)
	RegisterPGDumpSvcRoutes(r)
	RegisterPGORoleSvcRoutes(r)
	RegisterPGOUserSvcRoutes(r)
	RegisterPolicySvcRoutes(r)
	RegisterPVCSvcRoutes(r)
	RegisterReloadSvcRoutes(r)
	RegisterScheduleSvcRoutes(r)
	RegisterStatusSvcRoutes(r)
	RegisterUpgradeSvcRoutes(r)
	RegisterUserSvcRoutes(r)
	RegisterVersionSvcRoutes(r)
	RegisterWorkflowSvcRoutes(r)
}

// RegisterBackrestSvcRoutes registers all routes from the Backrest Service
func RegisterBackrestSvcRoutes(r *mux.Router) {
	r.HandleFunc("/backrestbackup", backrestservice.CreateBackupHandler).Methods("POST")
	r.HandleFunc("/backrest/{name}", backrestservice.ShowBackrestHandler).Methods("GET")
	r.HandleFunc("/restore", backrestservice.RestoreHandler).Methods("POST")
}

// RegisterBackupSvcRoutes registers all routes from the Backup Service
func RegisterBackupSvcRoutes(r *mux.Router) {
	r.HandleFunc("/backups/{name}", backupservice.ShowBackupHandler).Methods("GET")
	r.HandleFunc("/backupsdelete/{name}", backupservice.DeleteBackupHandler).Methods("GET")
	r.HandleFunc("/backups", backupservice.CreateBackupHandler).Methods("POST")
	r.HandleFunc("/pgbasebackuprestore", backupservice.RestoreHandler).Methods("POST")
}

// RegisterBenchmarkSvcRoutes registers all routes from the Benchmark Service
func RegisterBenchmarkSvcRoutes(r *mux.Router) {
	r.HandleFunc("/benchmark", benchmarkservice.CreateBenchmarkHandler).Methods("POST")
	r.HandleFunc("/benchmarkdelete", benchmarkservice.DeleteBenchmarkHandler).Methods("POST")
	r.HandleFunc("/benchmarkshow", benchmarkservice.ShowBenchmarkHandler).Methods("POST")
}

// RegisterCatSvcRoutes registers all routes from the Cat Service
func RegisterCatSvcRoutes(r *mux.Router) {
	r.HandleFunc("/cat", catservice.CatHandler).Methods("POST")
}

// RegisterCloneSvcRoutes registers all routes from the Clone Service
func RegisterCloneSvcRoutes(r *mux.Router) {
	r.HandleFunc("/clone", cloneservice.CloneHandler).Methods("POST")
}

// RegisterClusterSvcRoutes registers all routes from the Cluster Service
func RegisterClusterSvcRoutes(r *mux.Router) {
	r.HandleFunc("/clusters", clusterservice.CreateClusterHandler).Methods("POST")
	r.HandleFunc("/showclusters", clusterservice.ShowClusterHandler).Methods("POST")
	r.HandleFunc("/clustersdelete", clusterservice.DeleteClusterHandler).Methods("POST")
	r.HandleFunc("/clustersupdate", clusterservice.UpdateClusterHandler).Methods("POST")
	r.HandleFunc("/testclusters", clusterservice.TestClusterHandler).Methods("POST")
	r.HandleFunc("/clusters/scale/{name}", clusterservice.ScaleClusterHandler)
	r.HandleFunc("/scale/{name}", clusterservice.ScaleQueryHandler).Methods("GET")
	r.HandleFunc("/scaledown/{name}", clusterservice.ScaleDownHandler).Methods("GET")
}

// RegisterConfigSvcRoutes registers all routes from the Config Service
func RegisterConfigSvcRoutes(r *mux.Router) {
	r.HandleFunc("/config", configservice.ShowConfigHandler)
}

// RegisterDfSvcRoutes registers all routes from the Df Service
func RegisterDfSvcRoutes(r *mux.Router) {
	r.HandleFunc("/df/{name}", dfservice.DfHandler)
}

// RegisterFailoverSvcRoutes registers all routes from the Failover Service
func RegisterFailoverSvcRoutes(r *mux.Router) {
	r.HandleFunc("/failover", failoverservice.CreateFailoverHandler).Methods("POST")
	r.HandleFunc("/failover/{name}", failoverservice.QueryFailoverHandler).Methods("GET")
}

// RegisterLabelSvcRoutes registers all routes from the Label Service
func RegisterLabelSvcRoutes(r *mux.Router) {
	r.HandleFunc("/label", labelservice.LabelHandler).Methods("POST")
	r.HandleFunc("/labeldelete", labelservice.DeleteLabelHandler).Methods("POST")
}

// RegisterLoadSvcRoutes registers all routes from the Load Service
func RegisterLoadSvcRoutes(r *mux.Router) {
	r.HandleFunc("/load", loadservice.LoadHandler).Methods("POST")
}

// RegisterLsSvcRoutes registers all routes from the Ls Service
func RegisterLsSvcRoutes(r *mux.Router) {
	r.HandleFunc("/ls", lsservice.LsHandler).Methods("POST")
}

// RegisterNamespaceSvcRoutes registers all routes from the Namespace Service
func RegisterNamespaceSvcRoutes(r *mux.Router) {
	r.HandleFunc("/namespace", namespaceservice.ShowNamespaceHandler).Methods("POST")
	r.HandleFunc("/namespacedelete", namespaceservice.DeleteNamespaceHandler).Methods("POST")
	r.HandleFunc("/namespacecreate", namespaceservice.CreateNamespaceHandler).Methods("POST")
	r.HandleFunc("/namespaceupdate", namespaceservice.UpdateNamespaceHandler).Methods("POST")
}

// RegisterPGBouncerSvcRoutes registers all routes from the PGBouncer Service
func RegisterPGBouncerSvcRoutes(r *mux.Router) {
	r.HandleFunc("/pgbouncer", pgbouncerservice.CreatePgbouncerHandler).Methods("POST")
	r.HandleFunc("/pgbouncer", pgbouncerservice.DeletePgbouncerHandler).Methods("DELETE")
	r.HandleFunc("/pgbouncerdelete", pgbouncerservice.DeletePgbouncerHandler).Methods("POST")
}

// RegisterPGDumpSvcRoutes registers all routes from the PGDump Service
func RegisterPGDumpSvcRoutes(r *mux.Router) {
	r.HandleFunc("/pgdumpbackup", pgdumpservice.BackupHandler).Methods("POST")
	r.HandleFunc("/pgdump/{name}", pgdumpservice.ShowDumpHandler).Methods("GET")
	r.HandleFunc("/pgdumprestore", pgdumpservice.RestoreHandler).Methods("POST")
}

// RegisterPGORoleSvcRoutes registers all routes from the PGORole Service
func RegisterPGORoleSvcRoutes(r *mux.Router) {
	r.HandleFunc("/pgoroleupdate", pgoroleservice.UpdatePgoroleHandler).Methods("POST")
	r.HandleFunc("/pgoroledelete", pgoroleservice.DeletePgoroleHandler).Methods("POST")
	r.HandleFunc("/pgorolecreate", pgoroleservice.CreatePgoroleHandler).Methods("POST")
	r.HandleFunc("/pgoroleshow", pgoroleservice.ShowPgoroleHandler).Methods("POST")
}

// RegisterPGOUserSvcRoutes registers all routes from the PGOUser Service
func RegisterPGOUserSvcRoutes(r *mux.Router) {
	r.HandleFunc("/pgouserupdate", pgouserservice.UpdatePgouserHandler).Methods("POST")
	r.HandleFunc("/pgouserdelete", pgouserservice.DeletePgouserHandler).Methods("POST")
	r.HandleFunc("/pgousercreate", pgouserservice.CreatePgouserHandler).Methods("POST")
	r.HandleFunc("/pgousershow", pgouserservice.ShowPgouserHandler).Methods("POST")
}

// RegisterPolicySvcRoutes registers all routes from the Policy Service
func RegisterPolicySvcRoutes(r *mux.Router) {
	r.HandleFunc("/policies", policyservice.CreatePolicyHandler)
	r.HandleFunc("/showpolicies", policyservice.ShowPolicyHandler).Methods("POST")
	r.HandleFunc("/policiesdelete", policyservice.DeletePolicyHandler).Methods("POST")
	r.HandleFunc("/policies/apply", policyservice.ApplyPolicyHandler).Methods("POST")
}

// RegisterPVCSvcRoutes registers all routes from the PVC Service
func RegisterPVCSvcRoutes(r *mux.Router) {
	r.HandleFunc("/showpvc", pvcservice.ShowPVCHandler).Methods("POST")
}

// RegisterReloadSvcRoutes registers all routes from the Reload Service
func RegisterReloadSvcRoutes(r *mux.Router) {
	r.HandleFunc("/reload", reloadservice.ReloadHandler).Methods("POST")
}

// RegisterScheduleSvcRoutes registers all routes from the Schedule Service
func RegisterScheduleSvcRoutes(r *mux.Router) {
	r.HandleFunc("/schedule", scheduleservice.CreateScheduleHandler).Methods("POST")
	r.HandleFunc("/scheduledelete", scheduleservice.DeleteScheduleHandler).Methods("POST")
	r.HandleFunc("/scheduleshow", scheduleservice.ShowScheduleHandler).Methods("POST")
}

// RegisterStatusSvcRoutes registers all routes from the Status Service
func RegisterStatusSvcRoutes(r *mux.Router) {
	r.HandleFunc("/status", statusservice.StatusHandler)
}

// RegisterUpgradeSvcRoutes registers all routes from the Upgrade Service
func RegisterUpgradeSvcRoutes(r *mux.Router) {
	r.HandleFunc("/upgrades", upgradeservice.CreateUpgradeHandler).Methods("POST")
}

// RegisterUserSvcRoutes registers all routes from the User Service
func RegisterUserSvcRoutes(r *mux.Router) {
	r.HandleFunc("/userupdate", userservice.UpdateUserHandler).Methods("POST")
	r.HandleFunc("/usercreate", userservice.CreateUserHandler).Methods("POST")
	r.HandleFunc("/usershow", userservice.ShowUserHandler).Methods("POST")
	r.HandleFunc("/userdelete", userservice.DeleteUserHandler).Methods("POST")
}

// RegisterVersionSvcRoutes registers all routes from the Version Service
func RegisterVersionSvcRoutes(r *mux.Router) {
	r.HandleFunc("/version", versionservice.VersionHandler)
	r.HandleFunc("/health", versionservice.HealthHandler)
	r.HandleFunc("/healthz", versionservice.HealthyHandler)
}

// RegisterWorkflowSvcRoutes registers all routes from the Workflow Service
func RegisterWorkflowSvcRoutes(r *mux.Router) {
	r.HandleFunc("/workflow/{id}", workflowservice.ShowWorkflowHandler).Methods("GET")
}
