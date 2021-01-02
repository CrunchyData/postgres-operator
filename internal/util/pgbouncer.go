package util

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
	"fmt"

	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
)

// pgBouncerSecretFormat is the name of the Kubernetes Secret that pgBouncer
// uses that stores configuration and pgbouncer user information, and follows
// the format "<clusterName>-pgbouncer-secret"
const pgBouncerSecretFormat = "%s-pgbouncer-secret"

// pgBouncerUserFileFormat is the format of what the pgBouncer user management
// file looks like, i.e. `"username" "password"``
const pgBouncerUserFileFormat = `"%s" "%s"`

// GeneratePgBouncerSecretName returns the name of the secret that contains
// information around a pgBouncer deployment
func GeneratePgBouncerSecretName(clusterName string) string {
	return fmt.Sprintf(pgBouncerSecretFormat, clusterName)
}

// GeneratePgBouncerUsersFileBytes generates the byte string that is
// used by the pgBouncer secret to authenticate a user into pgBouncer that is
// acting as the pgBouncer "service user" (aka PgBouncerUser).
//
// The format of this file is `"username "hashed-password"`
//
// where "hashed-password" is a MD5 or SCRAM hashed password
//
// This is ultimately moutned by the pgBouncer Pod via the secret
func GeneratePgBouncerUsersFileBytes(hashedPassword string) []byte {
	data := fmt.Sprintf(pgBouncerUserFileFormat, crv1.PGUserPgBouncer, hashedPassword)
	return []byte(data)
}
