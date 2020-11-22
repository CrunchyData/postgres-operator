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
	"fmt"
	"os"
	"strings"

	"github.com/crunchydata/postgres-operator/cmd/pgo/api"
	"github.com/crunchydata/postgres-operator/cmd/pgo/util"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
)

// showPgBouncerTextPadding contains the values for what the text padding should be
type showPgBouncerTextPadding struct {
	ClusterName int
	ClusterIP   int
	ExternalIP  int
	Password    int
	ServiceName int
	Username    int
}

// updatePgBouncerTextPadding contains the values for what the text padding should be
type updatePgBouncerTextPadding struct {
	ClusterName  int
	ErrorMessage int
	Status       int
}

// PgBouncerReplicas is the total number of replica pods to deploy with a
// pgBouncer Deployment
var PgBouncerReplicas int32

// PgBouncerUninstall is used to ensure the objects intalled in PostgreSQL on
// behalf of pgbouncer are either not applied (in the case of a cluster create)
// or are removed (in the case of a pgo delete pgbouncer)
var PgBouncerUninstall bool

func createPgbouncer(args []string, ns string) {

	if Selector == "" && len(args) == 0 {
		fmt.Println("Error: The --selector flag is required.")
		return
	}

	request := msgs.CreatePgbouncerRequest{
		Args:          args,
		ClientVersion: msgs.PGO_VERSION,
		CPURequest:    PgBouncerCPURequest,
		CPULimit:      PgBouncerCPULimit,
		MemoryRequest: PgBouncerMemoryRequest,
		MemoryLimit:   PgBouncerMemoryLimit,
		Namespace:     ns,
		Replicas:      PgBouncerReplicas,
		Selector:      Selector,
		TLSSecret:     PgBouncerTLSSecret,
	}

	if err := util.ValidateQuantity(request.CPURequest, "cpu"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(request.CPULimit, "cpu-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(request.MemoryRequest, "memory"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(request.MemoryLimit, "memory-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	response, err := api.CreatePgbouncer(httpclient, &SessionCredentials, &request)

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

func deletePgbouncer(args []string, ns string) {

	if Selector == "" && len(args) == 0 {
		fmt.Println("Error: The --selector flag or a cluster name is required.")
		return
	}

	// set up the API request
	request := msgs.DeletePgbouncerRequest{
		Args:          args,
		ClientVersion: msgs.PGO_VERSION,
		Selector:      Selector,
		Namespace:     ns,
		Uninstall:     PgBouncerUninstall,
	}

	response, err := api.DeletePgbouncer(httpclient, &SessionCredentials, &request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for _, v := range response.Results {
			fmt.Println(v)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

// makeShowPgBouncerInterface returns an interface slice of the available values
// in show pgbouncer
func makeShowPgBouncerInterface(values []msgs.ShowPgBouncerDetail) []interface{} {
	// iterate through the list of values to make the interface
	showPgBouncerInterface := make([]interface{}, len(values))

	for i, value := range values {
		showPgBouncerInterface[i] = value
	}

	return showPgBouncerInterface
}

// makeUpdatePgBouncerInterface returns an interface slice of the available values
// in show pgbouncer
func makeUpdatePgBouncerInterface(values []msgs.UpdatePgBouncerDetail) []interface{} {
	// iterate through the list of values to make the interface
	updatePgBouncerInterface := make([]interface{}, len(values))

	for i, value := range values {
		updatePgBouncerInterface[i] = value
	}

	return updatePgBouncerInterface
}

// printShowPgBouncerText prints out the information around each PostgreSQL
// cluster's pgBouncer
// printShowPgBouncerText renders a text response
func printShowPgBouncerText(response msgs.ShowPgBouncerResponse) {
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

	// make the interface for the pgbouncer clusters
	showPgBouncerInterface := makeShowPgBouncerInterface(response.Results)

	// format the header
	// start by setting up the different text paddings
	padding := showPgBouncerTextPadding{
		ClusterName: getMaxLength(showPgBouncerInterface, headingCluster, "ClusterName"),
		ClusterIP:   getMaxLength(showPgBouncerInterface, headingClusterIP, "ServiceClusterIP"),
		ExternalIP:  getMaxLength(showPgBouncerInterface, headingExternalIP, "ServiceExternalIP"),
		ServiceName: getMaxLength(showPgBouncerInterface, headingService, "ServiceName"),
		Password:    getMaxLength(showPgBouncerInterface, headingPassword, "Password"),
		Username:    getMaxLength(showPgBouncerInterface, headingUsername, "Username"),
	}

	printShowPgBouncerTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printShowPgBouncerTextRow(result, padding)
	}
}

// printShowPgBouncerTextHeader prints out the header
func printShowPgBouncerTextHeader(padding showPgBouncerTextPadding) {
	// print the header
	fmt.Println("")
	fmt.Printf("%s", util.Rpad(headingCluster, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(headingService, " ", padding.ServiceName))
	fmt.Printf("%s", util.Rpad(headingUsername, " ", padding.Username))
	fmt.Printf("%s", util.Rpad(headingPassword, " ", padding.Password))
	fmt.Printf("%s", util.Rpad(headingClusterIP, " ", padding.ClusterIP))
	fmt.Printf("%s", util.Rpad(headingExternalIP, " ", padding.ExternalIP))
	fmt.Println("")

	// print the layer below the header...which prints out a bunch of "-" that's
	// 1 less than the padding value
	fmt.Println(
		strings.Repeat("-", padding.ClusterName-1),
		strings.Repeat("-", padding.ServiceName-1),
		strings.Repeat("-", padding.Username-1),
		strings.Repeat("-", padding.Password-1),
		strings.Repeat("-", padding.ClusterIP-1),
		strings.Repeat("-", padding.ExternalIP-1),
	)
}

// printShowPgBouncerTextRow prints a row of the text data
func printShowPgBouncerTextRow(result msgs.ShowPgBouncerDetail, padding showPgBouncerTextPadding) {
	fmt.Printf("%s", util.Rpad(result.ClusterName, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(result.ServiceName, " ", padding.ServiceName))
	fmt.Printf("%s", util.Rpad(result.Username, " ", padding.Username))
	fmt.Printf("%s", util.Rpad(result.Password, " ", padding.Password))
	fmt.Printf("%s", util.Rpad(result.ServiceClusterIP, " ", padding.ClusterIP))
	fmt.Printf("%s", util.Rpad(result.ServiceExternalIP, " ", padding.ExternalIP))
	fmt.Println("")
}

// printUpdatePgBouncerText prints out the information about how each pgBouncer
// updat efared after a request
// printShowPgBouncerText renders a text response
func printUpdatePgBouncerText(response msgs.UpdatePgBouncerResponse) {
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

	// make the interface for the pgbouncer clusters
	updatePgBouncerInterface := makeUpdatePgBouncerInterface(response.Results)

	// format the header
	// start by setting up the different text paddings
	padding := updatePgBouncerTextPadding{
		ClusterName:  getMaxLength(updatePgBouncerInterface, headingCluster, "ClusterName"),
		ErrorMessage: getMaxLength(updatePgBouncerInterface, headingErrorMessage, "ErrorMessage"),
		Status:       len(headingStatus) + 1,
	}

	printUpdatePgBouncerTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printUpdatePgBouncerTextRow(result, padding)
	}
}

// printUpdatePgBouncerTextHeader prints out the header
func printUpdatePgBouncerTextHeader(padding updatePgBouncerTextPadding) {
	// print the header
	fmt.Println("")
	fmt.Printf("%s", util.Rpad(headingCluster, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(headingStatus, " ", padding.Status))
	fmt.Printf("%s", util.Rpad(headingErrorMessage, " ", padding.ErrorMessage))
	fmt.Println("")

	// print the layer below the header...which prints out a bunch of "-" that's
	// 1 less than the padding value
	fmt.Println(
		strings.Repeat("-", padding.ClusterName-1),
		strings.Repeat("-", padding.Status-1),
		strings.Repeat("-", padding.ErrorMessage-1),
	)
}

// printUpdatePgBouncerTextRow prints a row of the text data
func printUpdatePgBouncerTextRow(result msgs.UpdatePgBouncerDetail, padding updatePgBouncerTextPadding) {
	// set the text-based status
	status := "ok"
	if result.Error {
		status = "error"
	}

	fmt.Printf("%s", util.Rpad(result.ClusterName, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(status, " ", padding.Status))
	fmt.Printf("%s", util.Rpad(result.ErrorMessage, " ", padding.ErrorMessage))
	fmt.Println("")
}

// showPgBouncer prepares to make an API requests to display information about
// one or more pgBouncer deployments. "clusterNames" is an array of cluster
// names to iterate over
func showPgBouncer(namespace string, clusterNames []string) {
	// first, determine if any arguments have been pass in
	if len(clusterNames) == 0 && Selector == "" {
		fmt.Println("Error: You must provide at least one cluster name, or use a selector with the `--selector` flag")
		os.Exit(1)
	}

	// next prepare the request!
	request := msgs.ShowPgBouncerRequest{
		ClusterNames: clusterNames,
		Namespace:    namespace,
		Selector:     Selector,
	}

	// and make the API request!
	response, err := api.ShowPgBouncer(httpclient, &SessionCredentials, request)

	// if there is a bona-fide error, log and exit
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	// great! now we can work on interpreting the results and outputting them
	// per the user's desired output format
	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		printJSON(response)
	default:
		printShowPgBouncerText(response)
	}
}

// updatePgBouncer prepares to make an API requests to update information about
// a pgBouncer deployment in a cluster
// one or more pgBouncer deployments. "clusterNames" is an array of cluster
// names to iterate over
func updatePgBouncer(namespace string, clusterNames []string) {
	// first, determine if any arguments have been pass in
	if len(clusterNames) == 0 && Selector == "" {
		fmt.Println("Error: You must provide at least one cluster name, or use a selector with the `--selector` flag")
		os.Exit(1)
	}

	// next prepare the request!
	request := msgs.UpdatePgBouncerRequest{
		ClusterNames:   clusterNames,
		CPURequest:     PgBouncerCPURequest,
		CPULimit:       PgBouncerCPULimit,
		MemoryRequest:  PgBouncerMemoryRequest,
		MemoryLimit:    PgBouncerMemoryLimit,
		Namespace:      namespace,
		Replicas:       PgBouncerReplicas,
		RotatePassword: RotatePassword,
		Selector:       Selector,
	}

	if err := util.ValidateQuantity(request.CPURequest, "cpu"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(request.CPULimit, "cpu-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(request.MemoryRequest, "memory"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(request.MemoryLimit, "memory-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// and make the API request!
	response, err := api.UpdatePgBouncer(httpclient, &SessionCredentials, request)

	// if there is a bona-fide error, log and exit
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	// great! now we can work on interpreting the results and outputting them
	// per the user's desired output format
	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		printJSON(response)
	default:
		printUpdatePgBouncerText(response)
	}
}
