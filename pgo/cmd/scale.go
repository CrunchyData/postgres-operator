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
	"github.com/crunchydata/postgres-operator/pgo/util"
	"github.com/spf13/cobra"
	"net/http"
	"strconv"
	"os"
)

var ReplicaCount int

var scaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale a Cluster",
	Long: `scale allows you to adjust a Cluster's replica configuration
For example:

pgo scale mycluster --replica-count=1

to list replicas:
pgo scale mycluster --query

to scale down:
pgo scale mycluster --replica-count=-1

to scale down a specific replica:
pgo scale mycluster --replica-count=-1 --target=mycluster-replica-xxxx
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("scale called")

		if len(args) == 0 {
			fmt.Println(`You must specify the clusters to scale.`)
		} else {
			if Query {
				queryCluster(args)
			} else {
 		if util.AskForConfirmation(NoPrompt, "") {
                } else {
                        fmt.Println("Aborting...")
                        os.Exit(2)
                }
			scaleCluster(args)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(scaleCmd)

	scaleCmd.Flags().IntVarP(&ReplicaCount, "replica-count", "r", 1, "The replica count to apply to the clusters, defaults to 1")
	scaleCmd.Flags().StringVarP(&ContainerResources, "resources-config", "", "", "The name of a container resource configuration in pgo.yaml that holds CPU and memory requests and limits")
	scaleCmd.Flags().StringVarP(&StorageConfig, "storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the replica storage.")
	scaleCmd.Flags().StringVarP(&NodeLabel, "node-label", "", "", "The node label (key=value) to use in placing the replica pod, if not set any node is used")
	scaleCmd.Flags().StringVarP(&ServiceType, "service-type", "", "", "The service type to use in the replica Service, if not set the default in pgo.yaml will be used")
	scaleCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCPImageTag to use for cluster creation, if specified overrides the .pgo.yaml setting")
	scaleCmd.Flags().BoolVarP(&Query, "query", "", false, "--query prints the list of replica candidates")
	scaleCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "--no-prompt causes there to be no command line confirmation when doing a scale command")
	scaleCmd.Flags().StringVarP(&Target, "target", "", "", "--target is the replica target which the scale will occur on. Only applies when --replica-cound=-1")

}

func scaleCluster(args []string) {

	var url string
	for _, arg := range args {
		log.Debugf(" %s ReplicaCount is %d\n", arg, ReplicaCount)
		url = APIServerURL + "/clusters/scale/" + arg + "?replica-count=" + strconv.Itoa(ReplicaCount) + "&resources-config=" + ContainerResources + "&storage-config=" + StorageConfig + "&node-label=" + NodeLabel + "&version=" + msgs.PGO_VERSION + "&ccp-image-tag=" + CCPImageTag + "&service-type=" + ServiceType
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
		StatusCheck(resp)

		defer resp.Body.Close()

		var response msgs.ClusterScaleResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if response.Status.Code == msgs.Ok {
			for _, v := range response.Results {
				fmt.Println(v)
			}
		} else {
			log.Error(RED(response.Status.Msg))
		}

	}
}

func queryCluster(args []string) {

	var url string
	for _, arg := range args {
			url = APIServerURL + "/scale/" + arg + "?version=" + msgs.PGO_VERSION
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
		StatusCheck(resp)

		defer resp.Body.Close()

		var response msgs.ScaleQueryResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if response.Status.Code == msgs.Ok {
 if len(response.Targets) > 0 {
                        fmt.Println("Replica targets include:")
                        for i := 0; i < len(response.Targets); i++ {
                                fmt.Println("\t" + response.Targets[i].Name + " (" + response.Targets[i].ReadyStatus + ") (" + response.Targets[i].Node + ")")
                        }
                }

			for _, v := range response.Results {
				fmt.Println(v)
			}
		} else {
			log.Error(RED(response.Status.Msg))
		}

	}
}
