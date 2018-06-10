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
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/spf13/cobra"
	"net/http"
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
	Short: "manage users",
	Long: `USER allows you to manage users and passwords across a set of clusters
For example:

pgo user --selector=name=mycluster --update-passwords
pgo user --expired=7 --selector=name=mycluster
pgo user --change-password=bob --selector=name=mycluster
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("user called")
		userManager()
	},
}

func init() {
	RootCmd.AddCommand(userCmd)

	userCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	userCmd.Flags().StringVarP(&Expired, "expired", "e", "", "--expired=7 shows passwords that will expired in 7 days")
	userCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "v", 30, "--valid-days=7 sets passwords for new users to 7 days")
	userCmd.Flags().StringVarP(&ChangePasswordForUser, "change-password", "c", "", "--change-password=bob updates the password for a user on selective clusters")
	userCmd.Flags().StringVarP(&UserDBAccess, "db", "b", "", "--db=userdb grants the user access to a database")
	//userCmd.Flags().StringVarP(&DeleteUser, "delete-user", "d", "", "--delete-user=bob deletes a user on selective clusters")
	userCmd.Flags().BoolVarP(&UpdatePasswords, "update-passwords", "u", false, "--update-passwords performs password updating on expired passwords")
	userCmd.Flags().BoolVarP(&ManagedUser, "managed", "m", false, "--managed creates a user with secrets")

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
	request.ClientVersion = ClientVersion

	jsonValue, _ := json.Marshal(request)

	url := APIServerURL + "/user"
	log.Debug("User called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}
	log.Debugf("%v\n", resp)
	StatusCheck(resp)

	defer resp.Body.Close()

	var response msgs.UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Error(err)
		log.Println(err)
		return
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		log.Error(RED(response.Status.Msg))
		os.Exit(2)
	}

}

func createUser(args []string) {

	if Selector == "" {
		log.Error("selector flag is required")
		return
	}

	if len(args) == 0 {
		log.Error("user name argument is required")
		return
	}

	r := new(msgs.CreateUserRequest)
	r.Name = args[0]
	r.Selector = Selector
	r.ManagedUser = ManagedUser
	r.UserDBAccess = UserDBAccess
	r.PasswordAgeDays = PasswordAgeDays
	r.ClientVersion = ClientVersion

	jsonValue, _ := json.Marshal(r)
	url := APIServerURL + "/users"
	log.Debug("createUser called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	log.Debugf("%v\n", resp)
	StatusCheck(resp)

	defer resp.Body.Close()

	var response msgs.CreateUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Error(err)
		log.Println(err)
		return
	}

	if response.Status.Code == msgs.Ok {
		for _, v := range response.Results {
			fmt.Println(v)
		}
	} else {
		log.Error(RED(response.Status.Msg))
		os.Exit(2)
	}

}

// deleteUser ...
func deleteUser(username string) {
	log.Debugf("deleteUser called %v\n", username)

	log.Debug("deleting user " + username + " selector " + Selector)

	url := APIServerURL + "/usersdelete/" + username + "?selector=" + Selector + "&version=" + ClientVersion

	log.Debug("delete users called [" + url + "]")

	action := "GET"
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}
	log.Debugf("%v\n", resp)
	StatusCheck(resp)
	defer resp.Body.Close()
	var response msgs.DeleteUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Error(err)
		log.Println(err)
		return
	}

	if response.Status.Code == msgs.Ok {
		for _, result := range response.Results {
			fmt.Println(result)
		}
	} else {
		log.Error(RED(response.Status.Msg))
	}

}
