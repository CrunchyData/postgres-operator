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
	_ "github.com/lib/pq"
	"strconv"
	"time"
	//"k8s.io/apimachinery/pkg/labels"
	"github.com/crunchydata/postgres-operator/operator/util"
	//"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
	//"io/ioutil"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"os/user"
	//"strings"
)

type ConnInfo struct {
	Username string
	Hostip   string
	StrPort  string
	Database string
	Password string
}
type PswResult struct {
	Rolname       string
	Rolvaliduntil string
	ConnDetails   ConnInfo
}

var Expired string
var UpdatePasswords bool

var pswCmd = &cobra.Command{
	Use:   "psw",
	Short: "manage passwords",
	Long: `PSW allows you to manage passwords across a set of clusters
For example:

pgo psw --selector=name=mycluster --update
pgo psw --expired=7 --selector=someotherpolicy
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("psw called")
		passwordManager()
	},
}

func init() {
	RootCmd.AddCommand(pswCmd)

	pswCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	pswCmd.Flags().StringVarP(&Expired, "expired", "e", "", "--expired=7 shows passwords that will expired in 7 days")
	pswCmd.Flags().BoolVarP(&UpdatePasswords, "update-passwords", "u", false, "--update-passwords performs password updating on expired passwords")

}

func passwordManager() {
	//build the selector based on the selector parameter
	//get the clusters list

	//get filtered list of Deployments
	var sel string
	if Selector != "" {
		sel = Selector + ",pg-cluster,!replica"
	} else {
		sel = "pg-cluster,!replica"
	}
	log.Infoln("selector string=[" + sel + "]")

	lo := meta_v1.ListOptions{LabelSelector: sel}
	deployments, err := Clientset.ExtensionsV1beta1().Deployments(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return
	}

	for _, d := range deployments.Items {
		if Expired != "" {
			results := callDB(d.ObjectMeta.Name, Expired)
			if len(results) > 0 {
				fmt.Println("deployment : " + d.ObjectMeta.Name)
				fmt.Println("expired passwords....")
				for _, v := range results {
					fmt.Printf("RoleName %s Role Valid Until %s\n", v.Rolname, v.Rolvaliduntil)
					if UpdatePasswords {
						newPassword := util.GeneratePassword(8)
						newExpireDate := GeneratePasswordExpireDate(60)
						err = updatePassword(v, newPassword, newExpireDate)
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

func callDB(clusterName, maxdays string) []PswResult {
	var conn *sql.DB

	results := []PswResult{}

	//get the service for the cluster
	service, err := Clientset.CoreV1().Services(Namespace).Get(clusterName, meta_v1.GetOptions{})
	if err != nil {
		log.Error("error getting list of services" + err.Error())
		return results
	}

	//get the secrets for this cluster
	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + clusterName}
	secrets, err := Clientset.Secrets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return results
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
	conn, err = sql.Open("postgres", "sslmode=disable user="+username+" host="+hostip+" port="+strPort+" dbname="+database+" password="+password)
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
		c := ConnInfo{Username: username, Hostip: hostip, StrPort: strPort, Database: database, Password: password}
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

func updatePassword(p PswResult, newPassword, passwordExpireDate string) error {
	var err error
	var conn *sql.DB

	conn, err = sql.Open("postgres", "sslmode=disable user="+p.ConnDetails.Username+" host="+p.ConnDetails.Hostip+" port="+p.ConnDetails.StrPort+" dbname="+p.ConnDetails.Database+" password="+p.ConnDetails.Password)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	//var ts string
	var rows *sql.Rows

	querystr := "ALTER user " + p.Rolname + " PASSWORD '" + newPassword + "'"
	log.Debug(querystr)
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		return err
	}
	querystr = "ALTER user " + p.Rolname + " VALID UNTIL '" + passwordExpireDate + "'"
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

func GeneratePasswordExpireDate(daysFromNow int) string {

	now := time.Now()
	totalHours := daysFromNow * 24
	diffDays, _ := time.ParseDuration(strconv.Itoa(totalHours) + "h")
	futureTime := now.Add(diffDays)
	return futureTime.Format("2006-01-02")

}
