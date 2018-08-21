// Package cmd provides the command line functions of the crunchy CLI
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
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/util"
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

var failoverCmd = &cobra.Command{
	Use:   "failover",
	Short: "Performs a manual failover",
	Long: `Performs a manual failover. For example:

	pgo failover mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("failover called")
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the cluster to failover.`)
		} else {
			if Query {
				queryFailover(args)
			} else if util.AskForConfirmation(NoPrompt, "") {
				if Target == "" {
					fmt.Println(`Error: The --target flag is required for failover.`)
					return
				}
				createFailover(args)
			} else {
				fmt.Println("Aborting...")
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(failoverCmd)

	failoverCmd.Flags().BoolVarP(&Query, "query", "", false, "Prints the list of failover candidates.")
	failoverCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "No command line confirmation.")
	failoverCmd.Flags().StringVarP(&Target, "target", "", "", "The replica target which the failover will occur on.")

}

// createFailover ....
func createFailover(args []string) {
	log.Debugf("createFailover called %v\n", args)

	request := new(msgs.CreateFailoverRequest)
	request.ClusterName = args[0]
	request.Target = Target
	request.ClientVersion = msgs.PGO_VERSION

	jsonValue, _ := json.Marshal(request)

	url := APIServerURL + "/failover"

	log.Debug("create failover called [" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Println("Error: NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		fmt.Println("Error: Do: ", err)
		return
	}
	log.Debugf("%v\n", resp)
	StatusCheck(resp)

	defer resp.Body.Close()

	var response msgs.CreateFailoverResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		log.Println(err)
		return
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

// queryFailover ....
func queryFailover(args []string) {
	log.Debugf("queryFailover called %v\n", args)

	url := APIServerURL + "/failover/" + args[0] + "?version=" + msgs.PGO_VERSION

	log.Debug("query failover called [" + url + "]")

	action := "GET"
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		fmt.Println("Error: NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		fmt.Println("Error: Do: ", err)
		return
	}
	StatusCheck(resp)

	defer resp.Body.Close()

	var response msgs.QueryFailoverResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		return
	}

	if response.Status.Code == msgs.Ok {
		fmt.Println(response.Status.Msg)
		if len(response.Targets) > 0 {
			fmt.Println("Failover targets include:")
			for i := 0; i < len(response.Targets); i++ {
				printTarget(response.Targets[i])
			}
		}
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

func printTarget(target msgs.FailoverTargetSpec) {
	fmt.Printf("\t%s (%s) (%s) ReceiveLoc (%d) ReplayLoc (%d)\n", target.Name, target.ReadyStatus, target.Node, target.ReceiveLocation, target.ReplayLocation)
}
