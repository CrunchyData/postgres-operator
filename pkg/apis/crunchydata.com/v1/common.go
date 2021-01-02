package v1

/*
Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

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
	// PGUserAdmin is a DEPRECATED user and is only included to filter this out
	// as a system user in older systems
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

// PGFSGroup stores the UID of the PostgreSQL user that runs the PostgreSQL
// process, which is 26. This also sets up for future work, as the
// PodSecurityContext structure takes a *int64 for its FSGroup
//
// This has to be a "var" as Kubernetes requires for this to be a pointer
var PGFSGroup int64 = 26

// PGUserSystemAccounts maintains an easy-to-access list of what the systems
// accounts are, which may affect how information is returned, etc.
var PGUserSystemAccounts = map[string]struct{}{
	PGUserAdmin:       {},
	PGUserMonitor:     {},
	PGUserPgBouncer:   {},
	PGUserReplication: {},
	PGUserSuperuser:   {},
}

// PgStorageSpec ...
// swagger:ignore
type PgStorageSpec struct {
	Name               string `json:"name"`
	StorageClass       string `json:"storageclass"`
	AccessMode         string `json:"accessmode"`
	Size               string `json:"size"`
	StorageType        string `json:"storagetype"`
	SupplementalGroups string `json:"supplementalgroups"`
	MatchLabels        string `json:"matchLabels"`
}

// GetSupplementalGroups converts the comma-separated list of SupplementalGroups
// into a slice of int64 IDs. If it errors, it returns an empty slice and logs
// a warning
func (s PgStorageSpec) GetSupplementalGroups() []int64 {
	supplementalGroups := []int64{}

	// split the supplemental group list
	results := strings.Split(s.SupplementalGroups, ",")

	// iterate through the results and try to append to the supplementalGroups
	// array
	for _, result := range results {
		result = strings.TrimSpace(result)

		// if the result is the empty string (likely because there are no
		// supplemental groups), continue on
		if result == "" {
			continue
		}

		supplementalGroup, err := strconv.Atoi(result)
		// if there is an error, only warn about it and continue through the loop
		if err != nil {
			log.Warnf("malformed storage supplemental group: %v", err)
			continue
		}

		// convert the int to an int64 to match the Kubernetes spec, and append to
		// the supplementalGroups slice
		supplementalGroups = append(supplementalGroups, int64(supplementalGroup))
	}

	return supplementalGroups
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
