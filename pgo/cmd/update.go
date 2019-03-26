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
	"github.com/crunchydata/postgres-operator/pgo/util"
	"github.com/spf13/cobra"
	"os"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a cluster",
	Long: `The update command allows you to update a cluster. For example:

	pgo update cluster mycluster --autofail=false
	pgo update cluster mycluster --autofail=true`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Println(`Error: You must specify the type of resource to update.  Valid resource types include:
	* cluster`)
		} else {
			switch args[0] {
			case "cluster":
				break
			default:
				fmt.Println(`Error: You must specify the type of resource to update.  Valid resource types include:
	* cluster`)
			}
		}

	},
}

var AutofailStringFlag string

func init() {
	RootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateClusterCmd)

	updateClusterCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	updateClusterCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	updateClusterCmd.Flags().StringVarP(&AutofailStringFlag, "autofail", "", "", "If set, will cause the autofail label on the pgcluster CRD for this cluster to be updated to either true or false, valid values are true or false.")

}

// updateClusterCmd ...
var updateClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Update a PostgreSQL cluster",
	Long: `Update a PostgreSQL cluster. For example:

    pgo update cluster all --autofail=false
    pgo update cluster mycluster --autofail=true`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if AutofailStringFlag == "true" || AutofailStringFlag == "false" {
		} else {
			fmt.Println("Error: --autofail=true or --autofail=false is required.")
			os.Exit(2)
		}

		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: A cluster name or selector is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				updateCluster(args, Namespace)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}
