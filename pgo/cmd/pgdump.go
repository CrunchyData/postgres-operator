// Package cmd provides the command line functions of the crunchy CLI
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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"os"
)

// createpgDumpBackup
func createpgDumpBackup(args []string, ns string) {
	log.Debugf("createpgDumpBackup called %v %s", args, BackupOpts)

	request := new(msgs.CreatepgDumpBackupRequest)
	request.Args = args
	request.Namespace = ns
	request.Selector = Selector
	request.PVCName = PVCName
	request.StorageConfig = StorageConfig
	request.BackupOpts = BackupOpts

	response, err := api.CreatepgDumpBackup(httpclient, &SessionCredentials, request)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.Results) == 0 {
		fmt.Println("No clusters found.")
		return
	}

}

// pgDump ....
func showpgDump(args []string, ns string) {
	log.Debugf("showpgDump called %v", args)

	for _, v := range args {
		response, err := api.ShowpgDump(httpclient, v, Selector, &SessionCredentials, ns)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

		if len(response.BackupList.Items) == 0 {
			fmt.Println("No pgDumps found for " + v + ".")
			return
		}

		log.Debugf("response = %v", response)
		log.Debugf("len of items = %d", len(response.BackupList.Items))

		for _, backup := range response.BackupList.Items {
			printDumpCRD(&backup)
		}
	}
}

// printBackrest
func printpgDump(result *msgs.ShowpgDumpDetail) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgDump : "+result.Name)
	fmt.Printf("%s%s\n", "", result.Info)

}

// printBackupCRD ...
func printDumpCRD(result *crv1.Pgbackup) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgdump : "+result.Spec.Name)

	fmt.Printf("%s%s\n", TreeBranch, "PVC Name:\t"+result.Spec.BackupPVC)
	fmt.Printf("%s%s\n", TreeBranch, "Access Mode:\t"+result.Spec.StorageSpec.AccessMode)
	fmt.Printf("%s%s\n", TreeBranch, "PVC Size:\t"+result.Spec.StorageSpec.Size)
	fmt.Printf("%s%s\n", TreeBranch, "Creation:\t"+result.ObjectMeta.CreationTimestamp.String())
	fmt.Printf("%s%s\n", TreeBranch, "CCPImageTag:\t"+result.Spec.CCPImageTag)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Status:\t"+result.Spec.BackupStatus)
	fmt.Printf("%s%s\n", TreeBranch, "Backup Host:\t"+result.Spec.BackupHost)
	fmt.Printf("%s%s\n", TreeBranch, "Backup User Secret:\t"+result.Spec.BackupUserSecret)
	fmt.Printf("%s%s\n", TreeTrunk, "Backup Port:\t"+result.Spec.BackupPort)
	fmt.Printf("%s%s\n", TreeTrunk, "Backup Opts:\t"+result.Spec.BackupOpts)

}
