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

	"github.com/crunchydata/postgres-operator/cmd/pgo/util"
	"github.com/spf13/cobra"
)

// Several prompts to help a user decide if they wish to actually delete their
// data from a cluster
const (
	deleteClusterAllPromptMessage         = "This will delete ALL OF YOUR DATA, including backups. Proceed?"
	deleteClusterKeepBackupsPromptMessage = "This will delete your active data, but your backups will still be available. Proceed?"
	deleteClusterKeepDataPromptMessage    = "This will delete your cluster as well as your backups, but your data is still accessible if you recreate the cluster. Proceed?"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an Operator resource",
	Long: `The delete command allows you to delete an Operator resource. For example:

	pgo delete backup mycluster
	pgo delete cluster mycluster
	pgo delete cluster mycluster --delete-data
	pgo delete cluster mycluster --delete-data --delete-backups
	pgo delete label mycluster --label=env=research
	pgo delete pgadmin mycluster
	pgo delete pgbouncer mycluster
	pgo delete pgbouncer mycluster --uninstall
	pgo delete pgouser someuser
	pgo delete pgorole somerole
	pgo delete policy mypolicy
	pgo delete namespace mynamespace
	pgo delete user --username=testuser --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Println(`Error: You must specify the type of resource to delete.  Valid resource types include:
	* backup
	* cluster
	* label
	* pgadmin
	* pgbouncer
	* pgouser
	* pgorole
	* namespace
	* policy
	* user`)
		} else {
			switch args[0] {
			case "backup",
				"cluster",
				"label",
				"pgadmin",
				"pgbouncer",
				"pgouser",
				"pgorole",
				"policy",
				"namespace",
				"user":
				break
			default:
				fmt.Println(`Error: You must specify the type of resource to delete.  Valid resource types include:
	* backup
	* cluster
	* label
	* pgadmin
	* pgbouncer
	* pgouser
	* pgorole
	* policy
	* namespace
	* user`)
			}
		}

	},
}

// DEPRECATED deleteBackups, if set to "true", indicates that backups can be deleted.
var deleteBackups bool

// KeepBackups, If set to "true", indicates that backups should be stored even
// after a cluster is deleted
var KeepBackups bool

// NoPrompt, If set to "true", indicates that the user should not be prompted
// before executing a delete command
var NoPrompt bool

// initialize variables specific for the "pgo delete" command and subcommands
func init() {
	// set the various commands that are provided by this file
	// First, add the root command, i.e. "pgo delete"
	RootCmd.AddCommand(deleteCmd)

	// "pgo delete backup"
	// used to delete backups
	deleteCmd.AddCommand(deleteBackupCmd)

	// "pgo delete cluster"
	// used to delete clusters
	deleteCmd.AddCommand(deleteClusterCmd)
	// "pgo delete cluster --all"
	// allows for the deletion of every cluster.
	deleteClusterCmd.Flags().BoolVar(&AllFlag, "all", false,
		"Delete all clusters. Backups and data subject to --delete-backups and --delete-data flags, respectively.")
	// "pgo delete cluster --delete-backups"
	// "pgo delete cluster -d"
	// instructs that any backups associated with a cluster should be deleted
	deleteClusterCmd.Flags().BoolVarP(&deleteBackups, "delete-backups", "b", false,
		"Causes the backups for specified cluster to be removed permanently.")
	deleteClusterCmd.Flags().MarkDeprecated("delete-backups",
		"Backups are deleted by default. If you would like to keep your backups, use the --keep-backups flag")
	// "pgo delete cluster --delete-data"
	// "pgo delete cluster -d"
	// instructs that the PostgreSQL cluster data should be deleted
	deleteClusterCmd.Flags().BoolVarP(&DeleteData, "delete-data", "d", false,
		"Causes the data for specified cluster to be removed permanently.")
	deleteClusterCmd.Flags().MarkDeprecated("delete-data",
		"Data is deleted by default. You can preserve your data by keeping your backups with the --keep-backups flag")
	// "pgo delete cluster --keep-backups"
	// instructs that any backups associated with a cluster should be kept and not deleted
	deleteClusterCmd.Flags().BoolVar(&KeepBackups, "keep-backups", false,
		"Keeps the backups available for use at a later time (e.g. recreating the cluster).")
	// "pgo delete cluster --keep-data"
	// instructs that any data associated with the cluster should be kept and not deleted
	deleteClusterCmd.Flags().BoolVar(&KeepData, "keep-data", false,
		"Keeps the data for the specified cluster. Can be reassigned to exact same cluster in the future.")
	// "pgo delete cluster --no-prompt"
	// does not display the warning prompt to ensure the user wishes to delete
	// a cluster
	deleteClusterCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false,
		"No command line confirmation before delete.")
	// "pgo delete cluster --selector"
	// "pgo delete cluster -s"
	// the selector flag that filters which clusters to delete
	deleteClusterCmd.Flags().StringVarP(&Selector, "selector", "s", "",
		"The selector to use for cluster filtering.")

	// "pgo delete label"
	// delete a cluster label
	deleteCmd.AddCommand(deleteLabelCmd)
	// pgo delete label --label
	// the label to be deleted
	deleteLabelCmd.Flags().StringVar(&LabelCmdLabel, "label", "",
		"The label to delete for any selected or specified clusters.")
	// "pgo delete label --selector"
	// "pgo delete label -s"
	// the selector flag that filters which clusters to delete the cluster
	// labels from
	deleteLabelCmd.Flags().StringVarP(&Selector, "selector", "s", "",
		"The selector to use for cluster filtering.")

	// "pgo delete namespace"
	// deletes a namespace and all of the objects within it (clusters, etc.)
	deleteCmd.AddCommand(deleteNamespaceCmd)

	// "pgo delete pgadmin"
	// delete a pgAdmin instance associated with a PostgreSQL cluster
	deleteCmd.AddCommand(deletePgAdminCmd)
	// "pgo delete pgadmin --no-prompt"
	// does not display the warning prompt to confirming pgAdmin deletion
	deletePgAdminCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation before delete.")
	// "pgo delete pgadmin --selector"
	// "pgo delete pgadmin -s"
	// the selector flag filtering clusters from which to delete the pgAdmin instances
	deletePgAdminCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")

	// "pgo delete pgbouncer"
	// delete a pgBouncer instance that is associated with a PostgreSQL cluster
	deleteCmd.AddCommand(deletePgbouncerCmd)
	// "pgo delete pgbouncer --no-prompt"
	// does not display the warning prompt to ensure the user wishes to delete
	// a pgBouncer instance
	deletePgbouncerCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation before delete.")
	// "pgo delete pgbouncer --selector"
	// "pgo delete pgbouncer -s"
	// the selector flag that filters which clusters to delete the pgBouncer
	// instances from
	deletePgbouncerCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	// "pgo delete pgbouncer --uninstall"
	// this flag removes all of the pgbouncer machinery that is installed in the
	// PostgreSQL cluster
	deletePgbouncerCmd.Flags().BoolVar(&PgBouncerUninstall, "uninstall", false, `Used to remove any "pgbouncer" owned object and user from the PostgreSQL cluster`)

	// "pgo delete pgorole"
	// delete a role that is able to issue commands interface with the
	// PostgreSQL Operator
	deleteCmd.AddCommand(deletePgoroleCmd)
	// "pgo delete pgorole --all"
	// allows for the deletion of every PostgreSQL Operator role.
	deletePgoroleCmd.Flags().BoolVar(&AllFlag, "all", false, "Delete all PostgreSQL Operator roles.")
	// "pgo delete pgorole --no-prompt"
	// does not display the warning prompt to ensure the user wishes to delete
	// a PostgreSQL Operator role
	deletePgoroleCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false,
		"No command line confirmation before delete.")

	// "pgo delete pgouser"
	// delete a user that is able to issue commands to the PostgreSQL Operator
	deleteCmd.AddCommand(deletePgouserCmd)
	// "pgo delete cluster --all"
	// allows for the deletion of every PostgreSQL Operator user.
	deletePgouserCmd.Flags().BoolVar(&AllFlag, "all", false,
		"Delete all PostgreSQL Operator users.")
	// "pgo delete pgouser --no-prompt"
	// does not display the warning prompt to ensure the user wishes to delete
	// a PostgreSQL Operator user
	deletePgouserCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false,
		"No command line confirmation before delete.")

	// "pgo delete policy"
	// delete a SQL policy associated with a PostgreSQL cluster
	deleteCmd.AddCommand(deletePolicyCmd)
	// "pgo delete policy --all"
	// delete all SQL policies for all clusters
	deletePolicyCmd.Flags().BoolVar(&AllFlag, "all", false, "Delete all SQL policies.")
	// "pgo delete policy --no-prompt"
	// does not display the warning prompt to ensure the user wishes to delete
	// a SQL policy
	deletePolicyCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation before delete.")

	// "pgo delete user"
	// Delete a user from a PostgreSQL cluster
	deleteCmd.AddCommand(deleteUserCmd)
	// "pgo delete user --all"
	// delete all users from all PostgreSQL clusteres
	deleteUserCmd.Flags().BoolVar(&AllFlag, "all", false,
		"Delete all PostgreSQL users from all clusters.")
	// "pgo delete user --no-prompt"
	// does not display the warning prompt to ensure the user wishes to delete
	// the PostgreSQL user from the cluster
	deleteUserCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false,
		"No command line confirmation before delete.")
	// "pgo delete user -o"
	// selects the type of output to use, choices are "json" and any other input
	// will render the text based one
	deleteUserCmd.Flags().StringVarP(&OutputFormat, "output", "o", "",
		`The output format. Supported types are: "json"`)
	// the selector flag that filters which PostgreSQL users should be deleted
	// from which clusters
	deleteUserCmd.Flags().StringVarP(&Selector, "selector", "s", "",
		"The selector to use for cluster filtering.")
	// the username of the PostgreSQL user to delete
	deleteUserCmd.Flags().StringVar(&Username, "username", "",
		"The username to delete.")
}

// deleteBackupCmd ...
var deleteBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Delete a backup",
	Long: `Delete a backup. For example:

    pgo delete backup mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 {
			fmt.Println("Error: A database or cluster name is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deleteBackup(args, Namespace)
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

    pgo delete cluster --all
    pgo delete cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && Selector == "" && !AllFlag {
			fmt.Println("Error: A cluster name,  selector, or --all is required for this command.")
		} else {
			// Set the prompt message based on whether or not "--keep-backups" is set
			promptMessage := deleteClusterAllPromptMessage

			if KeepBackups && KeepData {
				promptMessage = ""
			} else if KeepBackups {
				promptMessage = deleteClusterKeepBackupsPromptMessage
			} else if KeepData {
				promptMessage = deleteClusterKeepDataPromptMessage
			}

			if util.AskForConfirmation(NoPrompt, promptMessage) {
				deleteCluster(args, Namespace)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

// deleteLabelCmd ...
var deleteLabelCmd = &cobra.Command{
	Use:   "label",
	Short: "Delete a label from clusters",
	Long: `Delete a label from clusters. For example:

    pgo delete label mycluster --label=env=research
    pgo delete label all --label=env=research
    pgo delete label --selector=group=southwest --label=env=research`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: A cluster name or selector is required for this command.")
		} else {
			deleteLabel(args, Namespace)
		}
	},
}

// deleteNamespaceCmd ...
var deleteNamespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Delete namespaces",
	Long: `Delete namespaces. For example:

    pgo delete namespace mynamespace
    pgo delete namespace --selector=env=test`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: namespace name or selector is required to delete a namespace.")
			return
		}

		if util.AskForConfirmation(NoPrompt, "") {
			deleteNamespace(args, Namespace)
		} else {
			fmt.Println("Aborting...")
		}
	},
}

// deletePgoroleCmd ...
var deletePgoroleCmd = &cobra.Command{
	Use:   "pgorole",
	Short: "Delete a pgorole",
	Long: `Delete a pgorole. For example:

    pgo delete pgorole somerole`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 {
			fmt.Println("Error: A pgorole role name is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deletePgorole(args, Namespace)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

// deletePgouserCmd ...
var deletePgouserCmd = &cobra.Command{
	Use:   "pgouser",
	Short: "Delete a pgouser",
	Long: `Delete a pgouser. For example:

    pgo delete pgouser someuser`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 {
			fmt.Println("Error: A pgouser username is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deletePgouser(args, Namespace)
			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

// deletePgAdminCmd ...
var deletePgAdminCmd = &cobra.Command{
	Use:   "pgadmin",
	Short: "Delete a pgAdmin instance from a cluster",
	Long: `Delete a pgAdmin instance from a cluster. For example:

	pgo delete pgadmin mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: A cluster name or selector is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deletePgAdmin(args, Namespace)

			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

// deletePgbouncerCmd ...
var deletePgbouncerCmd = &cobra.Command{
	Use:   "pgbouncer",
	Short: "Delete a pgbouncer from a cluster",
	Long: `Delete a pgbouncer from a cluster. For example:

	pgo delete pgbouncer mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: A cluster name or selector is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deletePgbouncer(args, Namespace)

			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

// deletePolicyCmd ...
var deletePolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Delete a SQL policy",
	Long: `Delete a policy. For example:

    pgo delete policy mypolicy`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && !AllFlag {
			fmt.Println("Error: A policy name or --all is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deletePolicy(args, Namespace)
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

    pgo delete user --username=someuser --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && AllFlag == false && Selector == "" {
			fmt.Println("Error: --all, --selector, or a list of clusters is required for this command")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deleteUser(args, Namespace)

			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}
