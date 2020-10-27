package cmd

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"os"

	"github.com/crunchydata/postgres-operator/cmd/pgo/api"
	"github.com/crunchydata/postgres-operator/cmd/pgo/util"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Summary bool

func init() {
	RootCmd.AddCommand(statusCmd)

	statusCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", "The output format. Currently, json is the only supported value.")

}

func showStatus(args []string, ns string) {

	log.Debugf("showStatus called %v", args)

	if OutputFormat != "" && OutputFormat != "json" {
		fmt.Println("Error: json is the only supported --output-format value ")
		os.Exit(2)
	}

	response, err := api.ShowStatus(httpclient, &SessionCredentials, ns)

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
			fmt.Println("Error: ", err)
		}
		fmt.Println(string(b))
		return
	}

	printSummary(&response.Result)

}

func printSummary(status *msgs.StatusDetail) {

	WID := 25
	fmt.Printf("%s%d\n", util.Rpad("Databases:", " ", WID), status.NumDatabases)
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

	fmt.Printf("\n%s\n", "Labels (count > 1): [count] [label]")
	for i := 0; i < len(status.Labels); i++ {
		if status.Labels[i].Value > 1 {
			fmt.Printf("\t[%d]\t[%s]\n", status.Labels[i].Value, status.Labels[i].Key)
		}
	}
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display PostgreSQL cluster status",
	Long: `Display namespace wide information for PostgreSQL clusters.	For example:

	pgo status`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("status called")
		if Namespace == "" {
			Namespace = PGONamespace
		}
		showStatus(args, Namespace)
	},
}
