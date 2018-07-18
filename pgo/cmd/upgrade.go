// Package cmd provides the command line functions of the crunchy CLI
package cmd

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/util"
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

const MajorUpgrade = "major"
const MinorUpgrade = "minor"
const SEP = "-"

var UpgradeType string

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Perform an upgrade",
	Long: `UPGRADE performs an upgrade, for example:
				                pgo upgrade mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("upgrade called")
		if len(args) == 0 && Selector == "" {
			fmt.Println(`You must specify the cluster to upgrade or a selector value.`)
		} else {
			err := validateCreateUpdate(args)
			if err != nil {
				log.Error(err.Error())
			} else {
				createUpgrade(args)
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().StringVarP(&UpgradeType, "upgrade-type", "t", "minor", "The upgrade type to perform either minor or major, default is minor ")
	upgradeCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCP_IMAGE_TAG to use for the upgrade target")
}

func showUpgrade(args []string) {
	log.Debugf("showUpgrade called %v\n", args)

	for _, v := range args {

		url := APIServerURL + "/upgrades/" + v + "?version=" + ClientVersion
		log.Debug("showUpgrade called...[" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}

		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)
		StatusCheck(resp)

		defer resp.Body.Close()

		var response msgs.ShowUpgradeResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if response.Status.Code != msgs.Ok {
			log.Error(RED(response.Status.Msg))
			os.Exit(2)
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

func deleteUpgrade(args []string) {
	log.Debugf("deleteUpgrade called %v\n", args)

	for _, v := range args {

		url := APIServerURL + "/upgradesdelete/" + v + "?version=" + ClientVersion
		log.Debug("deleteUpgrade called...[" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}
		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)
		StatusCheck(resp)

		defer resp.Body.Close()

		var response msgs.DeleteUpgradeResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if response.Status.Code == msgs.Ok {
			if len(response.Results) == 0 {
				fmt.Println("no upgrades found")
				return
			}
			for k := range response.Results {
				fmt.Println("deleted upgrade " + response.Results[k])
			}
		} else {
			log.Error(RED(response.Status.Msg))
			os.Exit(2)
		}

	}

}

func validateCreateUpdate(args []string) error {
	var err error

	if UpgradeType == MajorUpgrade {
		if util.AskForConfirmation(NoPrompt, "") {
		} else {
			fmt.Println("Aborting...")
			os.Exit(2)
		}

	}
	if UpgradeType == MajorUpgrade || UpgradeType == MinorUpgrade {
	} else {
		return errors.New("upgrade-type requires either a value of major or minor, if not specified, minor is the default value")
	}
	return err
}

func createUpgrade(args []string) {
	log.Debugf("createUpgrade called %v\n", args)

	if len(args) == 0 && Selector == "" {
		log.Error("cluster names or a selector flag is required")
		os.Exit(2)
	}

	request := msgs.CreateUpgradeRequest{}
	request.Args = args
	request.Selector = Selector
	request.CCPImageTag = CCPImageTag
	request.UpgradeType = UpgradeType
	request.ClientVersion = ClientVersion

	jsonValue, _ := json.Marshal(request)

	url := APIServerURL + "/upgrades"
	log.Debug("createUpgrade called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}
	log.Debugf("%v\n", resp)
	StatusCheck(resp)

	defer resp.Body.Close()

	var response msgs.CreateUpgradeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Error(err)
		log.Println(err)
		return
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		log.Error(RED(response.Status.Msg))
		os.Exit(2)
	}

}
