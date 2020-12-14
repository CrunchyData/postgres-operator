package cluster

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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
	"strings"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// disable the a Postgres user from logging in. This is safe from SQL
	// injection as the string that is being interpolated is escaped
	//
	// This had the "PASSWORD NULL" feature, but this is only found in
	// PostgreSQL 11+, and given we don't want to check for the PG version before
	// running the command, we will not use it
	sqlDisableLogin = `ALTER ROLE %s NOLOGIN;`

	// sqlEnableLogin is the SQL to update the password
	// NOTE: this is safe from SQL injection as we explicitly add the inerpolated
	// string as a MD5 hash and we are using the username.
	// However, the escaping is handled in the util.SetPostgreSQLPassword function
	sqlEnableLogin = `ALTER ROLE %s PASSWORD %s LOGIN;`
)

// disablePostgresLogin disables the ability for a PostgreSQL user to log in
func disablePostgresLogin(clientset kubernetes.Interface, restconfig *rest.Config, cluster *crv1.Pgcluster, username string) error {
	log.Debugf("disable user %q on cluster %q", username, cluster.Name)
	// disable the pgbouncer user in the PostgreSQL cluster.
	// first, get the primary pod. If we cannot do this, let's consider it an
	// error and abort
	pod, err := util.GetPrimaryPod(clientset, cluster)
	if err != nil {
		return err
	}

	// This is safe from SQL injection as we are escaping the username
	sql := strings.NewReader(fmt.Sprintf(sqlDisableLogin, util.SQLQuoteIdentifier(username)))
	cmd := []string{"psql", "-p", cluster.Spec.Port}

	// exec into the pod to run the query
	_, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		cmd, "database", pod.Name, pod.ObjectMeta.Namespace, sql)
	// if there is an error, log the error from the stderr and return the error
	if err != nil {
		return fmt.Errorf(stderr)
	}

	return nil
}

// generatePassword generates a password that is used for the PostgreSQL user
// system accounts. This goes off of the configured value for password length
func generatePassword() (string, error) {
	// first, get the length of what the password should be
	generatedPasswordLength := util.GeneratedPasswordLength(operator.Pgo.Cluster.PasswordLength)
	// from there, the password can be generated!
	return util.GeneratePassword(generatedPasswordLength)
}

// makePostgresPassword creates the expected hash for a password type for a
// PostgreSQL password
func makePostgresPassword(passwordType pgpassword.PasswordType, username, password string) string {
	// get the PostgreSQL password generate based on the password type
	// as all of these values are valid, this not not error
	postgresPassword, _ := pgpassword.NewPostgresPassword(passwordType, username, password)

	// create the PostgreSQL style hashed password and return
	hashedPassword, _ := postgresPassword.Build()

	return hashedPassword
}

// setPostgreSQLPassword updates the password of a user in a PostgreSQL
// cluster by executing into the Pod provided (i.e. a primary) and changing it
func setPostgreSQLPassword(clientset kubernetes.Interface, restconfig *rest.Config, pod *v1.Pod, port,
	username, password string) error {
	log.Debugf("set %q password in PostgreSQL", username)

	// we use the PostgreSQL "md5" hashing mechanism here to pre-hash the
	// password. This is semi-hard coded but is now prepped for SCRAM as a
	// password type can be passed in. Almost to SCRAM!
	passwordHash := makePostgresPassword(pgpassword.MD5, username, password)

	if err := util.SetPostgreSQLPassword(clientset, restconfig, pod,
		port, username, passwordHash, sqlEnableLogin); err != nil {
		log.Error(err)
		return err
	}

	// and that's all!
	return nil
}
