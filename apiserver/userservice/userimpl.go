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
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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
	"k8s.io/client-go/kubernetes"
)

const (
	errSystemAccountFormat = `"%s" is a system account and cannot be used`
)

const (
	// sqlCreateRole is SQL that allows a new PostgreSQL user to be created. To
	// safely use this function, the role name and passsword must be escaped to
	// avoid SQL injections, which is handled in the SetPostgreSQLPassword
	// function
	sqlCreateRole = `CREATE ROLE %s PASSWORD %s LOGIN`
	// sqlValidUntilClause is a clause that allows one to pass in a valid until
	// timestamp. The value must be escaped to avoid SQL injections, using the
	// util.SQLQuoteLiteral function
	sqlValidUntilClause = `VALID UNTIL %s`
)

var alterRole = "DO $_$BEGIN EXECUTE $$ALTER USER $$ || quote_ident($$%s$$) || $$ PASSWORD $$ || quote_literal($$%s$$); END$_$;"

// connInfo ....
type connInfo struct {
	Username string
	Hostip   string
	Port     string
	Database string
	Password string
}

// pswResult ...
type pswResult struct {
	Rolname       string
	Rolvaliduntil string
	ConnDetails   connInfo
}

// defaultPasswordAgeDays password age length
var defaultPasswordAgeDays = 365

//  User ...
// pgo user --change-password=bob --db=userdb
//  --expired=7 --selector=env=research --update-passwords=true
//  --valid-days=30
func UpdateUser(request *msgs.UpdateUserRequest, pgouser string) msgs.UpdateUserResponse {
	var err error
	resp := msgs.UpdateUserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	getDefaults()

	if request.Username == "" && request.Expired == "" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "Either --expired or --username must be set."
		return resp
	}

	if request.Username != "" && request.Expired != "" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "The --expired cannot be used when a user is specified."
		return resp
	}

	//set up the selector
	if request.Selector == "" && request.AllFlag == false && len(request.Clusters) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "--selector, --all, or list of cluster names  is required"
		return resp

	}

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	if request.Selector != "" {
		err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
			&clusterList, request.Selector, request.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	} else if request.AllFlag {
		err = kubeapi.Getpgclusters(apiserver.RESTClient, &clusterList, request.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	} else {
		for i := 0; i < len(request.Clusters); i++ {
			cluster := crv1.Pgcluster{}
			found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, request.Clusters[i], request.Namespace)
			if !found {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
			clusterList.Items = append(clusterList.Items, cluster)
		}

	}

	if len(clusterList.Items) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "no clusters found"
		return resp
	}

	log.Debugf("user UpdateUser %d clusters to work on", len(clusterList.Items))

	for _, cluster := range clusterList.Items {
		selector := config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name + "," + config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name
		deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, request.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		for _, d := range deployments.Items {
			info, err := getPostgresUserInfo(request.Namespace, cluster.Spec.Name)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}

			if request.ExpireUser {
				expiredDate := GeneratePasswordExpireDate(-2)
				err := setUserValidUntil(info, request.Username, expiredDate)
				if err != nil {
					resp.Status.Code = msgs.Error
					resp.Status.Msg = err.Error()
					return resp
				}
				log.Debugf("expiring user %s", request.Username)
			}

			if request.Expired != "" {
				results := callDB(info, d.ObjectMeta.Name, request.Expired)
				if len(results) > 0 {
					log.Debug("expired passwords...")
					for _, v := range results {
						log.Debugf("RoleName %s Role Valid Until %s", v.Rolname, v.Rolvaliduntil)
						newPassword := util.GeneratePassword(request.PasswordLength)
						newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)
						err = updatePassword(cluster.Spec.Name, v.ConnDetails, v.Rolname, newPassword, newExpireDate, request.Namespace, request.PasswordLength)
						if err != nil {
							log.Error("error in updating password")
							resp.Status.Code = msgs.Error
							resp.Status.Msg = err.Error()
							return resp
						}
					}
				}
			}

			if request.Password != "" {
				//if the password is being changed...
				msg := "changing password of user " + request.Username + " on " + d.ObjectMeta.Name
				log.Debug(msg)
				resp.Results = append(resp.Results, msg)
				newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)

				err = updatePassword(cluster.Spec.Name, info, request.Username, request.Password, newExpireDate, request.Namespace, request.PasswordLength)
				if err != nil {
					log.Error(err.Error())
					resp.Status.Code = msgs.Error
					resp.Status.Msg = err.Error()
					return resp
				}

				//publish event for password change
				topics := make([]string, 1)
				topics[0] = events.EventTopicUser

				f := events.EventChangePasswordUserFormat{
					EventHeader: events.EventHeader{
						Namespace: request.Namespace,
						Username:  pgouser,
						Timestamp: time.Now(),
						Topic:     topics,
						EventType: events.EventChangePasswordUser,
					},
					Clustername:      cluster.Spec.Name,
					PostgresUsername: request.Username,
					PostgresPassword: request.Password,
				}

				err = events.Publish(f)
				if err != nil {
					log.Error(err.Error())
					resp.Status.Code = msgs.Error
					resp.Status.Msg = err.Error()
					return resp
				}
			} else if request.PasswordAgeDaysUpdate {
				//if the password is not being changed and the valid-days flag is set
				msg := "updating valid days for password of user " + request.Username + " on " + d.ObjectMeta.Name
				log.Debug(msg)
				resp.Results = append(resp.Results, msg)
				newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)

				err = updatePasswordValidUntil(cluster.Spec.Name, info, request.Username, newExpireDate, request.Namespace)
				if err != nil {
					log.Error(err.Error())
					resp.Status.Code = msgs.Error
					resp.Status.Msg = err.Error()
					return resp
				}
			}
		}
	}
	return resp

}

// callDB ...
func callDB(info connInfo, clusterName, maxdays string) []pswResult {
	var conn *sql.DB
	var err error

	results := []pswResult{}

	conn, err = sql.Open("postgres", "sslmode=disable user="+info.Username+" host="+info.Hostip+" port="+info.Port+" dbname="+info.Database+" password="+info.Password)
	if err != nil {
		log.Debug(err.Error())
		return results
	}

	var ts string
	var rows *sql.Rows

	querystr := "SELECT rolname, rolvaliduntil as expiring_soon FROM pg_authid WHERE rolvaliduntil < now() + '" + maxdays + " days'"
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		return results
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()
	for rows.Next() {
		p := pswResult{}
		c := connInfo{Username: info.Username, Hostip: info.Hostip, Port: info.Port, Database: info.Database, Password: info.Password}
		p.ConnDetails = c

		if err = rows.Scan(&p.Rolname, &p.Rolvaliduntil); err != nil {
			log.Debug(err.Error())
			return results
		}
		results = append(results, p)
		log.Debugf("returned %s", ts)
	}

	return results

}

// updatePassword ...
func updatePassword(clusterName string, p connInfo, username, newPassword, passwordExpireDate, namespace string, passwordLength int) error {
	var err error
	var conn *sql.DB

	conn, err = sql.Open("postgres", "sslmode=disable user="+p.Username+" host="+p.Hostip+" port="+p.Port+" dbname="+p.Database+" password="+p.Password)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	var rows *sql.Rows

	// Pre-hash the password in the PostgreSQL MD5 format to prevent the
	// plaintext value from appearing in the PostgreSQL logs.
	md5Password := "md5" + util.GetMD5HashForAuthFile(newPassword+username)

	// This call is the equivalent to
	// "ALTER USER " + username + " PASSWORD '" + md5Password + "'"
	_, err = AlterRole(conn, username, md5Password)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	querystr := "ALTER user \"" + username + "\" VALID UNTIL '" + passwordExpireDate + "'"
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
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

	//see if a secret exists for this user taco-user0-secret
	secretName := clusterName + "-" + username + "-" + "secret"
	_, _, err = util.GetPasswordFromSecret(apiserver.Clientset, namespace, secretName)
	if err != nil {
		log.Debugf("%s secret does not exist", secretName)
		return nil
	}

	err = util.UpdateUserSecret(apiserver.Clientset, clusterName, username, newPassword, namespace)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	return err
}

// updatePasswordValidUntil  ...
func updatePasswordValidUntil(clusterName string, p connInfo, username, passwordExpireDate, namespace string) error {
	var err error
	var conn *sql.DB

	conn, err = sql.Open("postgres", "sslmode=disable user="+p.Username+" host="+p.Hostip+" port="+p.Port+" dbname="+p.Database+" password="+p.Password)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	var rows *sql.Rows

	querystr := "ALTER user \"" + username + "\" VALID UNTIL '" + passwordExpireDate + "'"
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
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

	return err
}

func AlterRole(conn *sql.DB, username, password string) (sql.Result, error) {
	return conn.Exec(fmt.Sprintf(alterRole, username, password))
}

// GeneratePasswordExpireDate ...
func GeneratePasswordExpireDate(daysFromNow int) string {

	if daysFromNow == -1 {
		return "infinity"
	}

	now := time.Now()
	totalHours := daysFromNow * 24
	diffDays, _ := time.ParseDuration(strconv.Itoa(totalHours) + "h")
	futureTime := now.Add(diffDays)
	return futureTime.Format("2006-01-02")

}

// getDefaults ....
func getDefaults() {
	str := apiserver.Pgo.Cluster.PasswordAgeDays
	if str != "" {
		defaultPasswordAgeDays, _ = strconv.Atoi(str)
		log.Debugf("PasswordAgeDays set to %d\n", defaultPasswordAgeDays)
	}
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

// CreateUser ...
// pgo create user mycluster --username=user1
// pgo create user --username=user1 --managed --all
// pgo create user --username=user1 --managed --selector=name=mycluster
func CreateUser(request *msgs.CreateUserRequest, pgouser string) msgs.CreateUserResponse {
	response := msgs.CreateUserResponse{
		Results: []msgs.CreateUserResponseDetail{},
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
	validUntilDays := request.PasswordAgeDays
	validUntil := ""
	sqlValidUntil := ""

	// if this is zero (or less than zero), attempt to set the value supplied by
	// the server. If it's still zero, then the user can create a password without
	// expiration
	if validUntilDays <= 0 {
		validUntilDays = util.GeneratedPasswordValidUntilDays(apiserver.Pgo.Cluster.PasswordAgeDays)
	}

	// ...and we can generate the SQL snippet here, as it won't change
	if validUntilDays > 0 {
		validUntil = GeneratePasswordExpireDate(validUntilDays)
		sqlValidUntil = fmt.Sprintf(sqlValidUntilClause, util.SQLQuoteLiteral(validUntil))
	}

	// iterate through each cluster and add the new PostgreSQL role to each pod
	for _, cluster := range clusterList.Items {
		log.Debugf("creating user on cluster [%s]", cluster.Name)

		result := msgs.CreateUserResponseDetail{
			ClusterName: cluster.Name,
			Username:    request.Username,
			ValidUntil:  validUntil,
		}

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

		// Set the password
		// See if the user set a password, otherwise generate a password
		if request.Password != "" {
			result.Password = request.Password
		} else {
			// Determine if the user passed in a password length, otherwise us the
			// default
			passwordLength := request.PasswordLength

			if passwordLength == 0 {
				passwordLength = util.GeneratedPasswordLength(apiserver.Pgo.Cluster.PasswordLength)
			}

			// generate the password
			result.Password = util.GeneratePassword(passwordLength)
		}
		// create the hashed value of the password, and then set it in PostgreSQL!
		hashedPassword := util.GeneratePostgreSQLMD5Password(request.Username, result.Password)

		// attempt to set the password!
		if err := util.SetPostgreSQLPassword(apiserver.Clientset, apiserver.RESTConfig, pod, request.Username, hashedPassword, sql); err != nil {
			log.Error(err)

			result.Error = true
			result.ErrorMessage = err.Error()

			response.Results = append(response.Results, result)
			continue
		}

		// if this user is "managed" by the Operator, add a secret. If there is an
		// error, we can fall through as the next step is appending the results
		if request.ManagedUser {
			if err := util.CreateUserSecret(apiserver.Clientset, cluster.Name, request.Username,
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

// ShowUser ...
func ShowUser(request *msgs.ShowUserRequest) msgs.ShowUserResponse {
	var err error

	response := msgs.ShowUserResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]msgs.ShowUserDetail, 0)

	clusterList := crv1.PgclusterList{}
	if request.AllFlag {
		err = kubeapi.Getpgclusters(apiserver.RESTClient, &clusterList, request.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	} else if request.Selector != "" {
		err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
			&clusterList, request.Selector, request.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	} else {
		for i := 0; i < len(request.Clusters); i++ {
			cluster := crv1.Pgcluster{}
			found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, request.Clusters[i], request.Namespace)
			if !found {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
			clusterList.Items = append(clusterList.Items, cluster)
		}
	}

	if len(clusterList.Items) == 0 {
		response.Status.Code = msgs.Error
		response.Status.Msg = "no clusters found"
		return response
	}

	var expiredInt int
	if request.Expired != "" {
		expiredInt, err = strconv.Atoi(request.Expired)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = "--expired is not a valid integer"
			return response
		}
		if expiredInt < 1 {
			response.Status.Code = msgs.Error
			response.Status.Msg = "--expired is requited to be greater than 0"
			return response
		}
	}

	log.Debugf("clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {
		detail := msgs.ShowUserDetail{}
		detail.Cluster = c
		detail.ExpiredDays = expiredInt
		detail.ExpiredMsgs = make([]string, 0)

		if request.Expired != "" {
			detail.ExpiredOutput = true
			selector := config.LABEL_PG_CLUSTER + "=" + c.Spec.Name + "," + config.LABEL_SERVICE_NAME + "=" + c.Spec.Name
			deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, request.Namespace)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}

			for _, d := range deployments.Items {
				info, err := getPostgresUserInfo(request.Namespace, d.ObjectMeta.Name)
				if err != nil {
					response.Status.Code = msgs.Error
					response.Status.Msg = err.Error()
					return response
				}

				if request.Expired != "" {
					results := callDB(info, d.ObjectMeta.Name, request.Expired)
					if len(results) > 0 {
						log.Debug("expired passwords...")
						for _, v := range results {
							detail.ExpiredMsgs = append(detail.ExpiredMsgs, "RoleName "+v.Rolname+" Role Valid Until "+v.Rolvaliduntil)
							log.Debugf("RoleName %s Role Valid Until %s", v.Rolname, v.Rolvaliduntil)

						}
					}
				}

			}
		}

		detail.Secrets, err = apiserver.GetSecrets(&c, request.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.Results = append(response.Results, detail)
	}

	return response

}

func deleteUserSecret(clientset *kubernetes.Clientset, clustername, username, namespace string) error {
	//delete current secret
	secretName := clustername + "-" + username + "-secret"

	err := kubeapi.DeleteSecret(clientset, secretName, namespace)
	return err
}

// TODO: delete
func setUserValidUntil(p connInfo, username, passwordExpireDate string) error {
	var err error
	var conn *sql.DB
	var rows *sql.Rows

	conn, err = sql.Open("postgres", "sslmode=disable user="+p.Username+" host="+p.Hostip+" port="+p.Port+" dbname="+p.Database+" password="+p.Password)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	querystr := "ALTER user \"" + username + "\" VALID UNTIL '" + passwordExpireDate + "'"
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
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

	return nil
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
