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
	"github.com/spf13/cobra"
	"os"
)

var LabelCmdLabel string
var LabelMap map[string]string
var DeleteLabel bool

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Label a set of clusters",
	Long: `LABEL allows you to add or remove a label on a set of clusters. For example:

	pgo label mycluster yourcluster --label=environment=prod
	pgo label all --label=environment=prod 
	pgo label --label=environment=prod --selector=name=mycluster
	pgo label --label=environment=prod --selector=status=final --dry-run`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		log.Debug("label called")
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: A selector or list of clusters is required to label a policy.")
			return
		}
		if LabelCmdLabel == "" {
			fmt.Println("Error: You must specify the label to apply.")
		} else {
			labelClusters(args, Namespace)
		}
	},
}

func init() {
	RootCmd.AddCommand(labelCmd)

	labelCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	labelCmd.Flags().StringVarP(&LabelCmdLabel, "label", "", "", "The new label to apply for any selected or specified clusters.")
	labelCmd.Flags().BoolVarP(&DryRun, "dry-run", "", false, "Shows the clusters that the label would be applied to, without labelling them.")

}

func labelClusters(clusters []string, ns string) {
	var err error

	if len(clusters) == 0 && Selector == "" {
		fmt.Println("No clusters specified.")
		return
	}

	r := new(msgs.LabelRequest)
	r.Args = clusters
	r.Namespace = ns
	r.Selector = Selector
	r.DryRun = DryRun
	r.LabelCmdLabel = LabelCmdLabel
	r.DeleteLabel = DeleteLabel
	r.ClientVersion = msgs.PGO_VERSION

	log.Debugf("%s is the selector", r.Selector)
	response, err := api.LabelClusters(httpclient, &SessionCredentials, r)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if DryRun {
		fmt.Println("The label would have been applied on the following:")
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println("Label applied on " + response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}

// deleteLabel ...
func deleteLabel(args []string, ns string) {
	log.Debugf("deleteLabel called %v", args)

	req := msgs.DeleteLabelRequest{}
	req.Selector = Selector
	req.Namespace = ns
	req.Args = args
	req.LabelCmdLabel = LabelCmdLabel
	req.ClientVersion = msgs.PGO_VERSION

	response, err := api.DeleteLabel(httpclient, &SessionCredentials, &req)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for _, result := range response.Results {
			fmt.Println(result)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
	}

}
