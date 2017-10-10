/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

package cmd

import (
	"database/sql"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"github.com/crunchydata/kraken/util"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/labels"
	"os"
	"strconv"
	"time"
	//"io/ioutil"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"os/user"
	//"strings"
)

type ConnInfo struct {
	Username string
	Hostip   string
	Port     string
	Database string
	Password string
}
type PswResult struct {
	Rolname       string
	Rolvaliduntil string
	ConnDetails   ConnInfo
}

const DEFAULT_AGE_DAYS = 365
const DEFAULT_PSW_LEN = 8

var PasswordAgeDays, PasswordLength int

var ChangePasswordForUser string
var DeleteUser string
var ValidDays string
var UserDBAccess string
var AddUser string
var Expired string
var UpdatePasswords bool
var ManagedUser bool

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "manage users",
	Long: `USER allows you to manage users and passwords across a set of clusters
For example:

pgo user --selector=name=mycluster --update
pgo user --expired=7 --selector=name=mycluster
pgo user --add-user=bob --selector=sname=mycluster
pgo user --change-password=bob --selector=sname=mycluster
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("user called")
		userManager()
	},
}

func init() {
	RootCmd.AddCommand(userCmd)

	userCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	userCmd.Flags().StringVarP(&Expired, "expired", "e", "", "--expired=7 shows passwords that will expired in 7 days")
	userCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "v", 30, "--valid-days=7 sets passwords for new users to 7 days")
	userCmd.Flags().StringVarP(&AddUser, "add-user", "a", "", "--add-user=bob adds a new user to selective clusters")
	userCmd.Flags().StringVarP(&ChangePasswordForUser, "change-password", "c", "", "--change-password=bob updates the password for a user on selective clusters")
	userCmd.Flags().StringVarP(&UserDBAccess, "db", "b", "", "--db=userdb grants the user access to a database")
	userCmd.Flags().StringVarP(&DeleteUser, "delete-user", "d", "", "--delete-user=bob deletes a user on selective clusters")
	userCmd.Flags().BoolVarP(&UpdatePasswords, "update-passwords", "u", false, "--update-passwords performs password updating on expired passwords")
	userCmd.Flags().BoolVarP(&ManagedUser, "managed", "m", false, "--managed creates a user with secrets")
	getDefaults()

}

func userManager() {
	//build the selector based on the selector parameter

	//set up the selector
	var sel string
	if Selector != "" {
		sel = Selector + ",pg-cluster,!replica"
	} else {
		log.Error("--selector is required")
		return

	}
	log.Debug("selector string=[" + sel + "]")

	myselector, err := labels.Parse(sel)
	if err != nil {
		log.Error("could not parse --selector value " + err.Error())
		return
	}

	//get the clusters list
	clusterList := crv1.PgclusterList{}
	err = RestClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(Namespace).
		LabelsSelectorParam(myselector).
		Do().
		Into(&clusterList)
	if err != nil {
		log.Error("error getting cluster list" + err.Error())
		return
	}

	if len(clusterList.Items) == 0 {
		fmt.Println("no clusters found")
		return
	}

	for _, cluster := range clusterList.Items {

		//log.Infoln("working on cluster " + cluster.Spec.Name)
		//log.Infof("clusters %v\n", clusterList.Items)
		sel = "pg-cluster=" + cluster.Spec.Name + ",!replica"
		lo := meta_v1.ListOptions{LabelSelector: sel}
		deployments, err := Clientset.ExtensionsV1beta1().Deployments(Namespace).List(lo)
		if err != nil {
			log.Error("error getting list of deployments" + err.Error())
			return
		}

		for _, d := range deployments.Items {
			info := getPostgresUserInfo(d.ObjectMeta.Name)

			if ChangePasswordForUser != "" {
				fmt.Println("changing password of user " + ChangePasswordForUser + " on " + d.ObjectMeta.Name)
				newPassword := util.GeneratePassword(PasswordLength)
				newExpireDate := GeneratePasswordExpireDate(PasswordAgeDays)
				err = updatePassword(cluster.Spec.Name, info, ChangePasswordForUser, newPassword, newExpireDate)
				if err != nil {
					log.Error(err.Error())
					os.Exit(2)
				}
			}
			if DeleteUser != "" {
				fmt.Println("deleting user " + DeleteUser + " from " + d.ObjectMeta.Name)
				deleteUser(cluster.Spec.Name, info, DeleteUser)
			}
			if AddUser != "" {
				fmt.Println("adding new user " + AddUser + " to " + d.ObjectMeta.Name)
				addUser(d.ObjectMeta.Name, info, AddUser)
				newPassword := util.GeneratePassword(PasswordLength)
				newExpireDate := GeneratePasswordExpireDate(PasswordAgeDays)
				err = updatePassword(cluster.Spec.Name, info, AddUser, newPassword, newExpireDate)
				if err != nil {
					log.Error(err.Error())
					os.Exit(2)
				}
			}

			if Expired != "" {
				results := callDB(info, d.ObjectMeta.Name, Expired)
				if len(results) > 0 {
					fmt.Println("expired passwords....")
					for _, v := range results {
						fmt.Printf("RoleName %s Role Valid Until %s\n", v.Rolname, v.Rolvaliduntil)
						if UpdatePasswords {
							newPassword := util.GeneratePassword(PasswordLength)
							newExpireDate := GeneratePasswordExpireDate(PasswordAgeDays)
							err = updatePassword(cluster.Spec.Name, v.ConnDetails, v.Rolname, newPassword, newExpireDate)
							if err != nil {
								fmt.Println("error in updating password")
							}
							fmt.Printf("new password for %s is %s new expiration is %s\n", v.Rolname, newPassword, newExpireDate)
						}
					}
				}
			}

		}
	}

}

func callDB(info ConnInfo, clusterName, maxdays string) []PswResult {
	var conn *sql.DB
	var err error

	results := []PswResult{}

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
		p := PswResult{}
		c := ConnInfo{Username: info.Username, Hostip: info.Hostip, Port: info.Port, Database: info.Database, Password: info.Password}
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

func updatePassword(clusterName string, p ConnInfo, username, newPassword, passwordExpireDate string) error {
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

	err = util.UpdateUserSecret(Clientset, clusterName, username, newPassword, Namespace)
	if err != nil {
		log.Debug(err.Error())
		return err
	}
	return err
}

func GeneratePasswordExpireDate(daysFromNow int) string {

	now := time.Now()
	totalHours := daysFromNow * 24
	diffDays, _ := time.ParseDuration(strconv.Itoa(totalHours) + "h")
	futureTime := now.Add(diffDays)
	return futureTime.Format("2006-01-02")

}

func getDefaults() {
	PasswordAgeDays = DEFAULT_AGE_DAYS
	PasswordLength = DEFAULT_PSW_LEN
	str := viper.GetString("CLUSTER.PASSWORD_AGE_DAYS")
	if str != "" {
		PasswordAgeDays, _ = strconv.Atoi(str)
		log.Debugf("PasswordAgeDays set to %d\n", PasswordAgeDays)

	}
	str = viper.GetString("CLUSTER.PASSWORD_LENGTH")
	if str != "" {
		PasswordLength, _ = strconv.Atoi(str)
		log.Debugf("PasswordLength set to %d\n", PasswordLength)
	}

}

func getPostgresUserInfo(clusterName string) ConnInfo {
	info := ConnInfo{}

	//get the service for the cluster
	service, err := Clientset.CoreV1().Services(Namespace).Get(clusterName, meta_v1.GetOptions{})
	if err != nil {
		log.Error("error getting list of services" + err.Error())
		os.Exit(2)
		return info
	}

	//get the secrets for this cluster
	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + clusterName}
	secrets, err := Clientset.Secrets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		os.Exit(2)
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

func addUser(clusterName string, info ConnInfo, newUser string) {
	var conn *sql.DB
	var err error

	conn, err = sql.Open("postgres", "sslmode=disable user="+info.Username+" host="+info.Hostip+" port="+info.Port+" dbname="+info.Database+" password="+info.Password)
	if err != nil {
		log.Debug(err.Error())
		os.Exit(2)
	}

	var rows *sql.Rows

	querystr := "create user " + newUser
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		os.Exit(2)
	}

	if UserDBAccess != "" {
		querystr = "grant all on database " + UserDBAccess + " to  " + newUser
	} else {
		querystr = "grant all on database userdb to  " + newUser
	}
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		os.Exit(2)
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
		err = util.CreateUserSecret(Clientset, clusterName, newUser, info.Password, Namespace)
		if err != nil {
		}
	}

}
func deleteUser(clusterName string, info ConnInfo, user string) {
	var conn *sql.DB
	var err error

	conn, err = sql.Open("postgres", "sslmode=disable user="+info.Username+" host="+info.Hostip+" port="+info.Port+" dbname="+info.Database+" password="+info.Password)
	if err != nil {
		log.Debug(err.Error())
		//os.Exit(2)
	}

	var rows *sql.Rows

	querystr := "drop owned by  " + user + " cascade"
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		//os.Exit(2)
	}
	querystr = "drop user if exists " + user
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		//os.Exit(2)
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()

	err = util.DeleteUserSecret(Clientset, clusterName, user, Namespace)
	if err != nil {
		log.Debug(err.Error())
		//os.Exit(2)
	}

}
