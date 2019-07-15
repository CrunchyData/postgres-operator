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
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"os"
)

func showNamespace(args []string, ns string) {

	log.Debugf("showNamespace called %v", args)

	response, err := api.ShowNamespace(httpclient, &SessionCredentials, ns)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if OutputFormat == "json" {
		b, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println(string(b))
		return
	}

	defNS := os.Getenv("PGO_NAMESPACE")
	if defNS != "" {
		fmt.Printf("current local default namespace: %s\n", defNS)
	}
	if len(response.Results) == 0 {
		fmt.Println("Nothing found.")
		return
	}

	var accessible string
	for _, result := range response.Results {
		accessible = GREEN("accessible")
		if !result.UserAccess {
			accessible = RED("no access")
		}
		fmt.Printf("namespace: %s (%s)\n", result.Namespace, accessible)
	}

}

func createNamespace(args []string, ns string) {
	log.Debugf("createNamespace called %v [%s]", args, Selector)

	r := msgs.CreateNamespaceRequest{}
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns
	r.Args = args

	if len(args) == 0 {
		fmt.Println("Error: namespace names are required")
		os.Exit(2)
	}

	response, err := api.CreateNamespace(httpclient, &SessionCredentials, &r)
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

func deleteNamespace(args []string, ns string) {
	log.Debugf("deleteNamespace called %v [%s]", args, Selector)

	r := msgs.DeleteNamespaceRequest{}
	r.Selector = Selector
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns
	r.Args = args

	if Selector != "" && len(args) > 0 {
		fmt.Println("Error: can not specify both arguments and --selector")
		os.Exit(2)
	}

	response, err := api.DeleteNamespace(httpclient, &r, &SessionCredentials)
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
func updateNamespace(args []string) {
	var err error

	if len(args) == 0 {
		fmt.Println("Error: A Namespace name argument is required.")
		return
	}

	r := new(msgs.UpdateNamespaceRequest)
	r.Args = args
	r.ClientVersion = msgs.PGO_VERSION

	response, err := api.UpdateNamespace(httpclient, &SessionCredentials, r)

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
