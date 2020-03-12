package v1

/*
Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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

import ()

// RootSecretSuffix ...
const RootSecretSuffix = "-postgres-secret"

// UserSecretSuffix ...
const UserSecretSuffix = "-secret"

// PrimarySecretSuffix ...
const PrimarySecretSuffix = "-primaryuser-secret"

// CollectSecretSuffix ...
const CollectSecretSuffix = "-collect-secret"

// StorageExisting ...
const StorageExisting = "existing"

// StorageCreate ...
const StorageCreate = "create"

// StorageEmptydir ...
const StorageEmptydir = "emptydir"

// StorageDynamic ...
const StorageDynamic = "dynamic"

// the following are standard PostgreSQL user service accounts that are created
// as part of managed the PostgreSQL cluster environment via the Operator
const (
	// PGUserAdmin is a special user that can perform administrative actions
	// without being a superuser itself
	PGUserAdmin = "crunchyadm"
	// PGUserMonitor is the monitoring user that can access metric data
	PGUserMonitor = "ccp_monitoring"
	// PGUserPgBouncer is the user that's used for managing pgBouncer, which a
	// user can use to access pgBouncer stats, etc.
	PGUserPgBouncer = "pgbouncer"
	// PGUserReplication is the user that's used for replication, which has
	// elevated privileges
	PGUserReplication = "primaryuser"
	// PGUserSuperuser is the superuser account that can do anything
	PGUserSuperuser = "postgres"
)

// PGUserSystemAccounts maintains an easy-to-access list of what the systems
// accounts are, which may affect how information is returned, etc.
var PGUserSystemAccounts = map[string]struct{}{
	PGUserAdmin:       struct{}{},
	PGUserMonitor:     struct{}{},
	PGUserPgBouncer:   struct{}{},
	PGUserReplication: struct{}{},
	PGUserSuperuser:   struct{}{},
}

// PgStorageSpec ...
// swagger:ignore
type PgStorageSpec struct {
	Name               string `json:"name"`
	StorageClass       string `json:"storageclass"`
	AccessMode         string `json:"accessmode"`
	Size               string `json:"size"`
	StorageType        string `json:"storagetype"`
	Fsgroup            string `json:"fsgroup"`
	SupplementalGroups string `json:"supplementalgroups"`
	MatchLabels        string `json:"matchLabels"`
}

// PgContainerResource ...
// swagger:ignore
type PgContainerResources struct {
	RequestsMemory string `json:"requestsmemory"`
	RequestsCPU    string `json:"requestscpu"`
	LimitsMemory   string `json:"limitsmemory"`
	LimitsCPU      string `json:"limitscpu"`
}

// CompletedStatus -
const CompletedStatus = "completed"

// InProgressStatus -
const InProgressStatus = "in progress"

// SubmittedStatus -
const SubmittedStatus = "submitted"

// JobCompletedStatus ....
const JobCompletedStatus = "job completed"

// JobSubmittedStatus ....
const JobSubmittedStatus = "job submitted"

// JobErrorStatus ....
const JobErrorStatus = "job error"
