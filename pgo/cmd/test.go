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
	"encoding/json"
	"fmt"
	"os"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test cluster connectivity",
	Long: `TEST allows you to test the availability of a PostgreSQL cluster. For example:

	pgo test mycluster
	pgo test --selector=env=research
	pgo test --all`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("test called")
		if Selector == "" && len(args) == 0 && !AllFlag {
			fmt.Println(`Error: You must specify the name of the clusters to test or --all or a --selector.`)
		} else {
			if OutputFormat != "" && OutputFormat != "json" {
				fmt.Println("Error: Only 'json' is currently supported for the --output flag value.")
				os.Exit(2)
			}
			showTest(args, Namespace)
		}
	},
}

func init() {
	RootCmd.AddCommand(testCmd)
	testCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	testCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", "The output format. Currently, json is the only supported value.")
	testCmd.Flags().BoolVar(&AllFlag, "all", false, "test all resources.")

}

func showTest(args []string, ns string) {

	log.Debugf("showCluster called %v", args)

	log.Debugf("selector is %s", Selector)

	if len(args) == 0 && !AllFlag && Selector == "" {
		fmt.Println("Error: ", "--all needs to be set or a cluster name be entered or a --selector be specified")
		os.Exit(2)
	}
	if Selector != "" || AllFlag {
		args = make([]string, 1)
		args[0] = "all"
	}

	r := new(msgs.ClusterTestRequest)
	r.Selector = Selector
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION

	for _, arg := range args {
		r.Clustername = arg
		response, err := api.ShowTest(httpclient, &SessionCredentials, r)
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

		if len(response.Results) == 0 {
			fmt.Println("Nothing found.")
			return
		}

		for _, result := range response.Results {
			fmt.Println("")
			fmt.Println(fmt.Sprintf("cluster : %s", result.ClusterName))

			// first, print the test results for the endpoints, which make up
			// the services
			printTestResults("Services", result.Endpoints)
			// first, print the test results for the instances
			printTestResults("Instances", result.Instances)
		}
	}
}

// prints out a set of test results
func printTestResults(testName string, results []msgs.ClusterTestDetail) {
	// print out the header for this group of tests
	fmt.Println(fmt.Sprintf("%s%s", TreeBranch, testName))
	// iterate though the results and print them!
	for _, v := range results {
		fmt.Printf("%s%s%s (%s): ",
			TreeBranch, TreeBranch, v.InstanceType, v.Message)
		if v.Available {
			fmt.Println(fmt.Sprintf("%s", GREEN("UP")))
		} else {
			fmt.Println(fmt.Sprintf("%s", RED("DOWN")))
		}
	}
}
