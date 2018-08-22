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
	"github.com/spf13/cobra"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/labels"
	"net/http"
	"os"
)

var LoadConfig string

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Perform a data load",
	Long: `LOAD performs a load. For example:

	pgo load --load-config=./load.json --selector=project=xray`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("load called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: You must specify the cluster to load or a selector flag.`)
		} else {
			if LoadConfig == "" {
				fmt.Println("Error: You must specify the load-config.")
				return
			}

			createLoad(args)
		}

	},
}

func init() {
	RootCmd.AddCommand(loadCmd)

	loadCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	loadCmd.Flags().StringVarP(&LoadConfig, "load-config", "l", "", "The load configuration to use that defines the load job.")
	loadCmd.Flags().StringVarP(&PoliciesFlag, "policies", "z", "", "The policies to apply before loading a file, comma separated.")

}

func createLoad(args []string) {
	if PoliciesFlag != "" {
		log.Debug("policies=" + PoliciesFlag)
	} else {
		log.Debug("The --policies flag requires a value.")
	}
	if Selector != "" {
		//use the selector instead of an argument list to filter on

		_, err := labels.Parse(Selector)
		if err != nil {
			fmt.Println("Error: Could not parse selector flag.")
			return
		}
	}

	buf, err := ioutil.ReadFile(LoadConfig)
	request := msgs.LoadRequest{}
	request.LoadConfig = string(buf)
	request.Selector = Selector
	request.Policies = PoliciesFlag
	request.Args = args
	request.ClientVersion = msgs.PGO_VERSION

	//make the request

	jsonValue, _ := json.Marshal(request)

	url := APIServerURL + "/load"
	log.Debug("LoadPolicy called...[" + url + "]")

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

	var response msgs.LoadResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		log.Println(err)
		return
	}

	//get the response
	if response.Status.Code == msgs.Error {
		fmt.Println("Error: Error in loading...")
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	for value := range response.Results {
		fmt.Println(value)
	}

}
