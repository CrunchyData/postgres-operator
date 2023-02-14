package cmd

/*
 Copyright 2020 - 2023 Crunchy Data Solutions, Inc.
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
	"strings"

	"github.com/crunchydata/postgres-operator/cmd/pgo/api"
	"github.com/crunchydata/postgres-operator/cmd/pgo/util"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
)

// showPgAdminTextPadding contains the values for what the text padding should be
type showPgAdminTextPadding struct {
	ClusterName int
	ClusterIP   int
	ExternalIP  int
	ServiceName int
}

func createPgAdmin(args []string, ns string) {
	if Selector == "" && len(args) == 0 {
		fmt.Println("Error: The --selector flag is required when cluster is unspecified.")
		os.Exit(1)
	}

	request := msgs.CreatePgAdminRequest{
		Args:          args,
		ClientVersion: msgs.PGO_VERSION,
		Namespace:     ns,
		Selector:      Selector,
		StorageConfig: PGAdminStorageConfig,
		PVCSize:       PGAdminPVCSize,
	}

	response, err := api.CreatePgAdmin(httpclient, &SessionCredentials, &request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// this is slightly rewritten from the legacy method
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		for _, v := range response.Results {
			fmt.Println(v)
		}
		os.Exit(1)
	}

	for _, v := range response.Results {
		fmt.Println(v)
	}
}

func deletePgAdmin(args []string, ns string) {
	if Selector == "" && len(args) == 0 {
		fmt.Println("Error: The --selector flag or a cluster name is required.")
		os.Exit(1)
	}

	// set up the API request
	request := msgs.DeletePgAdminRequest{
		Args:          args,
		ClientVersion: msgs.PGO_VERSION,
		Selector:      Selector,
		Namespace:     ns,
	}

	response, err := api.DeletePgAdmin(httpclient, &SessionCredentials, &request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	if response.Status.Code == msgs.Ok {
		for _, v := range response.Results {
			fmt.Println(v)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}
}

// makeShowPgAdminInterface returns an interface slice of the available values
// in show pgadmin
func makeShowPgAdminInterface(values []msgs.ShowPgAdminDetail) []interface{} {
	// iterate through the list of values to make the interface
	showPgAdminInterface := make([]interface{}, len(values))

	for i, value := range values {
		showPgAdminInterface[i] = value
	}

	return showPgAdminInterface
}

// printShowPgAdminText prints out the information around each PostgreSQL
// cluster's pgAdmin
// printShowPgAdminText renders a text response
func printShowPgAdminText(response msgs.ShowPgAdminResponse) {
	// if the request errored, return the message here and exit with an error
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// if no results returned, return an error
	if len(response.Results) == 0 {
		fmt.Println("Nothing found.")
		return
	}

	// make the interface for the pgadmin clusters
	showPgAdminInterface := makeShowPgAdminInterface(response.Results)

	// format the header
	// start by setting up the different text paddings
	padding := showPgAdminTextPadding{
		ClusterName: getMaxLength(showPgAdminInterface, headingCluster, "ClusterName"),
		ClusterIP:   getMaxLength(showPgAdminInterface, headingClusterIP, "ServiceClusterIP"),
		ExternalIP:  getMaxLength(showPgAdminInterface, headingExternalIP, "ServiceExternalIP"),
		ServiceName: getMaxLength(showPgAdminInterface, headingService, "ServiceName"),
	}

	printShowPgAdminTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printShowPgAdminTextRow(result, padding)
	}
}

// printShowPgAdminTextHeader prints out the header
func printShowPgAdminTextHeader(padding showPgAdminTextPadding) {
	// print the header
	fmt.Println("")
	fmt.Printf("%s", util.Rpad(headingCluster, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(headingService, " ", padding.ServiceName))
	fmt.Printf("%s", util.Rpad(headingClusterIP, " ", padding.ClusterIP))
	fmt.Printf("%s", util.Rpad(headingExternalIP, " ", padding.ExternalIP))
	fmt.Println("")

	// print the layer below the header...which prints out a bunch of "-" that's
	// 1 less than the padding value
	fmt.Println(
		strings.Repeat("-", padding.ClusterName-1),
		strings.Repeat("-", padding.ServiceName-1),
		strings.Repeat("-", padding.ClusterIP-1),
		strings.Repeat("-", padding.ExternalIP-1),
	)
}

// printShowPgAdminTextRow prints a row of the text data
func printShowPgAdminTextRow(result msgs.ShowPgAdminDetail, padding showPgAdminTextPadding) {
	fmt.Printf("%s", util.Rpad(result.ClusterName, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(result.ServiceName, " ", padding.ServiceName))
	fmt.Printf("%s", util.Rpad(result.ServiceClusterIP, " ", padding.ClusterIP))
	fmt.Printf("%s", util.Rpad(result.ServiceExternalIP, " ", padding.ExternalIP))
	fmt.Println("")
}

// showPgAdmin prepares to make an API requests to display information about
// one or more pgAdmin deployments. "clusterNames" is an array of cluster
// names to iterate over
func showPgAdmin(namespace string, clusterNames []string) {
	// first, determine if any arguments have been pass in
	if len(clusterNames) == 0 && Selector == "" {
		fmt.Println("Error: You must provide at least one cluster name, or use a selector with the `--selector` flag")
		os.Exit(1)
	}

	request := msgs.ShowPgAdminRequest{
		ClusterNames: clusterNames,
		Namespace:    namespace,
		Selector:     Selector,
	}

	response, err := api.ShowPgAdmin(httpclient, &SessionCredentials, request)
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	// great! now we can work on interpreting the results and outputting them
	// per the user's desired output format
	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		fmt.Println("outputting in json")
		printJSON(response)
	default:
		fmt.Println("outputting text")
		printShowPgAdminText(response)
	}
}
