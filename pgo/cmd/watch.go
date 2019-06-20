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

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Print watch information for the PostgreSQL Operator",
	Long: `WATCH allows you to watch event information for the postgres-operator. For example:
				        pgo watch EventAll
				        pgo watch EventCluster EventPolicy`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}

		log.Debug("watch called")
		watch(args, Namespace)
	},
}

func init() {
	RootCmd.AddCommand(watchCmd)
}

func watch(args []string, ns string) {
	log.Debugf("watch called %v", args)

	r := msgs.WatchRequest{}
	r.Namespace = ns
	r.ClientVersion = msgs.PGO_VERSION
	//r.Topics = make([]string, 1)
	r.Topics = args

	response, err := api.Watch(httpclient, &r, &SessionCredentials)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Error {
		fmt.Println("Error: " + response.Status.Msg)
		return
	}

	if len(response.Results) == 0 {
		fmt.Println("No Watch Results")
		return
	}
	log.Debugf("response = %v", response)

	for _, v := range response.Results {
		fmt.Printf("%s%s\n", TreeTrunk, "/"+v)
	}

}
