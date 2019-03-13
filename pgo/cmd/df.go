package cmd

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

const CAPMAX = 50

var dfCmd = &cobra.Command{
	Use:   "df",
	Short: "Display disk space for clusters",
	Long: `Displays the disk status for PostgreSQL clusters. For example:

	pgo df mycluster
	pgo df all
	pgo df --selector=env=research`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("df called")
		if Selector == "" && len(args) == 0 {
			fmt.Println(`Error: You must specify the name of the clusters to test.`)
		} else {
			showDf(args, Namespace)
		}
	},
}

func init() {
	RootCmd.AddCommand(dfCmd)

	dfCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")

}

func showDf(args []string, ns string) {

	log.Debugf("showDf called %v", args)

	log.Debugf("selector is %s", Selector)
	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	for _, arg := range args {
		response, err := api.ShowDf(httpclient, arg, Selector, &SessionCredentials, ns)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Println("Error:", err)
			}
			fmt.Println(string(b))
			return
		}

		if len(response.Results) == 0 {
			fmt.Println("Nothing found.")
			return
		}

		var maxLen int
		for _, result := range response.Results {
			tmp := len(result.Name)
			if tmp > maxLen {
				maxLen = tmp
			}
		}
		maxLen++

		fmt.Printf("%s", util.Rpad("POD", " ", maxLen))
		fmt.Printf("%s", util.Rpad("STATUS", " ", 10))
		fmt.Printf("%s", util.Rpad("PGSIZE", " ", 10))
		fmt.Printf("%s", util.Rpad("CAPACITY", " ", 10))
		fmt.Printf("%s\n", util.Rpad("PCTUSED", " ", 10))
		fmt.Println("")

		for _, result := range response.Results {
			fmt.Printf("%s", util.Rpad(result.Name, " ", maxLen))
			if result.Working {
				fmt.Printf("%s", GREEN(util.Rpad("up", " ", 10)))
			} else {
				fmt.Printf("%s", RED(util.Rpad("down", " ", 10)))
			}
			fmt.Printf("%s", util.Rpad(result.PGSize, " ", 10))
			fmt.Printf("%s", util.Rpad(result.ClaimSize, " ", 10))
			if result.Pct > CAPMAX {
				fmt.Printf("%s\n", RED(fmt.Sprintf("%d", result.Pct)))
			} else {
				fmt.Printf("%s\n", GREEN(fmt.Sprintf("%d", result.Pct)))
			}
		}

	}
}
