package cmd

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"os"
)

func showPVC(args []string) {
	log.Debugf("showPVC called %v", args)

	if args[0] == "all" {
		//special case to just list all the PVCs
		printPVC(args[0], "")
	} else {
		//args are a list of pvc names...for this case show details
		for _, arg := range args {
			log.Debugf("show pvc called for %s", arg)
			printPVC(arg, PVCRoot)

		}
	}

}

func printPVC(pvcName, pvcRoot string) {

	response, err := api.ShowPVC(httpclient, pvcName, pvcRoot, &SessionCredentials)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Error {
		fmt.Println("Error: " + response.Status.Msg)
		return
	}

	if len(response.Results) == 0 {
		fmt.Println("No PVC Results")
		return
	}
	log.Debugf("response = %v", response)

	if pvcName == "all" {
		fmt.Println("All Operator Labeled PVCs")
	}

	for k, v := range response.Results {
		if pvcName == "all" {
			if v != "" {
				fmt.Printf("%s%s\n", TreeTrunk, v)
			}
		} else {
			if k == len(response.Results)-1 {
				fmt.Printf("%s%s\n", TreeTrunk, "/"+v)
			} else {
				fmt.Printf("%s%s\n", TreeBranch, "/"+v)
			}
		}
	}

}
