package cmd

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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

func updatePgorole(args []string, ns string) {
	var err error

	if Permissions == "" {
		fmt.Println("Error: --permissions flag is required.")
		return
	}

	if len(args) == 0 {
		fmt.Println("Error: A pgorole name argument is required.")
		return
	}

	r := new(msgs.UpdatePgoroleRequest)
	r.PgoroleName = args[0]
	r.Namespace = ns
	r.ChangePermissions = PgoroleChangePermissions
	r.PgorolePermissions = Permissions
	r.ClientVersion = msgs.PGO_VERSION

	response, err := api.UpdatePgorole(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		fmt.Println("pgorole updated ")
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

func showPgorole(args []string, ns string) {

	r := new(msgs.ShowPgoroleRequest)
	r.PgoroleName = args
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION

	if len(args) == 0 && !AllFlag {
		fmt.Println("Error: either a pgorole name or --all flag is required")
		os.Exit(2)
	}
	if len(args) == 0 && AllFlag {
		args = []string{""}
	}

	response, err := api.ShowPgorole(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.RoleInfo) == 0 {
		fmt.Println("No pgoroles found.")
		return
	}

	log.Debugf("response = %v", response)

	for _, pgorole := range response.RoleInfo {
		fmt.Println("")
		fmt.Println("pgorole : " + pgorole.Name)
		fmt.Println("permissions : " + pgorole.Permissions)
	}

}

func createPgorole(args []string, ns string) {

	if Permissions == "" {
		fmt.Println("Error: permissions flag is required.")
		return
	}

	if len(args) == 0 {
		fmt.Println("Error: A pgorole name argument is required.")
		return
	}
	var err error
	//create the request
	r := new(msgs.CreatePgoroleRequest)
	r.PgoroleName = args[0]
	r.PgorolePermissions = Permissions
	r.Namespace = ns
	r.ClientVersion = msgs.PGO_VERSION

	response, err := api.CreatePgorole(httpclient, &SessionCredentials, r)

	log.Debugf("response is %v", response)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		fmt.Println("Created pgorole.")
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

func deletePgorole(args []string, ns string) {

	log.Debugf("deletePgorole called %v", args)

	r := msgs.DeletePgoroleRequest{}
	r.PgoroleName = args
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns

	if AllFlag {
		args = make([]string, 1)
		args[0] = "all"
	}

	log.Debugf("deleting pgorole %v", args)

	response, err := api.DeletePgorole(httpclient, &r, &SessionCredentials)
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
