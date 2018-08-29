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
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	"github.com/spf13/cobra"
	"os"
)

var backRestCmd = &cobra.Command{
	Use:   "backrest",
	Short: "Perform a pgBackRest action",
	Long: `BACKREST performs a pgBackRest action. For example:

	pgo backrest mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("backup called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: You must specify the cluster to perform an action on or a cluster selector flag.`)
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

	backRestCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	backRestCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "No command line confirmation.")

}

// createBackrestBackup ....
func createBackrestBackup(args []string) {
	log.Debugf("createBackrestBackup called %v\n", args)

	request := new(msgs.CreateBackrestBackupRequest)
	request.Args = args
	request.Selector = Selector

	response, err := api.CreateBackrestBackup(httpclient, APIServerURL, BasicAuthUsername, BasicAuthPassword, request)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(2)
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

// showBackrest ....
func showBackrest(args []string) {
	log.Debugf("showBackrest called %v\n", args)

	for _, v := range args {
		response, err := api.ShowBackrest(httpclient, APIServerURL, v, Selector, BasicAuthUsername, BasicAuthPassword)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

		if len(response.Items) == 0 {
			fmt.Println("No pgBackRest found.")
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
