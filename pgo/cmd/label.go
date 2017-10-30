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
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

var LabelCmdLabel string
var LabelMap map[string]string
var DeleteLabel bool

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "label a set of clusters",
	Long: `LABEL allows you to add or remove a label on a set of clusters
For example:

pgo label mycluster yourcluster --label=environment=prod 
pgo label mycluster yourcluster --label=environment=prod  --delete-label
pgo label --label=environment=prod --selector=name=mycluster
pgo label --label=environment=prod --selector=status=final --dry-run
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("label called")
		if len(args) == 0 && Selector == "" {
			log.Error("selector or list of clusters is required to label a policy")
			return
		}
		if LabelCmdLabel == "" {
			log.Error(`You must specify the label to apply.`)
		} else {
			labelClusters(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(labelCmd)

	labelCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	labelCmd.Flags().StringVarP(&LabelCmdLabel, "label", "l", "", "The new label to apply for any selected or specified clusters")
	labelCmd.Flags().BoolVarP(&DryRun, "dry-run", "d", false, "--dry-run shows clusters that label would be applied to but does not actually label them")
	labelCmd.Flags().BoolVarP(&DeleteLabel, "delete-label", "x", false, "--delete-label deletes a label from matching clusters")

}

func labelClusters(clusters []string) {
	var err error

	if len(clusters) == 0 && Selector == "" {
		fmt.Println("no clusters specified")
		return
	}

	r := new(msgs.LabelRequest)
	r.Args = clusters
	r.Selector = Selector
	r.DryRun = DryRun
	r.Namespace = Namespace
	r.LabelCmdLabel = LabelCmdLabel
	r.DeleteLabel = DeleteLabel

	jsonValue, _ := json.Marshal(r)

	url := APIServerURL + "/label"
	log.Debug("label called...[" + url + "]")

	action := "POST"

	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	//log.Printf("response is %v\n", resp)

	if DryRun {
		fmt.Println("DRY RUN....would have applied label on ...")
	}
	var response msgs.LabelResponse
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
		fmt.Println(RED(response.Status.Msg))
		os.Exit(2)
	}

}
