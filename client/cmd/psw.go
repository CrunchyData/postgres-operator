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
	//"k8s.io/apimachinery/pkg/labels"
	//"github.com/crunchydata/postgres-operator/operator/util"
	//"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
	//"io/ioutil"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"os/user"
	//"strings"
)

type PswResult struct {
	Rolname       string
	Rolvaliduntil string
}

var pswCmd = &cobra.Command{
	Use:   "psw",
	Short: "manage passwords",
	Long: `PSW allows you to manage passwords across a set of clusters
For example:

pgo psw --selector=name=mycluster --update
pgo psw --dry-run --selector=someotherpolicy
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("psw called")
		updatePasswords()
	},
}

func init() {
	RootCmd.AddCommand(pswCmd)

	pswCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	pswCmd.Flags().BoolVarP(&DryRun, "dry-run", "d", false, "--dry-run shows clusters and passwords that would be updated to but does not actually apply them")

}

func updatePasswords() {
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

	if DryRun {
		fmt.Println("dry run only....")
	}

	for _, d := range deployments.Items {
		fmt.Println("deployment : " + d.ObjectMeta.Name)
		getExpiredInfo(d.ObjectMeta.Name, "7")
		if !DryRun {
		}

	}

}

func getExpiredInfo(clusterName, MAX_DAYS string) {
	//var err error

	results := callDB(clusterName, MAX_DAYS)
	for _, v := range results {
		fmt.Printf("RoleName %s Role Valid Until %s\n", v.Rolname, v.Rolvaliduntil)
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
		if err = rows.Scan(&p.Rolname, &p.Rolvaliduntil); err != nil {
			log.Debug(err.Error())
			return results
		}
		results = append(results, p)
		log.Debug("returned " + ts)
	}

	return results

}
