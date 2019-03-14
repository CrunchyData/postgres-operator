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
	"encoding/json"
	"fmt"
	"os"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

// PasswordLength password length flag
var PasswordLength int

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage PostgreSQL users",
	Long: `USER allows you to manage users and passwords across a set of clusters. For example:

	pgo user --selector=name=mycluster --update-passwords
	pgo user --change-password=bob --expired=300 --selector=name=mycluster --password=newpass`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("user called")
		userManager(Namespace)
	},
}

func init() {
	RootCmd.AddCommand(userCmd)

	userCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	userCmd.Flags().StringVarP(&Expired, "expired", "", "", "required flag when updating passwords that will expire in X days using --update-passwords flag.")
	userCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "", 30, "Sets passwords for new users to X days.")
	userCmd.Flags().StringVarP(&ChangePasswordForUser, "change-password", "", "", "Updates the password for a user on selective clusters.")
	userCmd.Flags().StringVarP(&UserDBAccess, "db", "", "", "Grants the user access to a database.")
	userCmd.Flags().StringVarP(&Password, "password", "", "", "Specifies the user password when updating a user password or creating a new user.")
	userCmd.Flags().BoolVarP(&UpdatePasswords, "update-passwords", "", false, "Performs password updating on expired passwords.")
	userCmd.Flags().IntVarP(&PasswordLength, "password-length", "", 12, "If no password is supplied, this is the length of the auto generated password")

}

// userManager ...
func userManager(ns string) {

	request := msgs.UserRequest{}
	request.Namespace = ns
	request.Selector = Selector
	request.Password = Password
	request.PasswordAgeDays = PasswordAgeDays
	request.ChangePasswordForUser = ChangePasswordForUser
	request.DeleteUser = DeleteUser
	request.ValidDays = ValidDays
	request.UserDBAccess = UserDBAccess
	request.Expired = Expired
	request.UpdatePasswords = UpdatePasswords
	request.ManagedUser = ManagedUser
	request.ClientVersion = msgs.PGO_VERSION
	request.PasswordLength = PasswordLength

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

func createUser(args []string, ns string) {

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
	r.Password = Password
	r.ManagedUser = ManagedUser
	r.UserDBAccess = UserDBAccess
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
func deleteUser(username string, ns string) {
	log.Debugf("deleteUser called %v", username)

	log.Debugf("deleting user %s selector=%s", username, Selector)
	response, err := api.DeleteUser(httpclient, username, Selector, &SessionCredentials, ns)

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

	for _, v := range args {

		response, err := api.ShowUser(httpclient, v, Selector, Expired, &SessionCredentials, ns)
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
	if len(detail.ExpiredMsgs) > 0 {
		fmt.Printf("\nexpired passwords: \n")
		for _, e := range detail.ExpiredMsgs {
			fmt.Println(e)
		}
	}

}
