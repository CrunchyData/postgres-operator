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
	"os"
)

const CAPMAX = 50

var dfCmd = &cobra.Command{
	Use:   "df",
	Short: "	Long: `Display disk status of Clusters",
	Long: `Display the disk status of Clusters using the df command. For example:

	pgo df mycluster
	pgo df --selector=env=research`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("df called")
		if Selector == "" && len(args) == 0 {
			fmt.Println(`You must specify the name of the clusters to test.`)
		} else {
			showDf(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(dfCmd)
	dfCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
}

func showDf(args []string) {

	log.Debugf("showDf called %v\n", args)

	log.Debug("The selector is " + Selector)
	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	for _, arg := range args {
		url := APIServerURL + "/df/" + arg + "?selector=" + Selector + "&version=" + ClientVersion
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

		var response msgs.DfResponse

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

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Println("error:", err)
			}
			fmt.Println(string(b))
			return
		}

		if len(response.Results) == 0 {
			fmt.Println("Nothing found.")
			return
		}

		fmt.Printf("%s", util.Rpad("CLUSTER", " ", 20))
		fmt.Printf("%s", util.Rpad("STATUS", " ", 10))
		fmt.Printf("%s", util.Rpad("PGSIZE", " ", 10))
		fmt.Printf("%s", util.Rpad("CAPACITY", " ", 10))
		fmt.Printf("%s\n", util.Rpad("PCTUSED", " ", 10))
		fmt.Println("")

		for _, result := range response.Results {
			fmt.Printf("%s", util.Rpad(result.Name, " ", 20))
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
