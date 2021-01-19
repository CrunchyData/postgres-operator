package cmd

/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	"github.com/spf13/cobra"
)

var (
	DeleteLabel bool
)

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
			os.Exit(1)
		}

		if len(UserLabels) == 0 {
			fmt.Println("Error: You must specify the label to apply.")
			os.Exit(1)
		}

		labelClusters(args, Namespace)
	},
}

func init() {
	RootCmd.AddCommand(labelCmd)

	labelCmd.Flags().BoolVar(&DryRun, "dry-run", false, "Shows the clusters that the label would be applied to, without labelling them.")
	labelCmd.Flags().StringSliceVar(&UserLabels, "label", []string{}, "Add labels to apply to the PostgreSQL cluster, "+
		"e.g. \"key=value\", \"prefix/key=value\". Can specify flag multiple times.")
	labelCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
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
	r.Labels = getLabels(UserLabels)
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
	req.Labels = getLabels(UserLabels)
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
