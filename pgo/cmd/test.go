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
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "test a Cluster",
	Long: `TEST allows you to test a new Cluster
				For example:

				pgo test mycluster
				.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("test called")
		if len(args) == 0 {
			fmt.Println(`You must specify the name of the clusters to test.`)
		} else {
			showTest(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(testCmd)
}

func showTest(args []string) {
	//get a list of all clusters
	clusterList := crv1.PgclusterList{}
	err := RestClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(Namespace).
		Do().Into(&clusterList)
	if err != nil {
		log.Error("error getting list of clusters" + err.Error())
		return
	}

	if len(clusterList.Items) == 0 {
		fmt.Println("no clusters found")
		return
	}

	itemFound := false

	//each arg represents a cluster name or the special 'all' value
	for _, arg := range args {
		for _, cluster := range clusterList.Items {
			fmt.Println("")
			if arg == "all" || cluster.Spec.Name == arg {
				itemFound = true
				fmt.Println("cluster : " + cluster.Spec.Name + " (" + cluster.Spec.CCP_IMAGE_TAG + ")")
				log.Debug("listing cluster " + arg)
				//list the services
				testServices(&cluster)
			}
		}
		if !itemFound {
			fmt.Println(arg + " was not found")
		}
		itemFound = false
	}
}

func testServices(cluster *crv1.Pgcluster) {

	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cluster.Spec.Name}
	services, err := Clientset.CoreV1().Services(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of services" + err.Error())
		return
	}

	lo = meta_v1.ListOptions{LabelSelector: "pg-database=" + cluster.Spec.Name}
	secrets, err := Clientset.Core().Secrets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return
	}

	for _, service := range services.Items {
		for _, s := range secrets.Items {
			username := string(s.Data["username"][:])
			password := string(s.Data["password"][:])
			database := "postgres"
			if username == cluster.Spec.PG_USER {
				database = cluster.Spec.PG_DATABASE
			}
			fmt.Print("psql -p " + cluster.Spec.Port + " -h " + service.Spec.ClusterIP + " -U " + username + " " + database)
			status := query(username, service.Spec.ClusterIP, cluster.Spec.Port, database, password)
			if status {
				fmt.Print(" is " + GREEN("working"))
			} else {
				fmt.Print(" is " + RED("NOT working"))
			}
			fmt.Println("")
		}
	}
}

func query(dbUser, dbHost, dbPort, database, dbPassword string) bool {
	var conn *sql.DB
	var err error

	conn, err = sql.Open("postgres", "sslmode=disable user="+dbUser+" host="+dbHost+" port="+dbPort+" dbname="+database+" password="+dbPassword)
	if err != nil {
		log.Debug(err.Error())
		return false
	}

	var ts string
	var rows *sql.Rows

	rows, err = conn.Query("select now()::text")
	if err != nil {
		log.Debug(err.Error())
		return false
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
		if err = rows.Scan(&ts); err != nil {
			log.Debug(err.Error())
			return false
		}
		log.Debug("returned " + ts)
		return true
	}
	return false

}
