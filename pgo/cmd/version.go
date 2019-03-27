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
	"fmt"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

	response, err := api.ShowVersion(httpclient, &SessionCredentials)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	fmt.Println("pgo client version " + msgs.PGO_VERSION)

	if response.Status.Code == msgs.Ok {
		fmt.Println("pgo-apiserver version " + response.Version)
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}
