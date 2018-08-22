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
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information for the PostgreSQL Operator",
	Long: `VERSION allows you to print version information for the postgres-operator. For example:

	pgo version`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("version called")
		showVersion()
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

func showVersion() {

	url := APIServerURL + "/version"
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

	defer resp.Body.Close()

	StatusCheck(resp)

	var response msgs.VersionResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		log.Println(err)
		return
	}

	fmt.Println("pgo client version " + msgs.PGO_VERSION)

	if response.Status.Code == msgs.Ok {
		fmt.Println("apiserver version " + response.Version)
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}
