package cmd

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
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/spf13/cobra"
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
		url := APIServerURL + "/clusters/test/" + arg
		log.Debug(url)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)

		defer resp.Body.Close()

		var response msgs.ClusterTestResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if len(response.Items) == 0 {
			fmt.Println("nothing found")
			return
		}

		for _, v := range response.Items {
			fmt.Printf("%s is ", v.PsqlString)
			if v.Working {
				fmt.Printf("%s\n", GREEN("working"))
			} else {
				fmt.Printf("%s\n", RED("NOT working"))
			}
		}

	}
}
