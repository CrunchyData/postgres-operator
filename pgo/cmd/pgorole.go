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
	"context"
	"fmt"
	"os"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
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

	r := msgs.UpdatePgoRoleRequest{
		PgoroleName:        args[0],
		Namespace:          ns,
		ChangePermissions:  PgoroleChangePermissions,
		PgorolePermissions: Permissions,
		ClientVersion:      msgs.PGO_VERSION,
	}

	response, err := apiClient.UpdatePgoRole(context.Background(), r)

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

	r := msgs.ShowPgoRoleRequest{
		PgoroleName:   args,
		Namespace:     ns,
		AllFlag:       AllFlag,
		ClientVersion: msgs.PGO_VERSION,
	}

	if len(args) == 0 && !AllFlag {
		fmt.Println("Error: either a pgorole name or --all flag is required")
		os.Exit(2)
	}
	if len(args) == 0 && AllFlag {
		args = []string{""}
	}

	response, err := apiClient.ShowPgoRole(context.Background(), r)

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
	r := msgs.CreatePgoRoleRequest{
		PgoroleName:        args[0],
		PgorolePermissions: Permissions,
		Namespace:          ns,
		ClientVersion:      msgs.PGO_VERSION,
	}

	response, err := apiClient.CreatePgoRole(context.Background(), r)

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

	r := msgs.DeletePgoRoleRequest{
		Namespace:     ns,
		PgoroleName:   args,
		AllFlag:       AllFlag,
		ClientVersion: msgs.PGO_VERSION,
	}

	if AllFlag {
		args = make([]string, 1)
		args[0] = "all"
	}

	log.Debugf("deleting pgorole %v", args)

	response, err := apiClient.DeletePgoRole(context.Background(), r)
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
