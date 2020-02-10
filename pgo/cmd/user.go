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
)

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

	if Username == "" {
		fmt.Println("Error: --username is required")
		return
	}

	r := new(msgs.CreateUserRequest)
	r.Clusters = args
	r.Username = Username
	r.Selector = Selector
	r.Password = Password
	r.ManagedUser = ManagedUser
	//r.UserDBAccess = UserDBAccess
	r.PasswordAgeDays = PasswordAgeDays
	r.ClientVersion = msgs.PGO_VERSION
	r.PasswordLength = PasswordLength
	r.Namespace = ns

	response, err := api.CreateUser(httpclient, &SessionCredentials, r)
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
