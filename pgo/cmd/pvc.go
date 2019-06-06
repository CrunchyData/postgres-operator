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

func showPVC(args []string, ns string) {
	log.Debugf("showPVC called %v", args)

	// ShowPVCRequest ...
	r := msgs.ShowPVCRequest{}
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION
	r.NodeLabel = NodeLabel
	r.PVCRoot = PVCRoot

	if AllFlag {
		//special case to just list all the PVCs
		r.PVCName = ""
		r.PVCRoot = ""
		printPVC(&r)
	} else {
		//args are a list of pvc names...for this case show details
		for _, arg := range args {
			r.PVCName = arg
			log.Debugf("show pvc called for %s", arg)
			printPVC(&r)

		}
	}

}

func printPVC(r *msgs.ShowPVCRequest) {

	response, err := api.ShowPVC(httpclient, r, &SessionCredentials)

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

	if AllFlag {
		fmt.Println("All Operator Labeled PVCs")
	}

	for k, v := range response.Results {
		if AllFlag {
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
