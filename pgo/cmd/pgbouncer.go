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
	"os"
)

func createPgbouncer(args []string, ns string) {

	if Selector == "" && len(args) == 0 {
		fmt.Println("Error: The --selector flag is required.")
		return
	}

	r := new(msgs.CreatePgbouncerRequest)
	r.Args = args
	r.Namespace = ns
	r.Selector = Selector
	r.ClientVersion = msgs.PGO_VERSION
	r.PgbouncerUser = "pgbouncer"
	r.PgbouncerPass = PgBouncerPassword

	if !(len(PgBouncerUser) > 0) {
		r.PgbouncerUser = "pgbouncer"
	} else {
		r.PgbouncerUser = PgBouncerUser
	}

	response, err := api.CreatePgbouncer(httpclient, &SessionCredentials, r)
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
		for _, v := range response.Results {
			fmt.Println(v)
		}
		os.Exit(2)
	}

}

func deletePgbouncer(args []string, ns string) {

	if Selector == "" && len(args) == 0 {
		fmt.Println("Error: The --selector flag or a cluster name is required.")
		return
	}

	r := new(msgs.DeletePgbouncerRequest)
	r.Args = args
	r.Selector = Selector
	r.Namespace = ns
	r.ClientVersion = msgs.PGO_VERSION

	response, err := api.DeletePgbouncer(httpclient, &SessionCredentials, r)
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
