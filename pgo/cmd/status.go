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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display PostgreSQL cluster status",
	Long: `Display namespace wide information for PostgreSQL clusters.	For example:

	pgo status`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("status called")
		showStatus(args)
	},
}

var Summary bool

func init() {
	RootCmd.AddCommand(statusCmd)
	
	statusCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", "The output format. Currently, JSON is supported.")

}

func showStatus(args []string) {

	log.Debugf("showStatus called %v\n", args)

	url := APIServerURL + "/status?version=" + msgs.PGO_VERSION
	log.Debug(url)

	req, err := http.NewRequest("GET", url, nil)
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

	var response msgs.StatusResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		log.Println(err)
		return
	}

	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if OutputFormat == "json" {
		b, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println(string(b))
		return
	}

	printSummary(&response.Result)

}

func printSummary(status *msgs.StatusDetail) {

	WID := 25
	fmt.Printf("%s%s\n", util.Rpad("Operator Start:", " ", WID), status.OperatorStartTime)
	fmt.Printf("%s%d\n", util.Rpad("Databases:", " ", WID), status.NumDatabases)
	fmt.Printf("%s%d\n", util.Rpad("Backups:", " ", WID), status.NumBackups)
	fmt.Printf("%s%d\n", util.Rpad("Claims:", " ", WID), status.NumClaims)
	fmt.Printf("%s%s\n", util.Rpad("Total Volume Size:", " ", WID), util.Rpad(status.VolumeCap, " ", 10))

	fmt.Printf("\n%s\n", "Database Images:")
	for k, v := range status.DbTags {
		fmt.Printf("%s%d\t%s\n", util.Rpad(" ", " ", WID), v, k)
	}

	fmt.Printf("\n%s\n", "Databases Not Ready:")
	for i := 0; i < len(status.NotReady); i++ {
		fmt.Printf("\t%s\n", util.Rpad(status.NotReady[i], " ", 30))
	}

	fmt.Printf("\n%s\n", "Nodes:")
	for i := 0; i < len(status.Nodes); i++ {
		fmt.Printf("\t%s\n", util.Rpad(status.Nodes[i].Name, " ", 30))
		fmt.Printf("\t\tStatus:%s\n", util.Rpad(status.Nodes[i].Status, " ", 30))
		fmt.Println("\t\tLabels:")
		for k, v := range status.Nodes[i].Labels {
			fmt.Printf("\t\t\t%s=%s\n", k, v)
		}
	}
	fmt.Printf("\n%s\n", "Labels (count > 1): [count] [label]")
	for i := 0; i < len(status.Labels); i++ {
		if status.Labels[i].Value > 1 {
			fmt.Printf("\t[%d]\t[%s]\n", status.Labels[i].Value, status.Labels[i].Key)
		}
	}
}
