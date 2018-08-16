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

var backRestCmd = &cobra.Command{
	Use:   "backrest",
	Short: "perform a pgbackrest action",
	Long: `BACKREST performs a pgbackrest action, for example:
		pgo backrest mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("backup called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`You must specify the cluster to perform an action on or a cluster selector flag.`)
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				createBackrestBackup(args)
			} else {
				fmt.Println("Aborting...")
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(backRestCmd)

	backRestCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	backRestCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "--no-prompt causes there to be no command line confirmation when doing a pgbackrest command")

}

// createBackrestBackup ....
func createBackrestBackup(args []string) {
	log.Debugf("createBackrestBackup called %v\n", args)

	request := new(msgs.CreateBackrestBackupRequest)
	request.Args = args
	request.Selector = Selector

	jsonValue, _ := json.Marshal(request)

	url := APIServerURL + "/backrestbackup"

	log.Debug("create backrest backup called [" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
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

	var response msgs.CreateBackrestBackupResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Error(err)
		log.Println(err)
		return
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println(RED(response.Status.Msg))
		os.Exit(2)
	}

	if len(response.Results) == 0 {
		fmt.Println("no clusters found")
		return
	}

}

// showBackrest ....
func showBackrest(args []string) {
	log.Debugf("showBackrest called %v\n", args)

	for _, v := range args {
		url := APIServerURL + "/backrest/" + v + "?version=" + msgs.PGO_VERSION + "&selector=" + Selector

		log.Debug("show backrest called [" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}
		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)
		StatusCheck(resp)

		defer resp.Body.Close()

		var response msgs.ShowBackrestResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if response.Status.Code != msgs.Ok {
			log.Error(RED(response.Status.Msg))
			os.Exit(2)
		}

		if len(response.Items) == 0 {
			fmt.Println("no backrest found")
			return
		}

		log.Debugf("response = %v\n", response)
		log.Debugf("len of items = %d\n", len(response.Items))

		for _, backup := range response.Items {
			printBackrest(&backup)
		}

	}

}

// printBackrest
func printBackrest(result *msgs.ShowBackrestDetail) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "backrest : "+result.Name)
	fmt.Printf("%s%s\n", "", result.Info)

}
