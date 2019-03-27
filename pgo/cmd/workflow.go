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
	"github.com/crunchydata/postgres-operator/pgo/util"
	log "github.com/sirupsen/logrus"
	"os"
)

func showWorkflow(args []string, ns string) {
	log.Debugf("showWorkflow called %v", args)

	if len(args) < 1 {
		fmt.Println("Error: workflow ID is a required parameter")
		os.Exit(2)
	}

	printWorkflow(args[0], ns)

}

func printWorkflow(id, ns string) {

	response, err := api.ShowWorkflow(httpclient, id, &SessionCredentials, ns)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Error {
		fmt.Println("Error: " + response.Status.Msg)
		return
	}

	log.Debugf("response = %v", response)

	fmt.Printf("%s%s\n", util.Rpad("parameter", " ", 20), "value")
	fmt.Printf("%s%s\n", util.Rpad("---------", " ", 20), "-----")
	for k, v := range response.Results.Parameters {
		fmt.Printf("%s%s\n", util.Rpad(k, " ", 20), v)
	}

}
