/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package naming

const (
	labelPrefix = "postgres-operator.crunchydata.com/"

	LabelCluster     = labelPrefix + "cluster"
	LabelInstance    = labelPrefix + "instance"
	LabelInstanceSet = labelPrefix + "instance-set"
	LabelPatroni     = labelPrefix + "patroni"
	LabelRole        = labelPrefix + "role"

	// LabelPGBackRest is used to indicate that a resource is for pgBackRest
	LabelPGBackRest = labelPrefix + "pgbackrest"

	// LabelPGBackRestRepo is used to indicate that a Deployment or Pod is for a pgBackRest
	// repository
	LabelPGBackRestRepo = labelPrefix + "pgbackrest-repo"

	// LabelUserSecret is used to identify the secret containing the Postgres
	// user connection information
	LabelUserSecret = labelPrefix + "pguser"

	RolePrimary = "primary"
	RoleReplica = "replica"

	// Patroni sets this LabelRole value on Pods that are following the leader.
	RolePatroniReplica = "replica"
)
