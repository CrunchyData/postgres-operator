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
	//"bytes"
	//"database/sql"
	//"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	//"github.com/crunchydata/postgres-operator/clusterservice"
	//"github.com/crunchydata/postgres-operator/tpr"
	//_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"log"
	"net/http"
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

	for _, arg := range args {
		fmt.Println(arg)
		url := "http://localhost:8080/clusters/test/newcluster"

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}
		fmt.Printf("%v\n", resp)

	}
}
