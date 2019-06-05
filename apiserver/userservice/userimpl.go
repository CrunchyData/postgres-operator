package userservice

/*
Copyright 2019 Crunchy Data Solutions, Inc.
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
	"regexp"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
//  --expired=7 --selector=env=research --update-passwords=true
//  --valid-days=30
func User(request *msgs.UserRequest, ns string) msgs.UserResponse {
	var err error
	resp := msgs.UserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	getDefaults()

	log.Debugf("in User with password [%]", request.Password)
	if request.Password != "" {
		err = validPassword(request.Password)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "invalid password format"
			return resp
		}
	}

	//set up the selector
	var sel string
	if request.Selector != "" {
		sel = request.Selector + "," + config.LABEL_PG_CLUSTER
	} else {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "--selector is required"
		return resp

	}

	if request.UpdatePasswords && request.Expired == "" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "--expired is required when --update-passwords is specified"
		return resp
	}

	log.Debugf("selector string=[%s]", sel)

	//get the clusters list
	clusterList := crv1.PgclusterList{}
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, sel, ns)
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
		selector := config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name + "," + config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name
		deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		for _, d := range deployments.Items {
			//info, err := getPostgresUserInfo(ns, d.ObjectMeta.Name)
			info, err := getPostgresUserInfo(ns, cluster.Spec.Name)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}

			if request.ChangePasswordForUser != "" {
				msg := "changing password of user " + request.ChangePasswordForUser + " on " + d.ObjectMeta.Name
				log.Debug(msg)
				resp.Results = append(resp.Results, msg)
				newPassword := util.GeneratePassword(request.PasswordLength)
				if request.Password != "" {
					err := validPassword(request.Password)
					if err != nil {
						resp.Status.Code = msgs.Error
						resp.Status.Msg = "invalid password format, can not contain non-alphanumerics or start with numbers"
						return resp
					}
					newPassword = request.Password
				}
				newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)
				pgbouncer := cluster.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true"
				pgpool := cluster.Spec.UserLabels[config.LABEL_PGPOOL] == "true"

				err = updatePassword(cluster.Spec.Name, info, request.ChangePasswordForUser, newPassword, newExpireDate, ns, pgpool, pgbouncer, request.PasswordLength)
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
					log.Debug("expired passwords...")
					for _, v := range results {
						log.Debugf("RoleName %s Role Valid Until %s", v.Rolname, v.Rolvaliduntil)
						if request.UpdatePasswords {
							newPassword := util.GeneratePassword(request.PasswordLength)
							newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)
							pgbouncer := cluster.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true"
							pgpool := cluster.Spec.UserLabels[config.LABEL_PGPOOL] == "true"
							err = updatePassword(cluster.Spec.Name, v.ConnDetails, v.Rolname, newPassword, newExpireDate, ns, pgpool, pgbouncer, request.PasswordLength)
							if err != nil {
								log.Error("error in updating password")
								resp.Status.Code = msgs.Error
								resp.Status.Msg = err.Error()
								return resp
							}
							//log.Debug("new password for %s is %s new expiration is %s\n", v.Rolname, newPassword, newExpireDate)
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
		log.Debugf("returned %s", ts)
	}

	return results

}

// updatePassword ...
func updatePassword(clusterName string, p connInfo, username, newPassword, passwordExpireDate, namespace string, pgpool bool, pgbouncer bool, passwordLength int) error {
	var err error
	var conn *sql.DB

	err = validPassword(newPassword)
	if err != nil {
		return err
	}

	conn, err = sql.Open("postgres", "sslmode=disable user="+p.Username+" host="+p.Hostip+" port="+p.Port+" dbname="+p.Database+" password="+p.Password)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	var rows *sql.Rows
	querystr := "SELECT 'md5'|| md5('" + newPassword + username + "')"
	//log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	var md5Password string
	for rows.Next() {
		err = rows.Scan(&md5Password)
		if err != nil {
			log.Debug(err.Error())
			return err
		}
	}

	//querystr = "ALTER user " + username + " PASSWORD '" + newPassword + "'"
	querystr = "ALTER user " + username + " PASSWORD '" + md5Password + "'"
	//log.Debug(querystr)
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

	if pgpool {
		err := reconfigurePgpool(clusterName, namespace)
		if err != nil {
			log.Error(err)
			return err
		}
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

	if request.Name != "" {
		parts := strings.Split(request.Name, " ")
		if len(parts) > 1 {
			return errors.New("invalid user name format, can not container special characters")
		}
	}
	//validate userdb if entered
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

	querystr = "create user " + request.Name
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if request.UserDBAccess != "" {
		querystr = "grant all on database " + request.UserDBAccess + " to  " + request.Name
	} else {
		querystr = "grant all on database userdb to  " + request.Name
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
	if request.ManagedUser {
		if request.Password != "" {
			info.Password = request.Password
		}
		err = util.CreateUserSecret(apiserver.Clientset, clusterName, request.Name, info.Password, namespace, request.PasswordLength)
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
// pgo create user user1
func CreateUser(request *msgs.CreateUserRequest, ns string) msgs.CreateUserResponse {
	var err error
	resp := msgs.CreateUserResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	getDefaults()

	log.Debugf("createUser selector is ", request.Selector)
	if request.Selector == "" {
		log.Error("--selector value is empty not allowed")
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "error in selector"
		return resp
	}

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, request.Selector, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if len(clusterList.Items) == 0 {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "no clusters found with that selector"
		return resp
	}

	log.Debugf("createUser clusters found len is %d", len(clusterList.Items))

	re := regexp.MustCompile("^[a-z0-9.-]*$")
	if !re.MatchString(request.Name) {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "user name is required to be lowercase letters and numbers only."
		return resp
	}
	if request.Password != "" {
		err := validPassword(request.Password)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	for _, c := range clusterList.Items {
		info, err := getPostgresUserInfo(ns, c.Name)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		err = addUser(request, ns, c.Name, info)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		} else {
			msg := "adding new user " + request.Name + " to " + c.Name
			log.Debug(msg)
			resp.Results = append(resp.Results, msg)
		}

		if request.PasswordLength == 0 {
			request.PasswordLength = defaultPasswordLength
		}

		newPassword := util.GeneratePassword(request.PasswordLength)
		if request.Password != "" {
			newPassword = request.Password
		}
		newExpireDate := GeneratePasswordExpireDate(request.PasswordAgeDays)

		pgbouncer := c.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true"
		pgpool := c.Spec.UserLabels[config.LABEL_PGPOOL] == "true"
		err = updatePassword(c.Name, info, request.Name, newPassword, newExpireDate, ns, pgpool, pgbouncer, request.PasswordLength)
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
func DeleteUser(name, selector, ns string) msgs.DeleteUserResponse {
	var err error

	response := msgs.DeleteUserResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	log.Debugf("DeleteUser called name=%s", name)
	clusterList := crv1.PgclusterList{}

	//get the clusters list
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, ns)
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

	re := regexp.MustCompile("^[a-z0-9.-]*$")
	if !re.MatchString(name) {
		response.Status.Code = msgs.Error
		response.Status.Msg = "user name is required to be lowercase letters and numbers only."
		return response
	}

	var managed bool
	var msg, clusterName string

	for _, cluster := range clusterList.Items {
		clusterName = cluster.Spec.Name
		info, err := getPostgresUserInfo(ns, clusterName)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		secretName := clusterName + "-" + name + "-secret"

		managed, err = isManaged(secretName, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		log.Debugf("DeleteUser %s managed %t", name, managed)

		err = deleteUser(ns, clusterName, info, name, managed)
		if err != nil {
			log.Error(err)
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		msg = name + " on " + clusterName + " removed managed=" + strconv.FormatBool(managed)
		log.Debug(msg)
		response.Results = append(response.Results, msg)

		//see if any pooler needs to be reconfigured
		if managed {
			if cluster.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true" {
				err := reconfigurePgbouncer(clusterName, ns)
				if err != nil {
					log.Error(err)
					response.Status.Code = msgs.Error
					response.Status.Msg = err.Error()
					return response
				}
			}
			if cluster.Spec.UserLabels[config.LABEL_PGPOOL] == "true" {
				err := reconfigurePgpool(clusterName, ns)
				if err != nil {
					log.Error(err)
					response.Status.Code = msgs.Error
					response.Status.Msg = err.Error()
					return response
				}
			}
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
func ShowUser(name, selector, expired, ns string) msgs.ShowUserResponse {
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
		&clusterList, selector, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	var expiredInt int
	if expired != "" {
		expiredInt, err = strconv.Atoi(expired)
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
		detail.ExpiredMsgs = make([]string, 0)

		if expired != "" {
			selector := config.LABEL_PG_CLUSTER + "=" + c.Spec.Name + "," + config.LABEL_SERVICE_NAME + "=" + c.Spec.Name
			deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}

			for _, d := range deployments.Items {
				info, err := getPostgresUserInfo(ns, d.ObjectMeta.Name)
				if err != nil {
					response.Status.Code = msgs.Error
					response.Status.Msg = err.Error()
					return response
				}

				if expired != "" {
					results := callDB(info, d.ObjectMeta.Name, expired)
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

		detail.Secrets, err = apiserver.GetSecrets(&c, ns)
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

func reconfigurePgpool(clusterName, ns string) error {
	var err error
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = config.LABEL_PGPOOL_TASK_RECONFIGURE + "-" + clusterName
	spec.TaskType = crv1.PgtaskReconfigurePgpool
	spec.StorageSpec = crv1.PgStorageSpec{}
	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_PGPOOL_TASK_CLUSTER] = clusterName

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}

	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[config.LABEL_PGPOOL_TASK_RECONFIGURE] = "true"

	//delete any existing pgtask for this
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

func validPassword(psw string) error {

	if len(psw) > 16 {
		return errors.New("valid passwords are less than 16 chars")
	}

	numbers := "0123456789"
	isAlpha := regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString

	if len(psw) < 1 {
		return errors.New("passwords can not be zero length")
	}

	firstChar := string(psw[0])
	log.Debugf("1st char is %s", firstChar)
	if strings.Contains(numbers, firstChar) {
		//log.Debugf("%s is not valid due to starting with a number", username)
		return errors.New("passwords can not start with a number")
	} else if !isAlpha(psw) {
		//log.Debugf("%q is not valid\n", username)
		return errors.New("password does not match standard pattern")

	} else {
		//log.Debugf("%q is valid\n", username)
		return nil
	}

	return nil

}
