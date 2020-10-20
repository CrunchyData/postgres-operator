package cmd

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"os"

	"github.com/crunchydata/postgres-operator/cmd/pgo/api"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

func showPVC(args []string, ns string) {
	log.Debugf("showPVC called %v", args)

	// ShowPVCRequest ...
	r := msgs.ShowPVCRequest{}
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION

	if AllFlag {
		//special case to just list all the PVCs
		r.ClusterName = ""
		printPVC(&r)
	} else {
		//args are a list of pvc names...for this case show details
		for _, arg := range args {
			r.ClusterName = arg
			log.Debugf("show pvc called for %s", arg)
			printPVC(&r)
		}
	}

}

func printPVC(r *msgs.ShowPVCRequest) {

	response, err := api.ShowPVC(httpclient, r, &SessionCredentials)

	log.Debugf("response = %v", response)

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

	fmt.Printf("%-20s\t%-30s\n", "Cluster Name", "PVC Name")

	for _, v := range response.Results {
		fmt.Printf("%-20s\t%-30s\n", v.ClusterName, v.PVCName)
	}

}
