package cmd

/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	utiloperator "github.com/crunchydata/postgres-operator/internal/util"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
)

// userTextPadding contains the values for what the text padding should be
type userTextPadding struct {
	ClusterName  int
	ErrorMessage int
	Expires      int
	Password     int
	Username     int
	Status       int
}

// PasswordAgeDays password age flag
var PasswordAgeDays int

// Username is a postgres username
var Username string

// Expired expired flag
var Expired int

// PasswordLength password length flag
var PasswordLength int

// PasswordValidAlways allows a user to explicitly set that their passowrd
// is always valid (i.e. no expiration time)
var PasswordValidAlways bool

// ShowSystemAccounts enables the display of the PostgreSQL user accounts that
// perform system functions, such as the "postgres" user, and for taking action
// on these accounts
var ShowSystemAccounts bool

func createUser(args []string, ns string) {
	username := strings.TrimSpace(Username)

	// ensure the username is nonempty
	if username == "" {
		fmt.Println("Error: --username is required")
		os.Exit(1)
	}

	// check to see if this is a system account. if it is, do not let the request
	// go through
	if utiloperator.IsPostgreSQLUserSystemAccount(username) {
		fmt.Println("Error:", username, "is a system account and cannot be used")
		os.Exit(1)
	}

	request := msgs.CreateUserRequest{
		AllFlag:         AllFlag,
		Clusters:        args,
		ManagedUser:     ManagedUser,
		Namespace:       ns,
		Password:        Password,
		PasswordAgeDays: PasswordAgeDays,
		PasswordLength:  PasswordLength,
		PasswordType:    PasswordType,
		Username:        username,
		Selector:        Selector,
	}

	// determine if the user provies a valid password type
	if _, err := msgs.GetPasswordType(PasswordType); err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	response, err := api.CreateUser(httpclient, &SessionCredentials, &request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// great! now we can work on interpreting the results and outputting them
	// per the user's desired output format
	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		printJSON(response)
	default:
		printCreateUserText(response)
	}
}

// deleteUser ...
func deleteUser(args []string, ns string) {
	log.Debugf("deleting user %s selector=%s args=%v", Username, Selector, args)

	if Username == "" {
		fmt.Println("Error: --username is required")
		return
	}

	request := msgs.DeleteUserRequest{
		AllFlag:   AllFlag,
		Clusters:  args,
		Namespace: ns,
		Selector:  Selector,
		Username:  Username,
	}

	response, err := api.DeleteUser(httpclient, &SessionCredentials, &request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// great! now we can work on interpreting the results and outputting them
	// per the user's desired output format
	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		printJSON(response)
	default:
		printDeleteUserText(response)
	}
}

// generateUserPadding returns the paddings based on the values of the response
func generateUserPadding(results []msgs.UserResponseDetail) userTextPadding {
	// make the interface for the users
	userInterface := makeUserInterface(results)

	// set up the text padding
	return userTextPadding{
		ClusterName:  getMaxLength(userInterface, headingCluster, "ClusterName"),
		ErrorMessage: getMaxLength(userInterface, headingErrorMessage, "ErrorMessage"),
		Expires:      getMaxLength(userInterface, headingExpires, "ValidUntil"),
		Password:     getMaxLength(userInterface, headingPassword, "Password"),
		Status:       len(headingStatus) + 1,
		Username:     getMaxLength(userInterface, headingUsername, "Username"),
	}
}

// makeUserInterface returns an interface slice of the available values
// in pgo create user
func makeUserInterface(values []msgs.UserResponseDetail) []interface{} {
	// iterate through the list of values to make the interface
	userInterface := make([]interface{}, len(values))

	for i, value := range values {
		userInterface[i] = value
	}

	return userInterface
}

// printCreateUserText prints out the information that is created after
// pgo create user is called
func printCreateUserText(response msgs.CreateUserResponse) {
	// if the request errored, return the message here and exit with an error
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// if no results returned, return an error
	if len(response.Results) == 0 {
		fmt.Println("No users created.")
		return
	}

	padding := generateUserPadding(response.Results)

	// print the header
	printUserTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printUserTextRow(result, padding)
	}
}

// printDeleteUserText prints out the information that is created after
// pgo delete user is called
func printDeleteUserText(response msgs.DeleteUserResponse) {
	// if the request errored, return the message here and exit with an error
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// if no results returned, return an error
	if len(response.Results) == 0 {
		fmt.Println("No users deleted.")
		return
	}

	padding := generateUserPadding(response.Results)

	// print the header
	printUserTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printUserTextRow(result, padding)
	}
}

// printShowUserText prints out the information from calling pgo show user
func printShowUserText(response msgs.ShowUserResponse) {
	// if the request errored, return the message here and exit with an error
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// if no results returned, return an error
	if len(response.Results) == 0 {
		fmt.Println("No users found.")
		return
	}

	padding := generateUserPadding(response.Results)

	// print the header
	printUserTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printUserTextRow(result, padding)
	}
}

// printUpdateUserText prints out the information from calling pgo update user
func printUpdateUserText(response msgs.UpdateUserResponse) {
	// if the request errored, return the message here and exit with an error
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// if no results returned, return an error
	if len(response.Results) == 0 {
		fmt.Println("No users updated.")
		return
	}

	padding := generateUserPadding(response.Results)

	// print the header
	printUserTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printUserTextRow(result, padding)
	}
}

// printUserTextHeader prints out the header
func printUserTextHeader(padding userTextPadding) {
	// print the header
	fmt.Println("")
	fmt.Printf("%s", util.Rpad(headingCluster, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(headingUsername, " ", padding.Username))
	fmt.Printf("%s", util.Rpad(headingPassword, " ", padding.Password))
	fmt.Printf("%s", util.Rpad(headingExpires, " ", padding.Expires))
	fmt.Printf("%s", util.Rpad(headingStatus, " ", padding.Status))
	fmt.Printf("%s", util.Rpad(headingErrorMessage, " ", padding.ErrorMessage))
	fmt.Println("")

	// print the layer below the header...which prints out a bunch of "-" that's
	// 1 less than the padding value
	fmt.Println(
		strings.Repeat("-", padding.ClusterName-1),
		strings.Repeat("-", padding.Username-1),
		strings.Repeat("-", padding.Password-1),
		strings.Repeat("-", padding.Expires-1),
		strings.Repeat("-", padding.Status-1),
		strings.Repeat("-", padding.ErrorMessage-1),
	)
}

// printUserTextRow prints a row of the text data
func printUserTextRow(result msgs.UserResponseDetail, padding userTextPadding) {
	expires := result.ValidUntil

	// check for special values of expires, e.g. if the password matches special
	// values to indicate if it has expired or not
	switch {
	case expires == "" || expires == utiloperator.SQLValidUntilAlways:
		expires = "never"
	case expires == utiloperator.SQLValidUntilNever:
		expires = "expired"
	}

	password := result.Password

	// set the text-based status, and use it to drive some of the display
	status := "ok"

	if result.Error {
		expires = ""
		password = ""
		status = "error"
	}

	fmt.Printf("%s", util.Rpad(result.ClusterName, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(result.Username, " ", padding.Username))
	fmt.Printf("%s", util.Rpad(password, " ", padding.Password))
	fmt.Printf("%s", util.Rpad(expires, " ", padding.Expires))
	fmt.Printf("%s", util.Rpad(status, " ", padding.Status))
	fmt.Printf("%s", util.Rpad(result.ErrorMessage, " ", padding.ErrorMessage))
	fmt.Println("")
}

// showUser prepares the API attributes for getting information about PostgreSQL
// users in clusters
func showUser(args []string, ns string) {
	request := msgs.ShowUserRequest{
		AllFlag:            AllFlag,
		Clusters:           args,
		Expired:            Expired,
		Namespace:          ns,
		Selector:           Selector,
		ShowSystemAccounts: ShowSystemAccounts,
	}

	response, err := api.ShowUser(httpclient, &SessionCredentials, &request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// great! now we can work on interpreting the results and outputting them
	// per the user's desired output format
	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		printJSON(response)
	default:
		printShowUserText(response)
	}
}

// updateUser prepares the API call for updating attributes of a PostgreSQL
// user
func updateUser(clusterNames []string, namespace string) {
	// set up the reuqest
	request := msgs.UpdateUserRequest{
		AllFlag:                  AllFlag,
		Clusters:                 clusterNames,
		Expired:                  Expired,
		ExpireUser:               ExpireUser,
		ManagedUser:              ManagedUser,
		Namespace:                namespace,
		Password:                 Password,
		PasswordAgeDays:          PasswordAgeDays,
		PasswordLength:           PasswordLength,
		PasswordValidAlways:      PasswordValidAlways,
		PasswordType:             PasswordType,
		RotatePassword:           RotatePassword,
		Selector:                 Selector,
		SetSystemAccountPassword: ShowSystemAccounts,
		Username:                 strings.TrimSpace(Username),
	}

	// check to see if EnableLogin or DisableLogin is set. If so, set a value
	// for the LoginState parameter
	if EnableLogin {
		request.LoginState = msgs.UpdateUserLoginEnable
	} else if DisableLogin {
		request.LoginState = msgs.UpdateUserLoginDisable
	}

	// check to see if this is a system account if a user name is passed in
	if request.Username != "" && utiloperator.IsPostgreSQLUserSystemAccount(request.Username) && !request.SetSystemAccountPassword {
		fmt.Println("Error:", request.Username, "is a system account and cannot be used. "+
			"You can override this with the \"--set-system-account-password\" flag.")
		os.Exit(1)
	}

	// determine if the user provies a valid password type
	if _, err := msgs.GetPasswordType(PasswordType); err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	response, err := api.UpdateUser(httpclient, &SessionCredentials, &request)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// great! now we can work on interpreting the results and outputting them
	// per the user's desired output format
	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		printJSON(response)
	default:
		printUpdateUserText(response)
	}
}
