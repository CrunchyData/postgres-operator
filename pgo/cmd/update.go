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
)

func init() {
	RootCmd.AddCommand(UpdateCmd)
	UpdateCmd.AddCommand(UpdatePgouserCmd)
	UpdateCmd.AddCommand(UpdateClusterCmd)

	UpdateClusterCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	UpdateClusterCmd.Flags().BoolVar(&AllFlag, "all", false, "all resources.")
	UpdateClusterCmd.Flags().BoolVar(&AutofailFlag, "autofail", false, "autofail default is false.")
	UpdateClusterCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	UpdatePgouserCmd.Flags().StringVarP(&PgouserPassword, "password", "", "", "The password to use for updating the pgouser password.")
	UpdatePgouserCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	UpdatePgouserCmd.Flags().BoolVar(&PgouserChangePassword, "change-password", false, "change password (default is false).")

}

// UpdateCmd represents the update command
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a pgouser or cluster",
	Long: `The update command allows you to update a pgouser or cluster. For example:

	pgo update pgouser someuser --change-password
	pgo update cluster --selector=name=mycluster --autofail=false
	pgo update cluster --all --autofail=true`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Println(`Error: You must specify the type of resource to update.  Valid resource types include:
	* pgouser
	* cluster`)
		} else {
			switch args[0] {
			case "cluster", "pgouser":
				break
			default:
				fmt.Println(`Error: You must specify the type of resource to update.  Valid resource types include:
	* cluster
	* pgouser`)
			}
		}

	},
}

var PgouserPassword string
var PgouserChangePassword bool

// UpdateClusterCmd ...
var UpdateClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Update a PostgreSQL cluster",
	Long: `Update a PostgreSQL cluster. For example:

    pgo update cluster mycluster --autofail=false
    pgo update cluster mycluster myothercluster --autofail=false
    pgo update cluster --selector=name=mycluster --autofail=false
    pgo update cluster --all --autofail=true`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}

		if len(args) == 0 && Selector == "" && !AllFlag {
			fmt.Println("Error: A cluster name(s) or selector or --all is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				updateCluster(args, Namespace)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

var UpdatePgouserCmd = &cobra.Command{
	Use:   "pgouser",
	Short: "Update a pgouser",
	Long: `UPDATE allows you to update a pgo user. For example:
		pgo update pgouser myuser --change-password
		pgo update pgouser myuser --change-password --password=somepassword --no-prompt`,
	Run: func(cmd *cobra.Command, args []string) {

		if Namespace == "" {
			Namespace = PGONamespace
		}

		if len(args) == 0 {
			fmt.Println("Error: You must specify the name of a pgouser.")
		} else {
			updatePgouser(args, Namespace)
		}
	},
}
