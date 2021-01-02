package userservice

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
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/pgadmin"
	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	errParsingExpiredUsernames = "Error parsing usernames for expired passwords."
	errSystemAccountFormat     = `"%s" is a system account and cannot be modified.`
)

const (
	// sqlAlterRole is SQL that allows for the management of a PostgreSQL user
	// this is really just the clause and effectively does nothing without
	// additional options being supplied to it, but allows for the user to be
	// supplied in. Note that the user must be escape to avoid SQL injections
	sqlAlterRole = `ALTER ROLE %s`
	// sqlCreateRole is SQL that allows a new PostgreSQL user to be created. To
	// safely use this function, the role name and passsword must be escaped to
	// avoid SQL injections, which is handled in the SetPostgreSQLPassword
	// function
	sqlCreateRole = `CREATE ROLE %s PASSWORD %s LOGIN`
	// sqlDisableLoginClause allows a user to disable login to a PostgreSQL
	// account
	sqlDisableLoginClause = `NOLOGIN`
	// sqlDropOwnedBy drops all the objects owned by a PostgreSQL user in a
	// specific **database**, not a cluster. As such, this needs to be executed
	// multiple times when trying to drop a user from a PostgreSQL cluster. The
	// value must be escaped with SQLQuoteIdentifier
	sqlDropOwnedBy = "DROP OWNED BY %s CASCADE"
	// sqlDropRole drops a PostgreSQL user from a PostgreSQL cluster. This must
	// be escaped with SQLQuoteIdentifier
	sqlDropRole = "DROP ROLE %s"
	// sqlEnableLoginClause allows a user to enable login to a PostgreSQL account
	sqlEnableLoginClause = `LOGIN`
	// sqlExpiredPasswordClause is the clause that is used to query a set of
	// PostgreSQL users that have an expired passwords, regardless of if they can
	// log in or not. Note that the value definitely needs to be escaped using
	// SQLQuoteLiteral
	sqlExpiredPasswordClause = `CURRENT_TIMESTAMP + %s::interval >= rolvaliduntil`
	// sqlFindDatabases finds all the database a user can connect to. This is used
	// to ensure we can drop all objects for a particular role. Amazingly, we do
	// not need to do an escaping here
	sqlFindDatabases = `SELECT datname FROM pg_catalog.pg_database WHERE datallowconn;`
	// sqlFindUsers returns information about PostgreSQL users that will be in
	// a format that we need to parse
	sqlFindUsers = `SELECT rolname, rolvaliduntil
FROM pg_catalog.pg_authid
WHERE rolcanlogin`
	// sqlOrderByUsername allows one to order a list from pg_authid by the
	// username
	sqlOrderByUsername = "ORDER BY rolname"
	// sqlPasswordClause is the clause that allows on to set the password. This
	// needs to be escaped to avoid SQL injections using the SQLQuoteLiteral
	// function
	sqlPasswordClause = `PASSWORD %s`
	// sqlSetDatestyle will ensure consistent date formats as we force the
	// datestyle to ISO...which differs from Golang's RFC3339, bu we handle this
	// with sqlTimeFormat.
	// This should be inserted as part of an instructions sent to PostgreSQL, and
	// is only active for that particular query session
	sqlSetDatestyle = `SET datestyle TO 'ISO'`
	// sqlValidUntilClause is a clause that allows one to pass in a valid until
	// timestamp. The value must be escaped to avoid SQL injections, using the
	// util.SQLQuoteLiteral function
	sqlValidUntilClause = `VALID UNTIL %s`
)

const (
	// sqlDelimiter is just a pipe
	sqlDelimiter = "|"
	// sqlTimeFormat is the defauly time format that is used
	sqlTimeFormat = "2006-01-02 15:04:05.999999999Z07"
)

var (
	// sqlCommand is the command that needs to be executed for running SQL
	sqlCommand = []string{"psql", "-A", "-t"}
)

// connInfo ....
type connInfo struct {
	Username string
	Hostip   string
	Port     string
	Database string
	Password string
}

// CreatueUser allows one to create a PostgreSQL user in one of more PostgreSQL
// clusters, and provides the abilit to do the following:
//
// - set a password or have one automatically generated
// - set a valid period where the account/password is activ// - setting password expirations
// - and more
//
// This corresponds to the `pgo update user` command
func CreateUser(request *msgs.CreateUserRequest, pgouser string) msgs.CreateUserResponse {
	response := msgs.CreateUserResponse{
		Results: []msgs.UserResponseDetail{},
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
	}

	log.Debugf("create user called, cluster [%v], selector [%s], all [%t]",
		request.Clusters, request.Selector, request.AllFlag)

	// if the username is one of the PostgreSQL system accounts, return here
	if util.IsPostgreSQLUserSystemAccount(request.Username) {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf(errSystemAccountFormat, request.Username)
		return response
	}

	// try to get a list of clusters. if there is an error, return
	clusterList, err := getClusterList(request.Namespace, request.Clusters, request.Selector, request.AllFlag)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// NOTE: this is a legacy requirement as the uesrname is kept in the name of
	// the secret, which requires RFC 1035 compliance. We could probably update
	// this check as well to be more accurate, and even more the MustCompile
	// statement to being a file-level constant, but for now this is just going
	// to sit here and changed in a planned later commit.
	re := regexp.MustCompile("^[a-z0-9.-]*$")
	if !re.MatchString(request.Username) {
		response.Status.Code = msgs.Error
		response.Status.Msg = "user name is required to contain lowercase letters, numbers, '.' and '-' only."
		return response
	}

	// determine if the user passed in a valid password type
	passwordType, err := msgs.GetPasswordType(request.PasswordType)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// as the password age is uniform throughout the request, we can check for the
	// user supplied value and the defaults here
	validUntil := generateValidUntilDateString(request.PasswordAgeDays)
	sqlValidUntil := fmt.Sprintf(sqlValidUntilClause, util.SQLQuoteLiteral(validUntil))

	// Return an error if any clusters identified for user creation are in standby mode.  Users
	// cannot be created in standby clusters because the database is in read-only mode while the
	// cluster replicates from a remote primary.
	if hasStandby, standbyClusters := apiserver.PGClusterListHasStandby(clusterList); hasStandby {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Request rejected, unable to create users for clusters "+
			"%s: %s.", strings.Join(standbyClusters, ","), apiserver.ErrStandbyNotAllowed.Error())
		return response
	}

	// iterate through each cluster and add the new PostgreSQL role to each pod
	for _, cluster := range clusterList.Items {
		result := msgs.UserResponseDetail{
			ClusterName: cluster.Spec.ClusterName,
			Username:    request.Username,
			ValidUntil:  validUntil,
		}

		log.Debugf("creating user [%s] on cluster [%s]", result.Username, cluster.Spec.ClusterName)

		// first, find the primary Pod
		pod, err := util.GetPrimaryPod(apiserver.Clientset, &cluster)

		// if the primary Pod cannot be found, we're going to continue on for the
		// other clusters, but provide some sort of error message in the response
		if err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		}

		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			response.Status.Code = msgs.Error
			response.Status.Msg = cluster.Spec.ClusterName + msgs.UpgradeError
			return response
		}

		// build up the SQL clause that will be executed.
		sql := sqlCreateRole

		// determine if there is a password expiration set. The SQL clause
		// is already generated and has its injectable input escaped
		if sqlValidUntil != "" {
			sql = fmt.Sprintf("%s %s", sql, sqlValidUntil)
		}

		// Set the password. We want a password to be generated if the user did not
		// set a password
		_, password, hashedPassword, err := generatePassword(result.Username, request.Password, passwordType, true, request.PasswordLength)

		// on the off-chance there is an error, record it and continue
		if err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		}

		result.Password = password

		// attempt to set the password!
		if err := util.SetPostgreSQLPassword(apiserver.Clientset, apiserver.RESTConfig, pod,
			cluster.Spec.Port, result.Username, hashedPassword, sql); err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		}

		// if this user is "managed" by the Operator, add a secret. If there is an
		// error, we can fall through as the next step is appending the results
		if request.ManagedUser {
			if err := util.CreateUserSecret(apiserver.Clientset, cluster.Spec.ClusterName, result.Username,
				result.Password, cluster.Spec.Namespace); err != nil {
				log.Error(err)

				result.Error = true
				result.ErrorMessage = err.Error()

				response.Results = append(response.Results, result)
				continue
			}
		}

		// if a pgAdmin deployment exists, attempt to add the user to it
		if err := updatePgAdmin(&cluster, result.Username, result.Password); err != nil {
			log.Error(err)
			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		}

		// append to the results
		response.Results = append(response.Results, result)
	}

	return response
}

// DeleteUser deletes a PostgreSQL user from clusters
func DeleteUser(request *msgs.DeleteUserRequest, pgouser string) msgs.DeleteUserResponse {
	response := msgs.DeleteUserResponse{
		Results: []msgs.UserResponseDetail{},
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
	}

	log.Debugf("delete user called, cluster [%v], selector [%s], all [%t]",
		request.Clusters, request.Selector, request.AllFlag)

	// if the username is one of the PostgreSQL system accounts, return here
	if util.IsPostgreSQLUserSystemAccount(request.Username) {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf(errSystemAccountFormat, request.Username)
		return response
	}

	// try to get a list of clusters. if there is an error, return
	clusterList, err := getClusterList(request.Namespace, request.Clusters, request.Selector, request.AllFlag)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// iterate through each cluster and try to delete the user!
loop:
	for _, cluster := range clusterList.Items {
		result := msgs.UserResponseDetail{
			ClusterName: cluster.Spec.ClusterName,
			Username:    request.Username,
		}

		log.Debugf("dropping user [%s] from cluster [%s]", result.Username, cluster.Spec.ClusterName)

		// first, find the primary Pod
		pod, err := util.GetPrimaryPod(apiserver.Clientset, &cluster)

		// if the primary Pod cannot be found, we're going to continue on for the
		// other clusters, but provide some sort of error message in the response
		if err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		}

		// first, get a list of all the databases in the cluster. We will need to
		// go through each database and drop any object that the user owns
		output, err := executeSQL(pod, cluster.Spec.Port, sqlFindDatabases, []string{})

		// if there is an error, record it and move on as we cannot actually deleted
		// the user
		if err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		}

		// create the buffer of all the databases, and iterate through them so
		// we can drop individuale objects in them
		databases := bufio.NewScanner(strings.NewReader(output))

		// so we need to parse each of these...and then determine if these are
		// managed accounts and make a call to the secret to get...the password
		for databases.Scan() {
			database := strings.TrimSpace(databases.Text())

			// set up the sql to drop the user object from the database
			sql := fmt.Sprintf(sqlDropOwnedBy, util.SQLQuoteIdentifier(result.Username))

			// and use the one instance where we need to pass in additional argments
			// to the execteSQL function
			// if there is an error, we'll make a note of it here, but we have to
			// continue in the outer loop
			if _, err := executeSQL(pod, cluster.Spec.Port, sql, []string{database}); err != nil {
				log.Error(err)

				result.Error = true
				result.ErrorMessage = err.Error()

				response.Results = append(response.Results, result)
				continue loop
			}
		}

		// and if we survie that unscathed, we can now delete the user, which we
		// have to escape to avoid SQL injections
		sql := fmt.Sprintf(sqlDropRole, util.SQLQuoteIdentifier(result.Username))

		// exceute the SQL. if there is an error, make note and continue
		if _, err := executeSQL(pod, cluster.Spec.Port, sql, []string{}); err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		}

		// alright, final step: try to delete the user secret. if it does not exist,
		// or it fails to delete, we don't care
		deleteUserSecret(cluster, result.Username)

		// remove user from pgAdmin, if enabled
		qr, err := pgadmin.GetPgAdminQueryRunner(apiserver.Clientset, apiserver.RESTConfig, &cluster)
		if err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		} else if qr != nil {
			err = pgadmin.DeleteUser(qr, result.Username)
			if err != nil {
				log.Error(err)

				result.Error = true
				result.ErrorMessage = err.Error()

				response.Results = append(response.Results, result)
				continue
			}
		}

		response.Results = append(response.Results, result)
	}

	return response
}

// ShowUser lets the caller view details about PostgreSQL users across the
// PostgreSQL clusters that are queried. This includes details such as:
//
// - when the password expires
// - if the user is active or not
//
// etc.
func ShowUser(request *msgs.ShowUserRequest) msgs.ShowUserResponse {
	response := msgs.ShowUserResponse{
		Results: []msgs.UserResponseDetail{},
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
	}

	log.Debugf("show user called, cluster [%v], selector [%s], all [%t]",
		request.Clusters, request.Selector, request.AllFlag)

	// first try to get a list of clusters based on the various ways one can get
	// them. If if this returns an error, exit here
	clusterList, err := getClusterList(request.Namespace,
		request.Clusters, request.Selector, request.AllFlag)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// to save some computing power, we can determine if the caller is looking
	// up if passwords are expiring for users. This value is passed in days, so
	// we can get the expiration mark that we are looking for
	expirationInterval := ""

	if request.Expired > 0 {
		// we need to find a set of user passwords that need to be updated
		// set the expiration interval
		expirationInterval = fmt.Sprintf("%d days", request.Expired)
	}

	// iterate through each cluster and look up information about each user
	for _, cluster := range clusterList.Items {
		// first, find the primary Pod
		pod, err := util.GetPrimaryPod(apiserver.Clientset, &cluster)

		// if the primary Pod cannot be found, we're going to continue on for the
		// other clusters, but provide some sort of error message in the response
		if err != nil {
			log.Error(err)

			result := msgs.UserResponseDetail{
				Error:        true,
				ErrorMessage: err.Error(),
			}

			response.Results = append(response.Results, result)
			continue
		}

		// we need to build out some SQL. Start with the base
		sql := fmt.Sprintf("%s; %s", sqlSetDatestyle, sqlFindUsers)

		// determine if we only want to find the users that have expiring passwords
		if expirationInterval != "" {
			sql = fmt.Sprintf("%s AND %s", sql,
				fmt.Sprintf(sqlExpiredPasswordClause, util.SQLQuoteLiteral(expirationInterval)))
		}

		// being a bit cute here, but ordering by the role name
		sql = fmt.Sprintf("%s %s", sql, sqlOrderByUsername)

		// great, now we can perform the user lookup
		output, err := executeSQL(pod, cluster.Spec.Port, sql, []string{})

		// if there is an error, record it and move on to the next cluster
		if err != nil {
			log.Error(err)

			result := msgs.UserResponseDetail{
				Error:        true,
				ErrorMessage: err.Error(),
			}

			response.Results = append(response.Results, result)
			continue
		}

		// get the rows into a buffer and start scanning
		rows := bufio.NewScanner(strings.NewReader(output))

		// the output corresponds to the following pattern:
		// "username|validuntil" which corresponds to:
		// string|sqlTimeFormat
		//
		// so we need to parse each of these...and then determine if these are
		// managed accounts and make a call to the secret to get...the password
		for rows.Scan() {
			row := strings.TrimSpace(rows.Text())

			// split aong the "sqlDelimiter" ("|") to get the 3 values
			values := strings.Split(row, sqlDelimiter)

			// if there are not two values, continue on, as this means this is not
			// the row we are interested in
			if len(values) != 2 {
				continue
			}

			// before continuing, check to see if this is a system account.
			// If it is, check to see that the user requested to view system accounts
			if !request.ShowSystemAccounts && util.IsPostgreSQLUserSystemAccount(values[0]) {
				continue
			}

			// start building a result
			result := msgs.UserResponseDetail{
				ClusterName: cluster.Spec.ClusterName,
				Username:    values[0],
				ValidUntil:  values[1],
			}

			// alright, attempt to get the password if it is "managed"...sigh
			// as we are in a loop, this is costly as there are a lot of network calls
			// so we may want to either add some concurrency or rethink how the
			// managed passwords are stored
			//
			// We ignore any errors...if the password get set, we add it. If not, we
			// don't
			secretName := fmt.Sprintf(util.UserSecretFormat, result.ClusterName, result.Username)
			password, _ := util.GetPasswordFromSecret(apiserver.Clientset, pod.Namespace, secretName)

			if password != "" {
				result.Password = password
			}

			// add the result
			response.Results = append(response.Results, result)
		}
	}

	return response
}

// UpdateUser allows one to update a PostgreSQL user across PostgreSQL clusters,
// and provides the ability to perform inline various updates, including:
//
// - resetting passwords
// - disabling accounts
// - setting password expirations
// - and more
//
// This corresponds to the `pgo update user` command
func UpdateUser(request *msgs.UpdateUserRequest, pgouser string) msgs.UpdateUserResponse {
	response := msgs.UpdateUserResponse{
		Results: []msgs.UserResponseDetail{},
		Status: msgs.Status{
			Code: msgs.Ok,
		},
	}

	log.Debugf("update user called, cluster [%v], selector [%s], all [%t]",
		request.Clusters, request.Selector, request.AllFlag)

	// either a username must be set, or the user is updating the passwords for
	// accounts that are about to expire
	if request.Username == "" && request.Expired == 0 {
		response.Status.Code = msgs.Error
		response.Status.Msg = "Either --username or --expired or must be set."
		return response
	}

	// if this involes updating a specific PostgreSQL account, and it is a system
	// account, return here
	if request.Username != "" && util.IsPostgreSQLUserSystemAccount(request.Username) {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf(errSystemAccountFormat, request.Username)
		return response
	}

	// determine if the user passed in a valid password type
	if _, err := msgs.GetPasswordType(request.PasswordType); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// try to get a list of clusters. if there is an error, return
	clusterList, err := getClusterList(request.Namespace, request.Clusters, request.Selector, request.AllFlag)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// Return an error if any clusters identified for the user updare are in standby mode.  Users
	// cannot be updated in standby clusters because the database is in read-only mode while the
	// cluster replicates from a remote primary
	if hasStandby, standbyClusters := apiserver.PGClusterListHasStandby(clusterList); hasStandby {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Request rejected, unable to update users for clusters "+
			"%s: %s.", strings.Join(standbyClusters, ", "), apiserver.ErrStandbyNotAllowed.Error())
		return response
	}

	for _, cluster := range clusterList.Items {
		var result msgs.UserResponseDetail

		// determine which update user actions needs to be performed
		switch {
		// determine if any passwords expiring in X days should be updated
		// it returns a slice of results, which are then append to the list
		case request.Expired > 0:
			results := rotateExpiredPasswords(request, &cluster)
			response.Results = append(response.Results, results...)
		// otherwise, perform a regular "update user" request which covers all the
		// other "regular" cases. It returns a result, which is append to the list
		default:
			result = updateUser(request, &cluster)
			response.Results = append(response.Results, result)
		}
	}

	return response
}

// deleteUserSecret deletes the user secret that stores information like the
// user's password.
// For the purposes of this module, we don't care if this fails. We'll log the
// error in here, but do nothing with it
func deleteUserSecret(cluster crv1.Pgcluster, username string) {
	secretName := fmt.Sprintf(util.UserSecretFormat, cluster.Spec.ClusterName, username)

	err := apiserver.Clientset.CoreV1().Secrets(cluster.Spec.Namespace).Delete(secretName, nil)

	if err != nil {
		log.Error(err)
	}
}

// executeSQL executes SQL on the primary PostgreSQL Pod. This occurs using the
// Kubernetes exec function, which allows us to perform the request over
// a PostgreSQL connection that's authenticated with peer authentication
func executeSQL(pod *v1.Pod, port, sql string, extraCommandArgs []string) (string, error) {
	command := sqlCommand

	// add the port
	command = append(command, "-p", port)

	// add any extra arguments
	command = append(command, extraCommandArgs...)

	// execute into the primary pod to run the query
	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(apiserver.RESTConfig,
		apiserver.Clientset, command,
		"database", pod.Name, pod.ObjectMeta.Namespace, strings.NewReader(sql))

	// if there is an error executing the command, which includes the stderr,
	// return the error
	if err != nil {
		return "", err
	} else if stderr != "" {
		return "", fmt.Errorf(stderr)
	}

	return stdout, nil
}

// generatePassword will return a password that is either set by the user or
// generated based upon a length that is passed in. Additionally, it will return
// the password in a hashed format so it can be saved safely by the PostgreSQL
// server. There is also a boolean parameter that indicates whether or not a
// password was updated: it's set to true if it is
//
// It also includes a boolean parameter to determine whether or not a password
// should be generated, which is helpful in the "update user" workflow.
//
// If both parameters return empty, then this means that no action should be
// taken on updating the password.
//
// A set password takes precedence over a password being generated. if
// "password" is empty, then a password will be generated. If both are set,
// then "password" is used.
//
// Finally, one can specify the "password type" to be generated, which right now
// is either one of MD5 of SCRAM, the two PostgreSQL password authentication
// methods. This will return a hash / verifier that is stored in PostgreSQL
func generatePassword(username, password string, passwordType pgpassword.PasswordType,
	generatePassword bool, generatedPasswordLength int) (bool, string, string, error) {
	// first, an early exit: nothing is updated
	if password == "" && !generatePassword {
		return false, "", "", nil
	}

	// give precedence to the user customized password
	if password == "" && generatePassword {
		// Determine if the user passed in a password length, otherwise us the
		// default
		passwordLength := generatedPasswordLength

		if passwordLength == 0 {
			passwordLength = util.GeneratedPasswordLength(apiserver.Pgo.Cluster.PasswordLength)
		}

		// generate the password
		generatedPassword, err := util.GeneratePassword(passwordLength)

		// if there is an error, return
		if err != nil {
			return false, "", "", err
		}

		password = generatedPassword
	}

	// finally, hash the password
	postgresPassword, err := pgpassword.NewPostgresPassword(passwordType, username, password)

	if err != nil {
		return false, "", "", err
	}

	hashedPassword, err := postgresPassword.Build()

	if err != nil {
		return false, "", "", err
	}

	// return!
	return true, password, hashedPassword, nil
}

// generateValidUntilDateString returns a RFC3339 string that is computed by
// adding the current time on the Operator server with the integer number of
// days that are passed in. If the total number of days passed in is <= 0, then
// it also checks the server configured value.
//
// If it's still less than 0, then the password is considered to be always
// valid and a value of "infinity" is returned
//
// otherwise, it computes the password expiration from the total number of days
func generateValidUntilDateString(validUntilDays int) string {
	// if validUntilDays is zero (or less than zero), attempt to set the value
	// supplied by the server. If it's still zero, then the user can create a
	// password without expiration
	if validUntilDays <= 0 {
		validUntilDays = util.GeneratedPasswordValidUntilDays(apiserver.Pgo.Cluster.PasswordAgeDays)

		if validUntilDays <= 0 {
			return util.SQLValidUntilAlways
		}
	}

	// okay, this is slightly annoying. So to get the total duration in days, we
	// need to set up validUntilDays * # hours in the time.Duration function, and then
	// multiple it by the value for hours
	duration := time.Duration(validUntilDays*24) * time.Hour

	// ok, set the validUntil time and return the correct format
	validUntil := time.Now().Add(duration)

	return validUntil.Format(time.RFC3339)
}

// getClusterList tries to return a list of clusters based on either having an
// argument list of cluster names, a Kubernetes selector, or set to "all"
func getClusterList(namespace string, clusterNames []string, selector string, all bool) (crv1.PgclusterList, error) {
	clusterList := crv1.PgclusterList{}

	// see if there are any in one of the three parametes used to return everything
	if len(clusterNames) == 0 && selector == "" && !all {
		err := fmt.Errorf("either a list of cluster names, a selector, or the all flag needs to be supplied for this comment")
		return clusterList, err
	}

	// if the all flag is set, let's return all the clusters here and return
	if all {
		// return the value of cluster list or that of the error here
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).List(metav1.ListOptions{})
		if err == nil {
			clusterList = *cl
		}
		return clusterList, err
	}

	// try to build the cluster list based on either the selector or the list
	// of arguments...or both. First, start with the selector
	if selector != "" {
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).List(metav1.ListOptions{LabelSelector: selector})

		// if there is an error, return here with an empty cluster list
		if err != nil {
			return crv1.PgclusterList{}, err
		}
		clusterList = *cl
	}

	// now try to get clusters based specific cluster names
	for _, clusterName := range clusterNames {
		cluster, err := apiserver.Clientset.CrunchydataV1().Pgclusters(namespace).Get(clusterName, metav1.GetOptions{})

		// if there is an error, capture it here and return here with an empty list
		if err != nil {
			return crv1.PgclusterList{}, err
		}

		// if successful, append to the cluster list
		clusterList.Items = append(clusterList.Items, *cluster)
	}

	log.Debugf("clusters founds: [%d]", len(clusterList.Items))

	// if after all this, there are no clusters found, return an error
	if len(clusterList.Items) == 0 {
		err := fmt.Errorf("no clusters found")
		return clusterList, err
	}

	// all set! return the cluster list with error
	return clusterList, nil
}

// rotateExpiredPasswords finds all of the PostgreSQL users in a cluster that can
// login but have their passwords expired or are expring in X days and rotates
// the passwords. This is accomplish in two steps:
//
// 1. Finding all of the non-system accounts and checking for expirations
// 2. Generating a new password and updating each account
func rotateExpiredPasswords(request *msgs.UpdateUserRequest, cluster *crv1.Pgcluster) []msgs.UserResponseDetail {
	results := []msgs.UserResponseDetail{}

	log.Debugf("rotate expired passwords on cluster [%s]", cluster.Spec.ClusterName)

	// first, find the primary Pod. If we can't do that, no rense in continuing
	pod, err := util.GetPrimaryPod(apiserver.Clientset, cluster)

	if err != nil {
		result := msgs.UserResponseDetail{
			ClusterName:  cluster.Spec.ClusterName,
			Error:        true,
			ErrorMessage: err.Error(),
		}
		results = append(results, result)
		return results
	}

	// start building the sql, which is the clause for finding users that can
	// login
	sql := sqlFindUsers

	// we need to find a set of user passwords that need to be updated
	// set the expiration interval
	expirationInterval := fmt.Sprintf("%d days", request.Expired)
	// and then immediately put it into SQL, with appropriate SQL injection
	// escaping
	sql = fmt.Sprintf("%s AND %s", sql,
		fmt.Sprintf(sqlExpiredPasswordClause, util.SQLQuoteLiteral(expirationInterval)))

	// alright, time to find if there are any expired accounts. If this errors,
	// then we will abort here
	output, err := executeSQL(pod, cluster.Spec.Port, sql, []string{})

	if err != nil {
		result := msgs.UserResponseDetail{
			ClusterName:  cluster.Spec.ClusterName,
			Error:        true,
			ErrorMessage: err.Error(),
		}
		results = append(results, result)
		return results
	}

	// put the list of usernames into a buffer that we will iterate through
	usernames := bufio.NewScanner(strings.NewReader(output))

	// before we start the loop, prepare for the update to the expiration time.
	// We do need to update the expiration time, otherwise these passwords will
	// still expire :)
	//
	// check to see if the user passedin the "never expire" flag, otherwise try
	// to update either from the user generated value or the default value (which
	// may very well be to not expire)
	validUntil := ""

	switch {
	case request.PasswordValidAlways:
		validUntil = util.SQLValidUntilAlways
	default:
		validUntil = generateValidUntilDateString(request.PasswordAgeDays)
	}

	// iterate through each user name, which will then be used to go through and
	// update the password for each user
	// Note that the query has the format "username|sqlTimeFormat" so we need
	// to parse that below
	for usernames.Scan() {
		// get the values out of the query
		values := strings.Split(strings.TrimSpace(usernames.Text()), "|")

		// if there is not at least one value, just abort here
		if len(values) < 1 {
			result := msgs.UserResponseDetail{
				Error:        true,
				ErrorMessage: errParsingExpiredUsernames,
			}
			results = append(results, result)
			continue
		}

		// otherwise, we can safely set the username
		username := values[0]

		// start building a result. The Username call strips off the newlines and
		// other garbage and returns the actual username
		result := msgs.UserResponseDetail{
			ClusterName: cluster.Spec.ClusterName,
			Username:    username,
			ValidUntil:  validUntil,
		}

		// start building the SQL
		sql := fmt.Sprintf(sqlAlterRole, util.SQLQuoteIdentifier(result.Username))

		// get the password type. the error is already evaluated in a called
		// function
		passwordType, _ := msgs.GetPasswordType(request.PasswordType)

		// generate a new password. Check to see if the user passed in a particular
		// length of the password, or passed in a password to rotate (though that
		// is not advised...). This forced the password to change
		_, password, hashedPassword, err := generatePassword(result.Username, request.Password, passwordType, true, request.PasswordLength)

		// on the off-chance there's an error in generating the password, record it
		// and continue
		if err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			results = append(results, result)
			continue
		}

		result.Password = password
		sql = fmt.Sprintf("%s %s", sql,
			fmt.Sprintf(sqlPasswordClause, util.SQLQuoteLiteral(hashedPassword)))

		// build the "valid until" value into the SQL string
		sql = fmt.Sprintf("%s %s", sql,
			fmt.Sprintf(sqlValidUntilClause, util.SQLQuoteLiteral(result.ValidUntil)))

		// and this is enough to execute
		// if there is an error, record it here. The next step is to continue
		// iterating through the loop, and we will continue to do so
		if _, err := executeSQL(pod, cluster.Spec.Port, sql, []string{}); err != nil {
			result.Error = true
			result.ErrorMessage = err.Error()
		}

		results = append(results, result)
	}

	return results
}

// updatePgAdmin will attempt to synchronize information in a pgAdmin
// deployment, should one exist. Basically, it adds or updates the credentials
// of a user should there be a pgadmin deploymen associated with this PostgreSQL
// cluster. Returns an error if anything goes wrong
func updatePgAdmin(cluster *crv1.Pgcluster, username, password string) error {
	// Sync user to pgAdmin, if enabled
	qr, err := pgadmin.GetPgAdminQueryRunner(apiserver.Clientset, apiserver.RESTConfig, cluster)

	// if there is an error, return as such
	if err != nil {
		return err
	}

	// likewise, if there is no pgAdmin associated this cluster, return no error
	if qr == nil {
		return nil
	}

	// proceed onward
	// Get service details and prep connection metadata
	service, err := apiserver.Clientset.CoreV1().Services(cluster.Namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// set up the server entry data
	dbService := pgadmin.ServerEntryFromPgService(service, cluster.Name)
	dbService.Password = password

	// attempt to set the username/password for this user in the pgadmin
	// deployment
	if err := pgadmin.SetLoginPassword(qr, username, password); err != nil {
		return err
	}

	// if the service name for the database is present, also set the cluster
	// if it's not set, early exit
	if dbService.Name == "" {
		return nil
	}

	if err := pgadmin.SetClusterConnection(qr, username, dbService); err != nil {
		return err
	}

	return nil
}

// updateUser, though perhaps poorly named in context, performs the standard
// "ALTER ROLE" type functionality on a user, which is just updating a single
// user account on a single PostgreSQL cluster. This is in contrast with some
// of the bulk updates that can occur with updating a user (e.g. resetting
// expired passwords), which is why it's broken out into its own function
func updateUser(request *msgs.UpdateUserRequest, cluster *crv1.Pgcluster) msgs.UserResponseDetail {
	result := msgs.UserResponseDetail{
		ClusterName: cluster.Spec.ClusterName,
		Username:    request.Username,
	}

	log.Debugf("updating user [%s] on cluster [%s]", result.Username, cluster.Spec.ClusterName)

	// first, find the primary Pod
	pod, err := util.GetPrimaryPod(apiserver.Clientset, cluster)

	// if the primary Pod cannot be found, we're going to continue on for the
	// other clusters, but provide some sort of error message in the response
	if err != nil {
		log.Error(err)

		result.Error = true
		result.ErrorMessage = err.Error()

		return result
	}

	// alright, so we can start building up some SQL now, as the other commands
	// here can all occur within ALTER ROLE!
	//
	// We first build it up with the username, being careful to escape the
	// identifier to avoid SQL injections :)
	sql := fmt.Sprintf(sqlAlterRole, util.SQLQuoteIdentifier(request.Username))

	// Though we do have an awesome function for setting a PostgreSQL password
	// (SetPostgreSQLPassword) the problem is we are going to be adding too much
	// to the string here, and we don't always know if the password is being
	// updated, which is one of the requirements of the function. So we will
	// perform any query execution here in this module

	// Speaking of passwords...let's first determine if the user updated their
	// password. See generatePassword for how precedence is given for password
	// updates
	passwordType, _ := msgs.GetPasswordType(request.PasswordType)
	isChanged, password, hashedPassword, err := generatePassword(result.Username,
		request.Password, passwordType, request.RotatePassword, request.PasswordLength)

	// in the off-chance there is an error generating the password, record it
	// and return
	if err != nil {
		log.Error(err)

		result.Error = true
		result.ErrorMessage = err.Error()

		return result
	}

	if isChanged {
		result.Password = password
		sql = fmt.Sprintf("%s %s", sql,
			fmt.Sprintf(sqlPasswordClause, util.SQLQuoteLiteral(hashedPassword)))

		// Sync user to pgAdmin, if enabled
		if err := updatePgAdmin(cluster, result.Username, result.Password); err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			return result
		}
	}

	// now, check to see if the request wants to expire the user's password
	// this will leverage the PostgreSQL ability to set a date as "-infinity"
	// so that the password is 100% expired
	//
	// Expiring the user also takes precedence over trying to move the update
	// password timeline, which we check for next
	//
	// Next we check to ensure the user wants to explicitly un-expire a
	// password, and/or ensure that the expiration time is unlimited. This takes
	// precednece over setting an explicitly expiration period, which we check
	// for last
	switch {
	case request.ExpireUser:
		// append the VALID UNTIL special clause here for explicitly disallowing
		// the user of a password
		result.ValidUntil = util.SQLValidUntilNever
	case request.PasswordValidAlways:
		// append the VALID UNTIL special clause here for explicitly always
		// allowing a password
		result.ValidUntil = util.SQLValidUntilAlways
	case request.PasswordAgeDays > 0:
		// Move the user's password expiration date
		result.ValidUntil = generateValidUntilDateString(request.PasswordAgeDays)
	}

	// if ValidUntil is updated, continue to build the SQL
	if result.ValidUntil != "" {
		sql = fmt.Sprintf("%s %s", sql,
			fmt.Sprintf(sqlValidUntilClause, util.SQLQuoteLiteral(result.ValidUntil)))
	}

	// Now, determine if we want to enable or disable the login. Enable takes
	// precedence over disable
	// None of these have SQL injectionsas they are fixed constants
	switch request.LoginState {
	case msgs.UpdateUserLoginEnable:
		sql = fmt.Sprintf("%s %s", sql, sqlEnableLoginClause)
	case msgs.UpdateUserLoginDisable:
		sql = fmt.Sprintf("%s %s", sql, sqlDisableLoginClause)
	}

	// execute the SQL! if there is an error, return the results
	if _, err := executeSQL(pod, cluster.Spec.Port, sql, []string{}); err != nil {
		log.Error(err)

		result.Error = true
		result.ErrorMessage = err.Error()

		// even though we return in the next line, having an explicit return here
		// in case we add any additional logic beyond this point
		return result
	}

	// If the password did change, it is not updated in the database. If the user
	// has a "managed" account (i.e. there is a secret for this user account"),
	// we can now updated the value of that password in the secret
	if isChanged {
		secretName := fmt.Sprintf(util.UserSecretFormat, cluster.Spec.ClusterName, result.Username)

		// only call update user secret if the secret exists
		if _, err := apiserver.Clientset.CoreV1().Secrets(cluster.Namespace).Get(secretName, metav1.GetOptions{}); err == nil {
			// if we cannot update the user secret, only warn that we cannot do so
			if err := util.UpdateUserSecret(apiserver.Clientset, cluster.Spec.ClusterName,
				result.Username, result.Password, cluster.Namespace); err != nil {
				log.Warn(err)
			}
		}
	}

	return result
}
