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

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a backup, benchmark, cluster, pgbouncer, pgpool, label, policy, or user",
	Long: `The delete command allows you to delete a backup, benchmark, cluster, label, pgbouncer, pgpool, policy, or user. For example:

	pgo delete backup mycluster
	pgo delete benchmark mycluster
	pgo delete cluster mycluster
	pgo delete cluster mycluster --delete-data
	pgo delete cluster mycluster --delete-data --delete-backups
	pgo delete label mycluster --label=env=research
	pgo delete pgbouncer mycluster
	pgo delete pgpool mycluster
	pgo delete policy mypolicy
	pgo delete schedule --schedule-name=mycluster-pgbackrest-full
	pgo delete schedule --selector=name=mycluster
	pgo delete schedule mycluster
	pgo delete user testuser --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Println(`Error: You must specify the type of resource to delete.  Valid resource types include:
	* backup
	* benchmark
	* cluster
	* label
	* pgbouncer
	* pgpool
	* policy
	* user`)
		} else {
			switch args[0] {
			case "backup":
			case "benchmark":
			case "cluster":
			case "label":
			case "pgbouncer":
			case "pgpool":
			case "policy":
			case "schedule":
			case "user":
				break
			default:
				fmt.Println(`Error: You must specify the type of resource to delete.  Valid resource types include:
	* backup
	* benchmark
	* cluster
	* label
	* pgbouncer
	* pgpool
	* policy
	* user`)
			}
		}

	},
}

var DeleteBackups bool
var NoPrompt bool

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteBackupCmd)
	deleteCmd.AddCommand(deleteBenchmarkCmd)
	deleteCmd.AddCommand(deleteClusterCmd)
	deleteCmd.AddCommand(deletePgbouncerCmd)
	deleteCmd.AddCommand(deletePgpoolCmd)
	deleteCmd.AddCommand(deletePolicyCmd)
	deleteCmd.AddCommand(deleteLabelCmd)
	deleteCmd.AddCommand(deleteScheduleCmd)
	deleteCmd.AddCommand(deleteUserCmd)

	deleteClusterCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	deleteBenchmarkCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	deleteClusterCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	deleteLabelCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	deleteLabelCmd.Flags().StringVarP(&LabelCmdLabel, "label", "", "", "The label to delete for any selected or specified clusters.")
	deleteClusterCmd.Flags().BoolVarP(&DeleteData, "delete-data", "d", false, "Causes the data for this cluster to be removed permanently.")
	deleteClusterCmd.Flags().BoolVar(&AllFlag, "all", false, "all resources.")
	deleteClusterCmd.Flags().BoolVarP(&DeleteBackups, "delete-backups", "b", false, "Causes the backups for this cluster to be removed permanently.")
	deletePgbouncerCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	deletePgbouncerCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	deletePgpoolCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	deletePgpoolCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	deletePolicyCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	deletePolicyCmd.Flags().BoolVar(&AllFlag, "all", false, "all resources.")
	deleteScheduleCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	deleteScheduleCmd.Flags().StringVarP(&ScheduleName, "schedule-name", "", "", "The name of the schedule to delete.")
	deleteScheduleCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	deleteUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	deleteUserCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")

}

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

// deleteUserCmd ...
var deleteUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Delete a user",
	Long: `Delete a user. For example:

    pgo delete user someuser --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 {
			fmt.Println("Error: A user name is required for this command.")
		} else if Selector == "" {
			fmt.Println("Error: A selector is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deleteUser(args[0], Namespace)

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
			if util.AskForConfirmation(NoPrompt, "") {
				deleteCluster(args, Namespace)
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

// deletePgpoolCmd ...
var deletePgpoolCmd = &cobra.Command{
	Use:   "pgpool",
	Short: "Delete a pgpool from a cluster",
	Long: `Delete a pgpool from a cluster. For example:
    
    pgo delete pgpool mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: A cluster name or selector is required for this command.")
		} else {
			if util.AskForConfirmation(NoPrompt, "") {
				deletePgpool(args, Namespace)

			} else {
				fmt.Println("Aborting...")
			}
		}
	},
}

var deleteScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Delete a schedule",
	Long: `Delete a cron-like schedule. For example:

    pgo delete schedule mycluster
    pgo delete schedule --selector=env=test
    pgo delete schedule --schedule-name=mycluster-pgbackrest-full`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 && Selector == "" && ScheduleName == "" {
			fmt.Println("Error: cluster name, schedule name or selector is required to delete a schedule.")
			return
		}

		if util.AskForConfirmation(NoPrompt, "") {
			deleteSchedule(args, Namespace)
		} else {
			fmt.Println("Aborting...")
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

var deleteBenchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Delete benchmarks for a cluster",
	Long: `Delete benchmarks for a cluster. For example:

    pgo delete benchmark mycluster
    pgo delete benchmark --selector=env=test`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: cluster name or selector is required to delete a benchmark.")
			return
		}

		if util.AskForConfirmation(NoPrompt, "") {
			deleteBenchmark(args, Namespace)
		} else {
			fmt.Println("Aborting...")
		}
	},
}
