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
	"strings"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	utiloperator "github.com/crunchydata/postgres-operator/util"

	log "github.com/sirupsen/logrus"
)

// createUserTextPadding contains the values for what the text padding should be
type createUserTextPadding struct {
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

// DeleteUser delete user flag
var DeleteUser string

// ValidDays valid days flag
var ValidDays string

// UserDBAccess user db access flag
//var UserDBAccess string

// Expired expired flag
var Expired string

// PasswordLength password length flag
var PasswordLength int

// userManager ...
func updateUser(args []string, ns string, PasswordAgeDaysUpdate bool) {

	request := msgs.UpdateUserRequest{}
	request.Namespace = ns
	request.ExpireUser = ExpireUser
	request.Clusters = args
	request.AllFlag = AllFlag
	request.Selector = Selector
	request.Password = Password
	request.PasswordAgeDays = PasswordAgeDays
	request.PasswordAgeDaysUpdate = PasswordAgeDaysUpdate
	request.Username = Username
	request.DeleteUser = DeleteUser
	request.ValidDays = ValidDays
	//request.UserDBAccess = UserDBAccess
	request.Expired = Expired
	request.ManagedUser = ManagedUser
	request.ClientVersion = msgs.PGO_VERSION
	request.PasswordLength = PasswordLength

	response, err := api.UpdateUser(httpclient, &SessionCredentials, &request)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

func createUser(args []string, ns string) {
	username := strings.TrimSpace(Username)

	// ensure the username is nonempty
	if username == "" {
		fmt.Println("Error: --username is required")
		os.Exit(1)
	}

	// check to see if this is a system account. if it is, do not let the request
	// go through
	if utiloperator.CheckPostgreSQLUserSystemAccount(username) {
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
		Username:        username,
		Selector:        Selector,
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

	r := new(msgs.DeleteUserRequest)
	r.Username = Username
	r.Clusters = args
	r.AllFlag = AllFlag
	r.Selector = Selector
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns

	response, err := api.DeleteUser(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	if response.Status.Code == msgs.Ok {
		for _, result := range response.Results {
			fmt.Println(result)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
	}

}

// makeCreateUserInterface returns an interface slice of the avaialble values
// in pgo create user
func makeCreateUserInterface(values []msgs.CreateUserResponseDetail) []interface{} {
	// iterate through the list of values to make the interface
	createUserInterface := make([]interface{}, len(values))

	for i, value := range values {
		createUserInterface[i] = value
	}

	return createUserInterface
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
		fmt.Println("Nothing found.")
		return
	}

	// make the interface for the users
	createUserInterface := makeCreateUserInterface(response.Results)

	// format the header
	// start by setting up the different text paddings
	padding := createUserTextPadding{
		ClusterName:  getMaxLength(createUserInterface, headingCluster, "ClusterName"),
		ErrorMessage: getMaxLength(createUserInterface, headingErrorMessage, "ErrorMessage"),
		Expires:      getMaxLength(createUserInterface, headingExpires, "ValidUntil"),
		Password:     getMaxLength(createUserInterface, headingPassword, "Password"),
		Status:       len(headingStatus) + 1,
		Username:     getMaxLength(createUserInterface, headingUsername, "Username"),
	}

	printCreateUserTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printCreateUserTextRow(result, padding)
	}
}

// printCreateUserTextHeader prints out the header
func printCreateUserTextHeader(padding createUserTextPadding) {
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

// printCreateUserTextRow prints a row of the text data
func printCreateUserTextRow(result msgs.CreateUserResponseDetail, padding createUserTextPadding) {
	expires := result.ValidUntil

	// if expires is empty here, set it to never. This may be ovrriden if there is
	// an error
	if expires == "" {
		expires = "never"
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

// showUsers ...
func showUser(args []string, ns string) {

	log.Debugf("showUser called %v", args)

	log.Debugf("selector is %s", Selector)
	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	r := msgs.ShowUserRequest{}
	r.Clusters = args
	r.ClientVersion = msgs.PGO_VERSION
	r.Selector = Selector
	r.Namespace = ns
	r.Expired = Expired
	r.AllFlag = AllFlag

	response, err := api.ShowUser(httpclient, &SessionCredentials, &r)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(2)
	}

	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}
	if len(response.Results) == 0 {
		fmt.Println("No clusters found.")
		return
	}

	if OutputFormat == "json" {
		b, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println(string(b))
		return
	}

	for _, clusterDetail := range response.Results {
		printUsers(&clusterDetail)
	}

}

// printUsers
func printUsers(detail *msgs.ShowUserDetail) {
	fmt.Println("")
	fmt.Println("cluster : " + detail.Cluster.Spec.Name)

	if detail.ExpiredOutput == false {
		for _, s := range detail.Secrets {
			fmt.Println("")
			fmt.Println("secret : " + s.Name)
			fmt.Println(TreeBranch + "username: " + s.Username)
			fmt.Println(TreeTrunk + "password: " + s.Password)
		}
	} else {
		fmt.Printf("\nuser passwords expiring within %d days:\n", detail.ExpiredDays)
		fmt.Println("")
		if len(detail.ExpiredMsgs) > 0 {
			for _, e := range detail.ExpiredMsgs {
				fmt.Println(e)
			}
		}
	}

}
