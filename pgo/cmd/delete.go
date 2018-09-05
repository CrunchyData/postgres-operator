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
	"fmt"
	"github.com/crunchydata/postgres-operator/pgo/util"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a user, policy, cluster, backup, or upgrade",
	Long: `The delete command allows you to delete a user, policy, cluster, backup, or upgrade. For example:

	pgo delete user testuser --selector=name=mycluster
	pgo delete policy mypolicy
	pgo delete cluster mycluster
	pgo delete cluster mycluster --delete-data
	pgo delete cluster mycluster --delete-data --delete-backups
	pgo delete backup mycluster
	pgo delete upgrade mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Println(`Error: You must specify the type of resource to delete.  Valid resource types include:
	* policy
	* user
	* cluster
	* backup
	* ingest
	* upgrade`)
		} else {
			switch args[0] {
			case "policy":
			case "user":
			case "cluster":
			case "backup":
			case "ingest":
			case "upgrade":
				break
			default:
				fmt.Println(`Error: You must specify the type of resource to delete.  Valid resource types include:
	* policy
	* user
	* cluster
	* backup
	* ingest
	* upgrade`)
			}
		}

	},
}

var DeleteBackups bool
var NoPrompt bool

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteUserCmd)
	deleteCmd.AddCommand(deletePolicyCmd)
	deleteCmd.AddCommand(deleteIngestCmd)
	deleteCmd.AddCommand(deleteClusterCmd)
	deletePolicyCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "No command line confirmation.")
	deleteClusterCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "No command line confirmation.")
	deleteClusterCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	deleteUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	deleteClusterCmd.Flags().BoolVarP(&DeleteData, "delete-data", "d", false, "Causes the data for this cluster to be removed permanently.")
	deleteClusterCmd.Flags().BoolVarP(&DeleteBackups, "delete-backups", "b", false, "Causes the backups for this cluster to be removed permanently.")
	deleteCmd.AddCommand(deleteBackupCmd)
	deleteCmd.AddCommand(deleteUpgradeCmd)

}

var deleteIngestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Delete an ingest",
	Long: `Delete an ingest. For example:
<<<<<<< HEAD
=======

>>>>>>> pre32
	pgo delete ingest myingest`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Error: A ingest name is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deleteIngest(args)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

var deleteUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Delete an upgrade",
	Long: `Delete an upgrade. For example:
<<<<<<< HEAD
=======

>>>>>>> pre32
	pgo delete upgrade mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Error: A database or cluster name is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deleteUpgrade(args)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

var deleteBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Delete a backup",
	Long: `Delete a backup. For example:
<<<<<<< HEAD
=======

>>>>>>> pre32
	pgo delete backup mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Error: A database or cluster name is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deleteBackup(args)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

// deleteUserCmd ...
var deleteUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Delete a user",
	Long: `Delete a user. For example:
<<<<<<< HEAD
=======

>>>>>>> pre32
	pgo delete user someuser --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Println("Error: A user name is required for this command.")
		} else if Selector == "" {
			fmt.Println("Error: A selector is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deleteUser(args[0])

			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

// deleteClusterCmd ...
var deleteClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Delete a PostgreSQL cluster",
	Long: `Delete a PostgreSQL cluster. For example:

	pgo delete cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: A cluster name or selector is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deleteCluster(args)

			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

var deletePolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Delete a SQL policy",
	Long: `Delete a policy. For example:

	pgo delete policy mypolicy`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Error: A policy name is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deletePolicy(args)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}
