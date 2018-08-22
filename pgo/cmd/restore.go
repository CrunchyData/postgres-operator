// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2018 Crunchy Data Solutions, Inc.
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
	//"github.com/crunchydata/postgres-operator/util"
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

var ToCluster string
var RestoreOpts string
var PITRTarget string

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Perform a pgBackRest restore",
	Long: `RESTORE performs a pgBackRest restore to a new PostgreSQL cluster. For example:

	pgo restore withbr --to-cluster=restored
	pgo create cluster restored --custom-config=backrest-restore-withbr-to-restored --secret-from=withbr --pgbackrest`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("restore called")
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the cluster name to restore from.`)
		} else {
			if ToCluster == "" {
				fmt.Println("Error: You must specify the --to-cluster flag.")
				os.Exit(2)
			}
			/**
			if RestoreOpts == "" {
				fmt.Println("Error: You must specify the --restore-opts flag.")
				os.Exit(2)
			}
			*/
			restore(args)
		}

	},
}

func init() {
	RootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVarP(&ToCluster, "to-cluster", "", "", "The name of the new cluster to restore to.")
	restoreCmd.Flags().StringVarP(&RestoreOpts, "restore-opts", "", "", "The options for the restore.")
	restoreCmd.Flags().StringVarP(&PITRTarget, "pitr-target", "", "", "The PITR target, being a PostgreSQL timestamp such as '2018-08-13 11:25:42.582117-04'.")

}

// restore ....
func restore(args []string) {
	log.Debugf("restore called %v\n", args)

	request := new(msgs.RestoreRequest)
	request.FromCluster = args[0]
	request.ToCluster = ToCluster
	request.RestoreOpts = RestoreOpts

	jsonValue, _ := json.Marshal(request)

	url := APIServerURL + "/restore"

	log.Debug("restore called [" + url + "]")

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

	var response msgs.RestoreResponse

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
