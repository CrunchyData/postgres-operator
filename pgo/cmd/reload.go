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

//unused but coming soon to a theatre near you
var ConfigMapName string

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Perform a cluster reload",
	Long: `RELOAD performs a PostgreSQL reload on a cluster or set of clusters. For example:

	pgo reload mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("reload called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: You must specify the cluster to reload or specify a selector flag.`)
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				reload(args)
			} else {
				fmt.Println("Aborting...")
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(reloadCmd)

	reloadCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	reloadCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "No command line confirmation.")

}

// reload ....
func reload(args []string) {
	log.Debugf("reload called %v\n", args)

	request := new(msgs.ReloadRequest)
	request.Args = args
	request.Selector = Selector

	jsonValue, _ := json.Marshal(request)

	url := APIServerURL + "/reload"

	log.Debug("reload called [" + url + "]")

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

	var response msgs.ReloadResponse

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

	if len(response.Results) == 0 {
		fmt.Println("No clusters found.")
		return
	}

}
