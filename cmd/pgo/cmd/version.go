package cmd

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"os"

	"github.com/crunchydata/postgres-operator/cmd/pgo/api"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// ClientVersionOnly indicates that only the client version should be returned, not make
// a call to the server
var ClientVersionOnly bool

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
	versionCmd.Flags().BoolVar(&ClientVersionOnly, "client", false, "Only return the version of the pgo client. This does not make a call to the API server.")
}

func showVersion() {
	// print the client version
	fmt.Println("pgo client version " + msgs.PGO_VERSION)

	// if the user selects only to display the client version, return here
	if ClientVersionOnly {
		return
	}

	// otherwise, get the server version
	response, err := api.ShowVersion(httpclient, &SessionCredentials)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	fmt.Println("pgo-apiserver version " + response.Version)
}
