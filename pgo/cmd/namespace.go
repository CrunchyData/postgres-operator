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
