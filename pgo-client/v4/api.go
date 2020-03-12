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

package v4

import (
	"context"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
)

// ClusterAPI is the interface that must be implemented to interact with the
// postgres-operator cluster service.
type ClusterAPI interface {
	// CreateCluster creates a cluster.
	CreateCluster(context.Context, msgs.CreateClusterRequest) (msgs.CreateClusterResponse, error)
	// DeleteCluster deletes a cluster.
	DeleteCluster(context.Context, msgs.DeleteClusterRequest) (msgs.DeleteClusterResponse, error)
	// ShowCluster retrieves information about a cluster.
	ShowCluster(context.Context, msgs.ShowClusterRequest) (msgs.ShowClusterResponse, error)
	// UpdateCluster updates a cluster.
	UpdateCluster(context.Context, msgs.UpdateClusterRequest) (msgs.UpdateClusterResponse, error)
	// Clone creates a clone of a cluster.
	Clone(context.Context, msgs.CloneRequest) (msgs.CloneResponse, error)
	// Reload reloads a cluster.
	Reload(context.Context, msgs.ReloadRequest) (msgs.ReloadResponse, error)
	// Load loads a cluster.
	Load(context.Context, msgs.LoadRequest) (msgs.LoadResponse, error)
	// ScaleCluster scales a cluster.
	ScaleCluster(context.Context, msgs.ClusterScaleRequest) (msgs.ClusterScaleResponse, error)
	// ScaleQuery queries for scaling information about a cluster.
	ScaleQuery(ctx context.Context, r msgs.ScaleQueryRequest) (msgs.ScaleQueryResponse, error)
	// ScaleDownCluster scales down a cluster.
	ScaleDownCluster(ctx context.Context, r msgs.ScaleDownRequest) (msgs.ScaleDownResponse, error)
	// CreateUpgrade creates a cluster upgrade.
	CreateUpgrade(ctx context.Context, r msgs.CreateUpgradeRequest) (msgs.CreateUpgradeResponse, error)
}

// BackupAPI is the interface that must be implemented to interact with the
// postgres-operator backup service.
type BackupAPI interface {
	// CreateBackrestBackup creates a PG BackRest based backup for a cluster.
	CreateBackrestBackup(context.Context, msgs.CreateBackrestBackupRequest) (msgs.CreateBackrestBackupResponse, error)
	// ShowBackrest retrieves information about a PG BackRest backup for a cluster.
	ShowBackrest(context.Context, msgs.ShowBackrestRequest) (msgs.ShowBackrestResponse, error)
	// RestorebackrestBackup restores a cluster from a PG BackRest backup.
	RestoreBackrestBackup(context.Context, msgs.RestoreRequest) (msgs.RestoreResponse, error)
	// CreatePGDumpBackup creates a PGDump backup for a cluster.
	CreatePGDumpBackup(context.Context, msgs.CreatePGDumpRequest) (msgs.CreatePGDumpResponse, error)
	// ShowPGDumpBackup retrieves information about a PGDump backup for a cluster.
	ShowPGDumpBackup(context.Context, msgs.ShowPGDumpRequest) (msgs.ShowBackupResponse, error)
	// RestorePGDumpBackup restores a cluster from a PGDump backup.
	RestorePGDumpBackup(context.Context, msgs.PgRestoreRequest) (msgs.RestoreResponse, error)
}

// FailoverAPI is the interface that must be implemented to interact with the
// postgres-operator failover service.
type FailoverAPI interface {
	// CreateFailover creates a failover for a cluster.
	CreateFailover(context.Context, msgs.CreateFailoverRequest) (msgs.CreateFailoverResponse, error)
	// QueryFailover queries a failover status for a cluster.
	QueryFailover(context.Context, msgs.QueryFailoverRequest) (msgs.QueryFailoverResponse, error)
}

// NamespaceAPI is the interface that must be implemeted to interact with the
// postgres-operator namespace service.
type NamespaceAPI interface {
	// CreateNamespace creates a managed namespace.
	CreateNamespace(context.Context, msgs.CreateNamespaceRequest) (msgs.CreateNamespaceResponse, error)
	// DeleteNamespace deletes a managed namespace.
	DeleteNamespace(context.Context, msgs.DeleteNamespaceRequest) (msgs.DeleteNamespaceResponse, error)
	// ShowNamespace shows information about a namespace.
	ShowNamespace(context.Context, msgs.ShowNamespaceRequest) (msgs.ShowNamespaceResponse, error)
	// UpdateNamespace updates a namespace.
	UpdateNamespace(context.Context, msgs.UpdateNamespaceRequest) (msgs.UpdateNamespaceResponse, error)
}

// UserAPI is the interface that must be implemented to interact with the
// postgres-operator user service.
type UserAPI interface {
	// CreateUser creates a cluster user.
	CreateUser(context.Context, msgs.CreateUserRequest) (msgs.CreateUserResponse, error)
	// DeleteUser deletes a cluster user.
	DeleteUser(context.Context, msgs.DeleteUserRequest) (msgs.DeleteUserResponse, error)
	// ShowUser gets cluster user information.
	ShowUser(context.Context, msgs.ShowUserRequest) (msgs.ShowUserResponse, error)
	// UpdateCluster updates a cluster user.
	UpdateUser(context.Context, msgs.UpdateUserRequest) (msgs.UpdateUserResponse, error)
}

// PGBouncerAPI is the interface that must be implemented to interact with the
// postgres-operator pgBouncer service.
type PGBouncerAPI interface {
	// CreatePgBouncer creates a new pgBouncer.
	CreatePgBouncer(context.Context, msgs.CreatePgbouncerRequest) (msgs.CreatePgbouncerResponse, error)
	// DeletePgBouncer deletes a pgBouncer.
	DeletePgBouncer(context.Context, msgs.DeletePgbouncerRequest) (msgs.DeletePgbouncerResponse, error)
	// UpdatePgBouncer updates a pgBouncer.
	UpdatePgBouncer(context.Context, msgs.UpdatePgBouncerRequest) (msgs.UpdatePgBouncerResponse, error)
	// ShowPgBouncer retrives pgBouncer information.
	ShowPgBouncer(context.Context, msgs.ShowPgBouncerRequest) (msgs.ShowPgBouncerResponse, error)
}

// PGORoleAPI is the interface that must be implemented to interact with the
// postgres-operator pgo role service.
type PGORoleAPI interface {
	// CreatePgoRole creates a pgo role.
	CreatePgoRole(context.Context, msgs.CreatePgoRoleRequest) (msgs.CreatePgoRoleResponse, error)
	// DeletePgoRole deletes a pgo role.
	DeletePgoRole(context.Context, msgs.DeletePgoRoleRequest) (msgs.DeletePgoRoleResponse, error)
	// UpdatePgoRole updates a pgo role.
	UpdatePgoRole(context.Context, msgs.UpdatePgoRoleRequest) (msgs.UpdatePgoRoleResponse, error)
	// ShowPgoRole retrieves pgo role information.
	ShowPgoRole(context.Context, msgs.ShowPgoRoleRequest) (msgs.ShowPgoRoleResponse, error)
}

// PGOUserAPI is the interface that must be implemented to interact with the
// postgres-operator pgo user service.
type PGOUserAPI interface {
	// CreatePgoUser creates a pgo user.
	CreatePgoUser(context.Context, msgs.CreatePgoUserRequest) (msgs.CreatePgoUserResponse, error)
	// DeletePgoUser deletes a pgo user.
	DeletePgoUser(context.Context, msgs.DeletePgoUserRequest) (msgs.DeletePgoUserResponse, error)
	// UpdatePgoUser updates a pgo user.
	UpdatePgoUser(context.Context, msgs.UpdatePgoUserRequest) (msgs.UpdatePgoUserResponse, error)
	// ShowPgoUser retrives pgo user information.
	ShowPgoUser(context.Context, msgs.ShowPgoUserRequest) (msgs.ShowPgoUserResponse, error)
}

// PolicyAPI is the interface that must be implemented to interact with the
// postgres-operator policy service.
type PolicyAPI interface {
	// CreatePolicy creates a policy.
	CreatePolicy(context.Context, msgs.CreatePolicyRequest) (msgs.CreatePolicyResponse, error)
	// DeletePolicy deletes a policy.
	DeletePolicy(context.Context, msgs.DeletePolicyRequest) (msgs.DeletePolicyResponse, error)
	// ApplyPolicy applies a policy.
	ApplyPolicy(context.Context, msgs.ApplyPolicyRequest) (msgs.ApplyPolicyResponse, error)
	// ShowPolicy retrieves policy information.
	ShowPolicy(context.Context, msgs.ShowPolicyRequest) (msgs.ShowPolicyResponse, error)
}

// ScheduleAPI is the interface that must be implemented to interact with the
// postgres-operator schedule service.
type ScheduleAPI interface {
	// CreateSchedule creates a schedule.
	CreateSchedule(context.Context, msgs.CreateScheduleRequest) (msgs.CreateScheduleResponse, error)
	// DeleteSchedule deletes a schedule.
	DeleteSchedule(context.Context, msgs.DeleteScheduleRequest) (msgs.DeleteScheduleResponse, error)
	// ShowSchedule retrieves schedule information.
	ShowSchedule(context.Context, msgs.ShowScheduleRequest) (msgs.ShowScheduleResponse, error)
}

// API provides bindings for the postgres-operator v4 client API.
type API interface {
	ClusterAPI
	BackupAPI
	FailoverAPI
	NamespaceAPI
	UserAPI
	PGBouncerAPI
	PGORoleAPI
	PGOUserAPI
	PolicyAPI
	ScheduleAPI

	// Df
	Df(context.Context, msgs.DfRequest) (msgs.DfResponse, error)

	// ShowPVC
	ShowPVC(context.Context, msgs.ShowPVCRequest) (msgs.ShowPVCResponse, error)

	// ShowStatus
	ShowStatus(ctx context.Context, r msgs.StatusRequest) (msgs.StatusResponse, error)

	// ShowTest
	ShowTest(ctx context.Context, r msgs.ClusterTestRequest) (msgs.ClusterTestResponse, error)

	// Cat performs a 'cat' operation on a file in a cluster.
	Cat(context.Context, msgs.CatRequest) (msgs.CatResponse, error)

	// ShowConfig retrieves configuration information for a namespace.
	ShowConfig(context.Context, msgs.ShowConfigRequest) (msgs.ShowConfigResponse, error)

	// LabelClusters
	LabelCluster(context.Context, msgs.LabelRequest) (msgs.LabelResponse, error)
	// DeleteLabel
	DeleteLabel(context.Context, msgs.DeleteLabelRequest) (msgs.LabelResponse, error)

	// ShowWorkflow retrieves workflow information.
	ShowWorkflow(ctx context.Context, r msgs.ShowWorkflowRequest) (msgs.ShowWorkflowResponse, error)

	// ShowVersion retrieves the server and client version information.
	ShowVersion(context.Context) (msgs.VersionResponse, error)
}
