package userservice

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

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	errSystemAccountFormat = `"%s" is a system account and cannot be used`
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
	// sqlEnableLoginClause allows a user to enable login to a PostgreSQL account
	sqlEnableLoginClause = `LOGIN`
	// sqlExpiredPasswordClause is the clause that is used to query a set of
	// PostgreSQL users that have an expired passwords, regardless of if they can
	// log in or not. Note that the value definitely needs to be escaped using
	// SQLQuoteLiteral
	sqlExpiredPasswordClause = `CURRENT_TIMESTAMP + %s::interval >= rolvaliduntil`
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

// userSecretFormat follows the pattern of how the user information is stored,
// which is "<clusteRName>-<userName>-secret"
const userSecretFormat = "%s-%s" + crv1.UserSecretSuffix

// connInfo ....
type connInfo struct {
	Username string
	Hostip   string
	Port     string
	Database string
	Password string
}

// getPostgresUserInfo...
// TODO: delete this function as it's useless
func getPostgresUserInfo(namespace, clusterName string) (connInfo, error) {
	var err error
	info := connInfo{}

	service, found, err := kubeapi.GetService(apiserver.Clientset, clusterName, namespace)
	if err != nil {
		return info, err
	}
	if !found {
		return info, errors.New("primary service not found for " + clusterName)
	}

	//get the secrets for this cluster
	selector := "!" + config.LABEL_PGO_BACKREST_REPO + "," + config.LABEL_PG_CLUSTER + "=" + clusterName
	secrets, err := kubeapi.GetSecrets(apiserver.Clientset, selector, namespace)
	if err != nil {
		return info, err
	}

	//get the postgres user secret info
	var username, password, database, hostip string
	for _, s := range secrets.Items {
		username = string(s.Data[config.LABEL_USERNAME][:])
		password = string(s.Data[config.LABEL_PASSWORD][:])
		database = "postgres"
		hostip = service.Spec.ClusterIP
		if username == "postgres" {
			log.Debug("got postgres user secrets")
			break
		}
	}

	//query the database for users that have expired
	strPort := fmt.Sprint(service.Spec.Ports[0].Port)
	info.Username = username
	info.Password = password
	info.Database = database
	info.Hostip = hostip
	info.Port = strPort

	return info, err
}

// deleteUser ...
func deleteUser(namespace, clusterName string, info connInfo, user string, managed bool) error {
	var conn *sql.DB
	var err error

	conn, err = sql.Open("postgres", "sslmode=disable user="+info.Username+" host="+info.Hostip+" port="+info.Port+" dbname="+info.Database+" password="+info.Password)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	var rows *sql.Rows

	querystr := "drop owned by \"" + user + "\" cascade"
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	querystr = "drop user if exists \"" + user + "\""
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()

	if managed {
		//delete current secret
		secretName := clusterName + "-" + user + "-secret"
		err := kubeapi.DeleteSecret(apiserver.Clientset, secretName, namespace)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}
	return err

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
	if util.CheckPostgreSQLUserSystemAccount(request.Username) {
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

	// as the password age is uniform throughout the request, we can check for the
	// user supplied value and the defaults here
	validUntil := generateValidUntilDateString(request.PasswordAgeDays)
	sqlValidUntil := fmt.Sprintf(sqlValidUntilClause, util.SQLQuoteLiteral(validUntil))

	// iterate through each cluster and add the new PostgreSQL role to each pod
	for _, cluster := range clusterList.Items {
		result := msgs.UserResponseDetail{
			ClusterName: cluster.Name,
			Username:    request.Username,
			ValidUntil:  validUntil,
		}

		log.Debugf("creating user [%s] on cluster [%s]", result.Username, cluster.Name)

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

		// build up the SQL clause that will be executed.
		sql := sqlCreateRole

		// determine if there is a password expiration set. The SQL clause
		// is already generated and has its injectable input escaped
		if sqlValidUntil != "" {
			sql = fmt.Sprintf("%s %s", sql, sqlValidUntil)
		}

		// Set the password. We want a password to be generated if the user did not
		// set a password
		_, password, hashedPassword := generatePassword(result.Username, request.Password, true, request.PasswordLength)
		result.Password = password

		// attempt to set the password!
		if err := util.SetPostgreSQLPassword(apiserver.Clientset, apiserver.RESTConfig, pod, result.Username, hashedPassword, sql); err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		}

		// if this user is "managed" by the Operator, add a secret. If there is an
		// error, we can fall through as the next step is appending the results
		if request.ManagedUser {
			if err := util.CreateUserSecret(apiserver.Clientset, cluster.Name, result.Username,
				result.Password, cluster.Spec.Namespace); err != nil {
				log.Error(err)

				result.Error = true
				result.ErrorMessage = err.Error()

				response.Results = append(response.Results, result)
				continue
			}
		}

		// append to the results
		response.Results = append(response.Results, result)
	}

	return response
}

// DeleteUser ...
func DeleteUser(request *msgs.DeleteUserRequest, pgouser string) msgs.DeleteUserResponse {
	var err error

	response := msgs.DeleteUserResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	log.Debugf("DeleteUser called name=%s", request.Username)
	if request.Username == "" {
		response.Status.Code = msgs.Error
		response.Status.Msg = "--username is required"
		return response
	}

	// try to get a list of clusters. if there is an error, return
	clusterList, err := getClusterList(request.Namespace, request.Clusters, request.Selector, request.AllFlag)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	var managed bool
	var msg, clusterName string

	for _, cluster := range clusterList.Items {
		clusterName = cluster.Spec.Name
		info, err := getPostgresUserInfo(request.Namespace, clusterName)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		secretName := clusterName + "-" + request.Username + "-secret"

		managed, err = isManaged(secretName, request.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		log.Debugf("DeleteUser %s managed %t", request.Username, managed)

		err = deleteUser(request.Namespace, clusterName, info, request.Username, managed)
		if err != nil {
			log.Error(err)
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		msg = request.Username + " on " + clusterName + " removed managed=" + strconv.FormatBool(managed)
		log.Debug(msg)
		response.Results = append(response.Results, msg)

		//publish event for delete user
		topics := make([]string, 1)
		topics[0] = events.EventTopicUser

		f := events.EventDeleteUserFormat{
			EventHeader: events.EventHeader{
				Namespace: request.Namespace,
				Username:  pgouser,
				Topic:     topics,
				Timestamp: time.Now(),
				EventType: events.EventDeleteUser,
			},
			Clustername:      clusterName,
			PostgresUsername: request.Username,
			Managed:          managed,
		}

		err = events.Publish(f)
		if err != nil {
			log.Error(err)
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

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
	validUntil := ""

	if request.Expired > 0 {
		validUntil = generateValidUntilDateString(request.Expired)
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
		if validUntil != "" {
			sql = fmt.Sprint("%s AND %s", sql,
				fmt.Sprintf(sqlExpiredPasswordClause, util.SQLQuoteLiteral(validUntil)))
		}

		// being a bit cute here, but ordering by the role name
		sql = fmt.Sprintf("%s %s", sql, sqlOrderByUsername)

		// great, now we can perform the user lookup
		output, err := executeSQL(pod, sql)

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
			if !request.ShowSystemAccounts && util.CheckPostgreSQLUserSystemAccount(values[0]) {
				continue
			}

			// start building a result
			result := msgs.UserResponseDetail{
				ClusterName: cluster.Name,
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
			secretName := fmt.Sprintf(userSecretFormat, result.ClusterName, result.Username)
			_, password, _ := util.GetPasswordFromSecret(apiserver.Clientset, pod.Namespace, secretName)

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
	// account, return ere
	if request.Username != "" && util.CheckPostgreSQLUserSystemAccount(request.Username) {
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

func isManaged(secretName, ns string) (bool, error) {
	_, found, err := kubeapi.GetSecret(apiserver.Clientset, secretName, ns)
	if !found {
		return false, nil
	}
	if found {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	return true, err

}

func deleteUserSecret(clientset *kubernetes.Clientset, clustername, username, namespace string) error {
	//delete current secret
	secretName := clustername + "-" + username + "-secret"

	err := kubeapi.DeleteSecret(clientset, secretName, namespace)
	return err
}

// executeSQL executes SQL on the primary PostgreSQL Pod. This occurs using the
// Kubernets exec function, which allows us to perform the request over
// a PostgreSQL connection that's authenticated with peer authentication
func executeSQL(pod *v1.Pod, sql string) (string, error) {
	// execute into the primary pod to run the query
	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(apiserver.RESTConfig,
		apiserver.Clientset, sqlCommand,
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
// In the future, this can be mdofifed to also support a password hashing type
// e.g. SCRAM :)
func generatePassword(username, password string, generatePassword bool, generatedPasswordLength int) (bool, string, string) {
	// first, an early exit: nothing is updated
	if password == "" && !generatePassword {
		return false, "", ""
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
		password = util.GeneratePassword(passwordLength)
	}

	// finally, hash the password
	hashedPassword := util.GeneratePostgreSQLMD5Password(username, password)

	// return!
	return true, password, hashedPassword
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
		err := kubeapi.Getpgclusters(apiserver.RESTClient, &clusterList, namespace)
		return clusterList, err
	}

	// try to build the cluster list based on either the selector or the list
	// of arguments...or both. First, start with the selector
	if selector != "" {
		err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList,
			selector, namespace)

		// if there is an error, return here with an empty cluster list
		if err != nil {
			return crv1.PgclusterList{}, err
		}
	}

	// now try to get clusters based specific cluster names
	for _, clusterName := range clusterNames {
		cluster := crv1.Pgcluster{}

		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster,
			clusterName, namespace)

		// if there is an error, capture it here and return here with an empty list
		if !found || err != nil {
			return crv1.PgclusterList{}, err
		}

		// if successful, append to the cluster list
		clusterList.Items = append(clusterList.Items, cluster)
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

	log.Debugf("rotate expired passwords on cluster [%s]", cluster.Name)

	// first, find the primary Pod. If we can't do that, no rense in continuing
	pod, err := util.GetPrimaryPod(apiserver.Clientset, cluster)

	if err != nil {
		result := msgs.UserResponseDetail{
			ClusterName:  cluster.Name,
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
	output, err := executeSQL(pod, sql)

	if err != nil {
		result := msgs.UserResponseDetail{
			ClusterName:  cluster.Name,
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
	for usernames.Scan() {
		// start building a result. The Username call strips off the newlines and
		// other garbage and returns the actual username
		result := msgs.UserResponseDetail{
			ClusterName: cluster.Name,
			Username:    strings.TrimSpace(usernames.Text()),
			ValidUntil:  validUntil,
		}

		// start building the SQL
		sql := fmt.Sprintf(sqlAlterRole, util.SQLQuoteIdentifier(result.Username))

		// generate a new password. Check to see if the user passed in a particular
		// length of the password, or passed in a password to rotate (though that
		// is not advised...). This forced the password to change
		_, password, hashedPassword := generatePassword(result.Username, request.Password, true, request.PasswordLength)

		result.Password = password
		sql = fmt.Sprintf("%s %s", sql,
			fmt.Sprintf(sqlPasswordClause, util.SQLQuoteLiteral(hashedPassword)))

		// build the "valid until" value into the SQL string
		sql = fmt.Sprintf("%s %s", sql,
			fmt.Sprintf(sqlValidUntilClause, util.SQLQuoteLiteral(result.ValidUntil)))

		// and this is enough to execute
		_, err := executeSQL(pod, sql)

		// if there is an error, record it here. The next step is to continue
		// iterating through the loop, and we will continue to do so
		if err != nil {
			result.Error = true
			result.ErrorMessage = err.Error()
		}

		results = append(results, result)
	}

	return results
}

// updateUser, though perhaps poorly named in context, performs the standard
// "ALTER ROLE" type functionality on a user, which is just updating a single
// user account on a single PostgreSQL cluster. This is in contrast with some
// of the bulk updates that can occur with updating a user (e.g. resetting
// expired passwords), which is why it's broken out into its own function
func updateUser(request *msgs.UpdateUserRequest, cluster *crv1.Pgcluster) msgs.UserResponseDetail {
	result := msgs.UserResponseDetail{
		ClusterName: cluster.Name,
		Username:    request.Username,
	}

	log.Debugf("updating user [%s] on cluster [%s]", result.Username, cluster.Name)

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
	isChanged, password, hashedPassword := generatePassword(result.Username,
		request.Password, request.RotatePassword, request.PasswordLength)

	if isChanged {
		result.Password = password
		sql = fmt.Sprintf("%s %s", sql,
			fmt.Sprintf(sqlPasswordClause, util.SQLQuoteLiteral(hashedPassword)))
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
		// append the VALID UNTIL special clause here for explictly disallowing
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
	if _, err := executeSQL(pod, sql); err != nil {
		log.Error(err)

		result.Error = true
		result.ErrorMessage = err.Error()

		// even though we return in the next line, having an explicit return here
		// in case we add any additional logic beyond this point
		return result
	}

	return result
}
