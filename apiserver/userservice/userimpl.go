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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"math/rand"
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
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const charset = "!@#~$%^&*{_+=-;}abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456890"

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

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

// defaultPasswordLength password length
var defaultPasswordLength = 32

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
						newPassword := GeneratePassword(request.PasswordLength)
						newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)
						pgbouncer := cluster.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true"
						err = updatePassword(cluster.Spec.Name, v.ConnDetails, v.Rolname, newPassword, newExpireDate, request.Namespace, pgbouncer, request.PasswordLength)
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
				pgbouncer := cluster.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true"

				err = updatePassword(cluster.Spec.Name, info, request.Username, request.Password, newExpireDate, request.Namespace, pgbouncer, request.PasswordLength)
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
func updatePassword(clusterName string, p connInfo, username, newPassword, passwordExpireDate, namespace string, pgbouncer bool, passwordLength int) error {
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

	err = util.UpdateUserSecret(apiserver.Clientset, clusterName, username, newPassword, namespace, passwordLength)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	if pgbouncer {
		err := reconfigurePgbouncer(clusterName, namespace)
		if err != nil {
			log.Error(err)
			return err
		}
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
	str = apiserver.Pgo.Cluster.PasswordLength
	if str != "" {
		defaultPasswordLength, _ = strconv.Atoi(str)
		log.Debugf("PasswordLength set to %d\n", defaultPasswordLength)
	}

}

// getPostgresUserInfo...
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

// addUser ...
func addUser(request *msgs.CreateUserRequest, namespace, clusterName string, info connInfo) error {
	var conn *sql.DB
	var err error

	conn, err = sql.Open("postgres", "sslmode=disable user="+info.Username+" host="+info.Hostip+" port="+info.Port+" dbname="+info.Database+" password="+info.Password)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	var rows *sql.Rows
	var querystr string

	if request.Username != "" {
		parts := strings.Split(request.Username, " ")
		if len(parts) > 1 {
			return errors.New("invalid user name format, can not container special characters")
		}
	}
	//validate userdb if entered
	/**
	if request.UserDBAccess != "" {
		parts := strings.Split(request.UserDBAccess, " ")
		if len(parts) > 1 {
			return errors.New("invalid db name format, can not container special characters")
		}
		querystr = "select count(datname) from pg_catalog.pg_database where datname = '" + request.UserDBAccess + "'"
		log.Debug(querystr)
		rows, err = conn.Query(querystr)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		var returnedName int
		for rows.Next() {
			err = rows.Scan(&returnedName)
			if err != nil {
				log.Error(err)
				return err
			}
			log.Debugf(" returned name %d", returnedName)
			if returnedName == 0 {
				return errors.New("dbname is not valid database name")
			}
		}
	}
	*/

	querystr = "create user \"" + request.Username + "\""
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	/**
	if request.UserDBAccess != "" {
		querystr = "grant all on database " + request.UserDBAccess + " to  " + request.Username
	} else {
		querystr = "grant all on database userdb to  " + request.Username
	}
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	*/

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()

	//add a secret if managed
	if request.ManagedUser {
		if request.Password != "" {
			info.Password = request.Password
		}
		err = util.CreateUserSecret(apiserver.Clientset, clusterName, request.Username, info.Password, namespace, request.PasswordLength)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}
	return err

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
	var err error
	resp := msgs.CreateUserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	getDefaults()

	log.Debugf("createUser selector is ", request.Selector)
	if request.Selector == "" && request.AllFlag == false && len(request.Clusters) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "--selector, --all, or list of cluster names is required for this command"
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

	log.Debugf("createUser clusters found len is %d", len(clusterList.Items))

	re := regexp.MustCompile("^[a-z0-9.-]*$")
	if !re.MatchString(request.Username) {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "user name is required to contain lowercase letters, numbers, '.' and '-' only."
		return resp
	}

	for _, c := range clusterList.Items {
		info, err := getPostgresUserInfo(request.Namespace, c.Name)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		err = addUser(request, request.Namespace, c.Name, info)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		} else {
			msg := "adding new user " + request.Username + " to " + c.Name
			log.Debug(msg)
			resp.Results = append(resp.Results, msg)
		}

		if request.PasswordLength == 0 {
			request.PasswordLength = defaultPasswordLength
		}

		newPassword := GeneratePassword(request.PasswordLength)
		if request.Password != "" {
			newPassword = request.Password
		}
		newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)

		pgbouncer := c.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true"
		err = updatePassword(c.Name, info, request.Username, newPassword, newExpireDate, request.Namespace, pgbouncer, request.PasswordLength)
		if err != nil {
			log.Error(err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		//publish event for create user
		topics := make([]string, 1)
		topics[0] = events.EventTopicUser

		f := events.EventCreateUserFormat{
			EventHeader: events.EventHeader{
				Namespace: request.Namespace,
				Username:  pgouser,
				Topic:     topics,
				Timestamp: time.Now(),
				EventType: events.EventCreateUser,
			},
			Clustername:      c.Name,
			PostgresUsername: request.Username,
			PostgresPassword: newPassword,
			Managed:          request.ManagedUser,
		}

		err = events.Publish(f)
		if err != nil {
			log.Error(err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

	}
	return resp

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

	clusterList := crv1.PgclusterList{}

	//get the clusters list
	if request.Selector != "" {
		err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
			&clusterList, request.Selector, request.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	} else if request.AllFlag {
		err = kubeapi.Getpgclusters(apiserver.RESTClient, &clusterList, request.Namespace)
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

		//see if any connection proxies (e.g. pgbouncer) need to be reconfigured
		if managed {
			if cluster.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true" {
				err := reconfigurePgbouncer(clusterName, request.Namespace)
				if err != nil {
					log.Error(err)
					response.Status.Code = msgs.Error
					response.Status.Msg = err.Error()
					return response
				}
			}
		}

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

func reconfigurePgbouncer(clusterName, ns string) error {
	var err error
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = config.LABEL_PGBOUNCER_TASK_RECONFIGURE + "-" + clusterName
	spec.TaskType = crv1.PgtaskReconfigurePgbouncer
	spec.StorageSpec = crv1.PgStorageSpec{}
	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER] = clusterName

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}

	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[config.LABEL_PGBOUNCER_TASK_RECONFIGURE] = "true"

	//try deleting any previous pgtask for this cluster
	err = kubeapi.Deletepgtask(apiserver.RESTClient, spec.Name, ns)
	if kerrors.IsNotFound(err) {
		log.Debugf("pgtask %s is not found prior which is ok", spec.Name)
	} else if err != nil {
		log.Error(err)
		return err
	}

	//create the pgtask
	err = kubeapi.Createpgtask(apiserver.RESTClient, newInstance, ns)
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}

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

// stringWithCharset returns a generated string value
func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// GeneratePassword generate a password of a given length
func GeneratePassword(length int) string {
	return stringWithCharset(length, charset)
}
