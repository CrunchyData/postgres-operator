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
	"os"
)

func updatePgouser(args []string, ns string) {
	var err error

	if len(args) == 0 {
		fmt.Println("Error: A pgouser name argument is required.")
		return
	}

	r := new(msgs.UpdatePgouserRequest)
	r.PgouserName = args[0]
	r.Namespace = ns
	r.ChangePassword = PgouserChangePassword
	r.PgouserPassword = PgouserPassword
	r.ClientVersion = msgs.PGO_VERSION

	response, err := api.UpdatePgouser(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		fmt.Println("pgouser updated ")
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

func showPgouser(args []string, ns string) {

	r := new(msgs.ShowPgouserRequest)
	r.PgouserName = args
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION

	if len(args) == 0 && AllFlag {
		args = []string{""}
	}

	response, err := api.ShowPgouser(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.PgouserName) == 0 {
		fmt.Println("No pgousers found.")
		return
	}

	log.Debugf("response = %v", response)

	for _, pgouser := range response.PgouserName {
		fmt.Println("")
		fmt.Println("pgouser : " + pgouser)
	}

}

func createPgouser(args []string, ns string) {

	if PgouserPassword == "" {
		fmt.Println("Error: pgouser-password or generate-password flag is required.")
		return
	}

	if len(args) == 0 {
		fmt.Println("Error: A pgouser username argument is required.")
		return
	}
	var err error
	//create the request
	r := new(msgs.CreatePgouserRequest)
	r.PgouserName = args[0]
	r.PgouserPassword = PgouserPassword
	r.Namespace = ns
	r.ClientVersion = msgs.PGO_VERSION

	response, err := api.CreatePgouser(httpclient, &SessionCredentials, r)

	log.Debugf("response is %v", response)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		fmt.Println("Created pgouser.")
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

func deletePgouser(args []string, ns string) {

	log.Debugf("deletePgouser called %v", args)

	r := msgs.DeletePgouserRequest{}
	r.PgouserName = args
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns

	if AllFlag {
		args = make([]string, 1)
		args[0] = "all"
	}

	log.Debugf("deleting pgouser %v", args)

	response, err := api.DeletePgouser(httpclient, &r, &SessionCredentials)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}

	if response.Status.Code == msgs.Ok {
		for _, v := range response.Results {
			fmt.Println(v)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
	}

}
