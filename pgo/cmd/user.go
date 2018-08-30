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
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/spf13/cobra"
	"os"
)

// PasswordAgeDays password age flag
var PasswordAgeDays int

// ChangePasswordForUser change password flag
var ChangePasswordForUser string

// DeleteUser delete user flag
var DeleteUser string

// ValidDays valid days flag
var ValidDays string

// UserDBAccess user db access flag
var UserDBAccess string

// Expired expired flag
var Expired string

// UpdatePasswords update passwords flag
var UpdatePasswords bool

// ManagedUser managed user flag
var ManagedUser bool

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage PostgreSQL users",
	Long: `USER allows you to manage users and passwords across a set of clusters. For example:

	pgo user --selector=name=mycluster --update-passwords
	pgo user --expired=7 --selector=name=mycluster
	pgo user --change-password=bob --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("user called")
		userManager()
	},
}

func init() {
	RootCmd.AddCommand(userCmd)

	userCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	userCmd.Flags().StringVarP(&Expired, "expired", "e", "", "Shows passwords that will expire in X days.")
	userCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "v", 30, "Sets passwords for new users to X days.")
	userCmd.Flags().StringVarP(&ChangePasswordForUser, "change-password", "c", "", "Updates the password for a user on selective clusters.")
	userCmd.Flags().StringVarP(&UserDBAccess, "db", "b", "", "Grants the user access to a database.")
	userCmd.Flags().BoolVarP(&UpdatePasswords, "update-passwords", "u", false, "Performs password updating on expired passwords.")
	userCmd.Flags().BoolVarP(&ManagedUser, "managed", "m", false, "Creates a user with secrets that can be managed by the Operator.")

}

// userManager ...
func userManager() {

	request := msgs.UserRequest{}
	request.Selector = Selector
	request.PasswordAgeDays = PasswordAgeDays
	request.ChangePasswordForUser = ChangePasswordForUser
	request.DeleteUser = DeleteUser
	request.ValidDays = ValidDays
	request.UserDBAccess = UserDBAccess
	request.Expired = Expired
	request.UpdatePasswords = UpdatePasswords
	request.ManagedUser = ManagedUser
	request.ClientVersion = msgs.PGO_VERSION

	response, err := api.UserManager(httpclient, &SessionCredentials, &request)

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

func createUser(args []string) {

	if Selector == "" {
		fmt.Println("Error: The --selector flag is required.")
		return
	}

	if len(args) == 0 {
		fmt.Println("Error: A user name argument is required.")
		return
	}

	r := new(msgs.CreateUserRequest)
	r.Name = args[0]
	r.Selector = Selector
	r.ManagedUser = ManagedUser
	r.UserDBAccess = UserDBAccess
	r.PasswordAgeDays = PasswordAgeDays
	r.ClientVersion = msgs.PGO_VERSION

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
func deleteUser(username string) {
	log.Debugf("deleteUser called %v\n", username)

	log.Debug("deleting user " + username + " selector " + Selector)
	response, err := api.DeleteUser(httpclient, username, Selector, &SessionCredentials)

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
func showUser(args []string) {

	log.Debugf("showUser called %v\n", args)

	log.Debug("selector is " + Selector)
	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	for _, v := range args {

		response, err := api.ShowUser(httpclient, v, Selector, &SessionCredentials)
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

}

// printUsers
func printUsers(detail *msgs.ShowUserDetail) {
	fmt.Println("")
	fmt.Println("cluster : " + detail.Cluster.Spec.Name)

	for _, s := range detail.Secrets {
		fmt.Println("")
		fmt.Println("secret : " + s.Name)
		fmt.Println(TreeBranch + "username: " + s.Username)
		fmt.Println(TreeTrunk + "password: " + s.Password)
	}

}
