package userservice

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	//libpq uses this blank import
	"database/sql"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	_ "github.com/lib/pq"
	"strconv"
	"time"
)

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
var defaultPasswordLength = 8

//  User ...
// pgo user --change-password=bob --db=userdb
//  --expired=7 --managed=true --selector=env=research --update-passwords=true
//  --valid-days=30
func User(request *msgs.UserRequest) msgs.UserResponse {
	var err error
	resp := msgs.UserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	getDefaults()

	//set up the selector
	var sel string
	if request.Selector != "" {
		sel = request.Selector + "," + util.LABEL_PG_CLUSTER + "," + util.LABEL_PRIMARY + "=true"
	} else {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "--selector is required"
		return resp

	}

	log.Debug("selector string=[" + sel + "]")

	//get the clusters list
	clusterList := crv1.PgclusterList{}
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, sel, apiserver.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if len(clusterList.Items) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "no clusters found"
		return resp
	}

	for _, cluster := range clusterList.Items {
		selector := util.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name + "," + util.LABEL_PRIMARY + "=true"
		deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, apiserver.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		for _, d := range deployments.Items {
			info := getPostgresUserInfo(apiserver.Namespace, d.ObjectMeta.Name)

			if request.ChangePasswordForUser != "" {
				msg := "changing password of user " + request.ChangePasswordForUser + " on " + d.ObjectMeta.Name
				log.Debug(msg)
				resp.Results = append(resp.Results, msg)
				newPassword := util.GeneratePassword(defaultPasswordLength)
				newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)
				err = updatePassword(cluster.Spec.Name, info, request.ChangePasswordForUser, newPassword, newExpireDate, apiserver.Namespace)
				if err != nil {
					log.Error(err.Error())
					resp.Status.Code = msgs.Error
					resp.Status.Msg = err.Error()
					return resp
				}
			}

			if request.Expired != "" {
				results := callDB(info, d.ObjectMeta.Name, request.Expired)
				if len(results) > 0 {
					log.Debug("expired passwords....")
					for _, v := range results {
						resp.Results = append(resp.Results, "RoleName "+v.Rolname+" Role Valid Until "+v.Rolvaliduntil)
						log.Debug("RoleName " + v.Rolname + " Role Valid Until " + v.Rolvaliduntil)
						if request.UpdatePasswords {
							newPassword := util.GeneratePassword(defaultPasswordLength)
							newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)
							err = updatePassword(cluster.Spec.Name, v.ConnDetails, v.Rolname, newPassword, newExpireDate, apiserver.Namespace)
							if err != nil {
								log.Error("error in updating password")
							}
							log.Debug("new password for %s is %s new expiration is %s\n", v.Rolname, newPassword, newExpireDate)
						}
					}
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
		log.Debug("returned " + ts)
	}

	return results

}

// updatePassword ...
func updatePassword(clusterName string, p connInfo, username, newPassword, passwordExpireDate, namespace string) error {
	var err error
	var conn *sql.DB

	conn, err = sql.Open("postgres", "sslmode=disable user="+p.Username+" host="+p.Hostip+" port="+p.Port+" dbname="+p.Database+" password="+p.Password)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	//var ts string
	var rows *sql.Rows
	querystr := "ALTER user " + username + " PASSWORD '" + newPassword + "'"
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		return err
	}
	querystr = "ALTER user " + username + " VALID UNTIL '" + passwordExpireDate + "'"
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
		log.Debug(secretName + " secret does not exist")
		return nil
	}

	err = util.UpdateUserSecret(apiserver.Clientset, clusterName, username, newPassword, namespace)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	return err
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
func getPostgresUserInfo(namespace, clusterName string) connInfo {
	info := connInfo{}

	//get the service for the cluster
	service, found, err := kubeapi.GetService(apiserver.Clientset, clusterName, namespace)
	if !found || err != nil {
		return info
	}

	//get the secrets for this cluster
	selector := util.LABEL_PG_DATABASE + "=" + clusterName
	secrets, err := kubeapi.GetSecrets(apiserver.Clientset, selector, namespace)
	if err != nil {
		return info
	}

	//get the postgres user secret info
	var username, password, database, hostip string
	for _, s := range secrets.Items {
		username = string(s.Data["username"][:])
		password = string(s.Data["password"][:])
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

	return info
}

// addUser ...
func addUser(UserDBAccess, namespace, clusterName string, info connInfo, newUser string, ManagedUser bool) error {
	var conn *sql.DB
	var err error

	conn, err = sql.Open("postgres", "sslmode=disable user="+info.Username+" host="+info.Hostip+" port="+info.Port+" dbname="+info.Database+" password="+info.Password)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	var rows *sql.Rows

	querystr := "create user " + newUser
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if UserDBAccess != "" {
		querystr = "grant all on database " + UserDBAccess + " to  " + newUser
	} else {
		querystr = "grant all on database userdb to  " + newUser
	}
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

	//add a secret if managed
	if ManagedUser {
		err = util.CreateUserSecret(apiserver.Clientset, clusterName, newUser, info.Password, namespace)
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

	querystr := "drop owned by  " + user + " cascade"
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	querystr = "drop user if exists " + user
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
		err = util.DeleteUserSecret(apiserver.Clientset, clusterName, user, namespace)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}
	return err

}

// CreateUser ...
// pgo create user user1
func CreateUser(request *msgs.CreateUserRequest) msgs.CreateUserResponse {
	var err error
	resp := msgs.CreateUserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	getDefaults()

	log.Debug("createUser selector is " + request.Selector)
	if request.Selector == "" {
		log.Error("--selector value is empty not allowed")
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "error in selector"
		return resp
	}

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, request.Selector, apiserver.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	log.Debug("createUser clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {
		info := getPostgresUserInfo(apiserver.Namespace, c.Name)

		err = addUser(request.UserDBAccess, apiserver.Namespace, c.Name, info, request.Name, request.ManagedUser)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		} else {
			msg := "adding new user " + request.Name + " to " + c.Name
			log.Debug(msg)
			resp.Results = append(resp.Results, msg)
		}
		newPassword := util.GeneratePassword(defaultPasswordLength)
		newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)
		err = updatePassword(c.Name, info, request.Name, newPassword, newExpireDate, apiserver.Namespace)
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
func DeleteUser(name, selector string) msgs.DeleteUserResponse {
	var err error

	response := msgs.DeleteUserResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	clusterList := crv1.PgclusterList{}

	//get the clusters list
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, apiserver.Namespace)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	if len(clusterList.Items) == 0 {
		log.Debug("no clusters found")
		response.Status.Code = msgs.Error
		response.Status.Msg = "no clusters found"
		return response
	}

	var managed bool
	var msg, clusterName string

	for _, cluster := range clusterList.Items {
		clusterName = cluster.Spec.Name
		info := getPostgresUserInfo(apiserver.Namespace, clusterName)

		secretName := clusterName + "-" + name + "-secret"

		managed, err = isManaged(secretName)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		err = deleteUser(apiserver.Namespace, clusterName, info, name, managed)
		if err != nil {
			log.Error(err)
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		msg = name + " on " + clusterName + " removed managed=" + strconv.FormatBool(managed)
		log.Debug(msg)
		response.Results = append(response.Results, msg)

	}

	return response

}

func isManaged(secretName string) (bool, error) {
	_, found, err := kubeapi.GetSecret(apiserver.Clientset, secretName, apiserver.Namespace)
	if !found || err != nil {
		return false, err
	}

	return true, err

}

// ShowUser ...
func ShowUser(name, selector string) msgs.ShowUserResponse {
	var err error

	response := msgs.ShowUserResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]msgs.ShowUserDetail, 0)

	if selector == "" && name == "all" {
	} else {
		if selector == "" {
			selector = "name=" + name
		}
	}

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, apiserver.Namespace)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debug("clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {
		detail := msgs.ShowUserDetail{}
		detail.Cluster = c
		detail.Secrets, err = apiserver.GetSecrets(&c)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.Results = append(response.Results, detail)
	}

	return response

}
