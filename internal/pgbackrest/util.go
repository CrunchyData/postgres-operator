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

package pgbackrest

import "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"

// RepoHostEnabled determines whether not a pgBackRest repository host is enabled according to the
// provided PostgresCluster
func RepoHostEnabled(postgresCluster *v1beta1.PostgresCluster) bool {
	return (postgresCluster.Spec.Archive.PGBackRest.RepoHost != nil)
}

// DedicatedRepoHostEnabled determines whether not a pgBackRest dedicated repository host is
// enabled according to the provided PostgresCluster
func DedicatedRepoHostEnabled(postgresCluster *v1beta1.PostgresCluster) bool {
	return (postgresCluster.Spec.Archive.PGBackRest.RepoHost != nil &&
		postgresCluster.Spec.Archive.PGBackRest.RepoHost.Dedicated != nil)
}
