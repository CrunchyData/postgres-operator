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

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/spf13/cobra"
)

const TreeBranch = "\t"
const TreeTrunk = "\t"

var AllFlag bool

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the description of a cluster",
	Long: `Show allows you to show the details of a policy, backup, pvc, or cluster. For example:

	pgo show backup mycluster
	pgo show backup mycluster --backup-type=pgbackrest
	pgo show cluster mycluster
	pgo show config
	pgo show pgouser someuser
	pgo show policy policy1
	pgo show pvc mycluster
	pgo show namespace
	pgo show workflow 25927091-b343-4017-be4b-71575f0b3eb5
	pgo show user --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the type of resource to show.
Valid resource types include:
	* backup
	* cluster
	* config
	* pgadmin
	* pgbouncer
	* pgouser
	* policy
	* pvc
	* namespace
	* workflow
	* user
	`)
		} else {
			switch args[0] {
			case "backup", "cluster", "config", "pgadmin", "pgbouncer",
				"pgouser", "policy", "pvc", "namespace",
				"workflow", "user":
				break
			default:
				fmt.Println(`Error: You must specify the type of resource to show.
Valid resource types include:
	* backup
	* cluster
	* config
	* pgadmin
	* pgbouncer
	* pgouser
	* policy
	* pvc
	* namespace
	* workflow
	* user`)
			}
		}

	},
}

var showBackupType string

func init() {
	RootCmd.AddCommand(ShowCmd)
	ShowCmd.AddCommand(ShowBackupCmd)
	ShowCmd.AddCommand(ShowClusterCmd)
	ShowCmd.AddCommand(ShowConfigCmd)
	ShowCmd.AddCommand(ShowNamespaceCmd)
	ShowCmd.AddCommand(ShowPgAdminCmd)
	ShowCmd.AddCommand(ShowPgBouncerCmd)
	ShowCmd.AddCommand(ShowPgouserCmd)
	ShowCmd.AddCommand(ShowPgoroleCmd)
	ShowCmd.AddCommand(ShowPolicyCmd)
	ShowCmd.AddCommand(ShowPVCCmd)
	ShowCmd.AddCommand(ShowWorkflowCmd)
	ShowCmd.AddCommand(ShowUserCmd)

	ShowBackupCmd.Flags().StringVarP(&showBackupType, "backup-type", "", "pgbackrest", "The backup type output to list. Valid choices are pgbackrest or pgdump.")
	ShowClusterCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "", "", "Filter the results based on the image tag of the cluster.")
	ShowClusterCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", "The output format. Currently, json is the only supported value.")
	ShowClusterCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	ShowNamespaceCmd.Flags().BoolVar(&AllFlag, "all", false, "show all resources.")
	ShowClusterCmd.Flags().BoolVar(&AllFlag, "all", false, "show all resources.")
	ShowPolicyCmd.Flags().BoolVar(&AllFlag, "all", false, "show all resources.")
	ShowPgAdminCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	ShowPgAdminCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", `The output format. Supported types are: "json"`)
	ShowPgBouncerCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	ShowPgBouncerCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", `The output format. Supported types are: "json"`)
	ShowPVCCmd.Flags().BoolVar(&AllFlag, "all", false, "show all resources.")
	ShowUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	ShowUserCmd.Flags().BoolVar(&AllFlag, "all", false, "show all clusters.")
	ShowUserCmd.Flags().IntVarP(&Expired, "expired", "", 0, "Shows passwords that will expire in X days.")
	ShowUserCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", `The output format. Supported types are: "json"`)
	ShowUserCmd.Flags().BoolVar(&ShowSystemAccounts, "show-system-accounts", false, "Include the system accounts in the results.")
	ShowPgouserCmd.Flags().BoolVar(&AllFlag, "all", false, "show all resources.")
	ShowPgoroleCmd.Flags().BoolVar(&AllFlag, "all", false, "show all resources.")
}

var ShowConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show configuration information",
	Long: `Show configuration information for the Operator. For example:

	pgo show config`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		showConfig(args, Namespace)
	},
}

var ShowPgAdminCmd = &cobra.Command{
	Use:   "pgadmin",
	Short: "Show pgadmin deployment information",
	Long: `Show service information about a pgadmin deployment. For example:

	pgo show pgadmin thecluster
	pgo show pgadmin --selector=app=theapp
	`,

	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		showPgAdmin(Namespace, args)
	},
}

var ShowPgBouncerCmd = &cobra.Command{
	Use:   "pgbouncer",
	Short: "Show pgbouncer deployment information",
	Long: `Show user, password, and service information about a pgbouncer deployment. For example:

	pgo show pgbouncer hacluster
	pgo show pgbouncer --selector=app=payment
	`,

	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		showPgBouncer(Namespace, args)
	},
}

var ShowPgouserCmd = &cobra.Command{
	Use:   "pgouser",
	Short: "Show pgouser information",
	Long: `Show pgouser information for an Operator user. For example:

	pgo show pgouser someuser`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		showPgouser(args, Namespace)
	},
}

var ShowPgoroleCmd = &cobra.Command{
	Use:   "pgorole",
	Short: "Show pgorole information",
	Long: `Show pgorole information . For example:

	pgo show pgorole somerole`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		showPgorole(args, Namespace)
	},
}

var ShowNamespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Show namespace information",
	Long: `Show namespace information for the Operator. For example:

	pgo show namespace`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		showNamespace(args)
	},
}

var ShowWorkflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Show workflow information",
	Long: `Show workflow information for a given workflow. For example:

	pgo show workflow 25927091-b343-4017-be4b-71575f0b3eb5`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		showWorkflow(args, Namespace)
	},
}

var ShowPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Show policy information",
	Long: `Show policy information. For example:

	pgo show policy --all
	pgo show policy policy1`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && !AllFlag {
			fmt.Println("Error: Policy name(s) or --all required for this command.")
		} else {
			if Namespace == "" {
				Namespace = PGONamespace
			}
			showPolicy(args, Namespace)
		}
	},
}

var ShowPVCCmd = &cobra.Command{
	Use:   "pvc",
	Short: "Show PVC information for a cluster",
	Long: `Show PVC information. For example:

	pgo show pvc mycluster
	pgo show pvc --all`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && !AllFlag {
			fmt.Println("Error: Cluster name(s) or --all required for this command.")
		} else {
			if Namespace == "" {
				Namespace = PGONamespace
			}
			showPVC(args, Namespace)
		}
	},
}

// showBackupCmd represents the show backup command
var ShowBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Show backup information",
	Long: `Show backup information. For example:

	pgo show backup mycluser`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if len(args) == 0 {
			fmt.Println("Error: cluster name(s) required for this command.")
		} else {
			// default is pgbackrest
			if showBackupType == "" || showBackupType == config.LABEL_BACKUP_TYPE_BACKREST {
				showBackrest(args, Namespace)
			} else if showBackupType == config.LABEL_BACKUP_TYPE_PGDUMP {
				showpgDump(args, Namespace)
			} else {
				fmt.Println("Error: Valid backup-type values are pgbackrest and pgdump. The default if not supplied is pgbackrest.")
			}
		}
	},
}

// ShowClusterCmd represents the show cluster command
var ShowClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Show cluster information",
	Long: `Show a PostgreSQL cluster. For example:

	pgo show cluster --all
	pgo show cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if Selector == "" && len(args) == 0 && !AllFlag {
			fmt.Println("Error: Cluster name(s), --selector, or --all required for this command.")
		} else {
			showCluster(args, Namespace)
		}
	},
}

// ShowUserCmd represents the show user command
var ShowUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Show user information",
	Long: `Show users on a cluster. For example:

	pgo show user --all
	pgo show user mycluster
	pgo show user --selector=name=nycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}
		if Selector == "" && AllFlag == false && len(args) == 0 {
			fmt.Println("Error: --selector, --all, or cluster name()s required for this command")
		} else {
			showUser(args, Namespace)
		}
	},
}
