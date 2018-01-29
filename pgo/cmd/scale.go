package cmd

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

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/spf13/cobra"
	"net/http"
	"strconv"
)

var ReplicaCount int

var scaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale a Cluster",
	Long: `scale allows you to adjust a Cluster's replica configuration
For example:

pgo scale mycluster --replica-count=1
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("scale called")
		if ReplicaCount < 0 {
			fmt.Println(`--replica-count command line flag is required`)
		} else if len(args) == 0 {
			fmt.Println(`You must specify the clusters to scale.`)
		} else {
			scaleCluster(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(scaleCmd)

	scaleCmd.Flags().IntVarP(&ReplicaCount, "replica-count", "r", -1, "The replica count to apply to the clusters")

}

func scaleCluster(args []string) {

	for _, arg := range args {
		log.Debugf(" %s ReplicaCount is %d\n", arg, ReplicaCount)
		url := APIServerURL + "/clusters/scale/" + arg + "?replica-count=" + strconv.Itoa(ReplicaCount)
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

		defer resp.Body.Close()

		var response msgs.ClusterScaleResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if response.Status.Code == msgs.Ok {
			fmt.Println("Ok")
		} else {
			fmt.Println("Error")
			fmt.Println(response.Status.Msg)
		}

	}
}
