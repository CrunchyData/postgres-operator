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
	"encoding/json"
	"fmt"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	log "github.com/sirupsen/logrus"
	"os"
)

func showNamespace(args []string) {
	// copy arg list to keep track of original cli args
	nsList := make([]string, len(args))
	copy(nsList, args)

	defNS := os.Getenv("PGO_NAMESPACE")
	if defNS != "" {
		fmt.Printf("current local default namespace: %s\n", defNS)
		found := false
		// check if default is already in nsList
		for _, ns := range nsList {
			if ns == defNS {
				found = true
				break
			}
		}

		if !found {
			log.Debugf("adding default namespace [%s] to args", defNS)
			nsList = append(nsList, defNS)
		}
	}

	r := msgs.ShowNamespaceRequest{}
	r.ClientVersion = msgs.PGO_VERSION
	r.Args = nsList
	r.AllFlag = AllFlag

	if len(nsList) == 0 && AllFlag == false {
		fmt.Println("Error: namespace args or --all is required")
		os.Exit(2)
	}

	log.Debugf("showNamespace called %v", nsList)

	response, err := api.ShowNamespace(httpclient, &SessionCredentials, &r)

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

	if len(response.Results) == 0 {
		fmt.Println("Nothing found.")
		return
	}

	fmt.Printf("pgo username: %s\n", response.Username)

	fmt.Printf("%s", util.Rpad("namespace", " ", 25))
	fmt.Printf("%s", util.Rpad("useraccess", " ", 20))
	fmt.Printf("%s\n", util.Rpad("installaccess", " ", 20))

	var accessible, iAccessible string
	for _, result := range response.Results {
		accessible = GREEN(util.Rpad("accessible", " ", 20))
		if !result.UserAccess {
			accessible = RED(util.Rpad("no access", " ", 20))
		}
		iAccessible = GREEN(util.Rpad("accessible", " ", 20))
		if !result.InstallationAccess {
			iAccessible = RED(util.Rpad("no access", " ", 20))
		}
		fmt.Printf("%s", util.Rpad(result.Namespace, " ", 25))
		fmt.Printf("%s", accessible)
		fmt.Printf("%s\n", iAccessible)
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

	log.Debugf("createNamespace response %v", response)
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

	response, err := api.UpdateNamespace(httpclient, r, &SessionCredentials)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		fmt.Println("namespace updated ")
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}
