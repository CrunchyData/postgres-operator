// Package cmd provides the command line functions of the crunchy CLI
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
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"net/http"
)

const MajorUpgrade = "major"
const MinorUpgrade = "minor"
const SEP = "-"

var UpgradeType string

func showUpgrade(args []string) {
	log.Debugf("showUpgrade called %v\n", args)

	if Namespace == "" {
		log.Error("Namespace can not be empty")
		return
	}

	for _, v := range args {

		url := APIServerURL + "/upgrades/" + v + "?namespace=" + Namespace
		log.Debug("showPolicy called...[" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}

		client := &http.Client{}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}

		defer resp.Body.Close()

		var response msgs.ShowUpgradeResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if len(response.UpgradeList.Items) == 0 {
			fmt.Println("no upgrades found")
			return
		}

		log.Debugf("response = %v\n", response)
		for _, upgrade := range response.UpgradeList.Items {
			showUpgradeItem(&upgrade)
		}

	}

}

func showUpgradeItem(upgrade *crv1.Pgupgrade) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgupgrade : "+upgrade.Spec.Name)
	fmt.Printf("%s%s\n", TreeBranch, "upgrade_status : "+upgrade.Spec.UpgradeStatus)
	fmt.Printf("%s%s\n", TreeBranch, "resource_type : "+upgrade.Spec.ResourceType)
	fmt.Printf("%s%s\n", TreeBranch, "upgrade_type : "+upgrade.Spec.UpgradeType)
	fmt.Printf("%s%s\n", TreeBranch, "pvc_access_mode : "+upgrade.Spec.StorageSpec.AccessMode)
	fmt.Printf("%s%s\n", TreeBranch, "pvc_size : "+upgrade.Spec.StorageSpec.Size)
	fmt.Printf("%s%s\n", TreeBranch, "ccp_image_tag : "+upgrade.Spec.CCPImageTag)
	fmt.Printf("%s%s\n", TreeBranch, "old_database_name : "+upgrade.Spec.OldDatabaseName)
	fmt.Printf("%s%s\n", TreeBranch, "new_database_name : "+upgrade.Spec.NewDatabaseName)
	fmt.Printf("%s%s\n", TreeBranch, "old_version : "+upgrade.Spec.OldVersion)
	fmt.Printf("%s%s\n", TreeBranch, "new_version : "+upgrade.Spec.NewVersion)
	fmt.Printf("%s%s\n", TreeBranch, "old_pvc_name : "+upgrade.Spec.OldPVCName)
	fmt.Printf("%s%s\n", TreeTrunk, "new_pvc_name : "+upgrade.Spec.NewPVCName)

	fmt.Println("")

}
