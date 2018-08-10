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
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

const TreeBranch = "\t"
const TreeTrunk = "\t"

var PostgresVersion string
var ShowPVC bool
var PVCRoot string

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show a description of a cluster",
	Long: `Show allows you to show the details of a policy, backup, pvc, or cluster.
For example:

	pgo show policy policy1
	pgo show pvc mycluster
	pgo show backup mycluster
	pgo show backrest mycluster
	pgo show ingest myingest
	pgo show config
	pgo show user mycluster
	pgo show cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println(`You must specify the type of resource to show.  
Valid resource types include:
	* cluster
	* pvc
	* policy
	* backrest
	* user
	* ingest
	* config
	* upgrade
	* backup`)
		} else {
			switch args[0] {
			case "cluster":
			case "pvc":
			case "backrest":
			case "policy":
			case "ingest":
			case "user":
			case "config":
			case "upgrade":
			case "backup":
				break
			default:
				fmt.Println(`You must specify the type of resource to show.  
Valid resource types include:
	* cluster
	* pvc
	* backrest
	* policy
	* ingest
	* user
	* config
	* upgrade
	* backup`)
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(ShowCmd)
	ShowCmd.AddCommand(ShowClusterCmd)
	ShowCmd.AddCommand(ShowBackupCmd)
	ShowCmd.AddCommand(ShowBackrestCmd)
	ShowCmd.AddCommand(ShowPolicyCmd)
	ShowCmd.AddCommand(ShowPVCCmd)
	ShowCmd.AddCommand(ShowIngestCmd)
	ShowCmd.AddCommand(ShowConfigCmd)
	ShowCmd.AddCommand(ShowUpgradeCmd)
	ShowCmd.AddCommand(ShowUserCmd)

	ShowClusterCmd.Flags().StringVarP(&PostgresVersion, "version", "v", "", "The postgres version to filter on")
	ShowClusterCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	ShowBackrestCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	ShowUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	ShowPVCCmd.Flags().StringVarP(&PVCRoot, "pvc-root", "r", "", "The PVC directory to list")
	ShowClusterCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", "The output format, json is currently supported")

}

var ShowConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show config information",
	Long: `Show config information. For example:

				pgo show config`,
	Run: func(cmd *cobra.Command, args []string) {
		showConfig(args)
	},
}

var ShowPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Show policy information",
	Long: `Show policy information. For example:

				pgo show policy policy1`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("policy name(s) required for this command")
		} else {
			showPolicy(args)
		}
	},
}

var ShowPVCCmd = &cobra.Command{
	Use:   "pvc",
	Short: "Show pvc information",
	Long: `Show pvc information. For example:

				pgo show pvc all
				pgo show pvc mycluster-backup
				pgo show pvc mycluster-xlog
				pgo show pvc mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("PVC name(s) required for this command")
		} else {
			showPVC(args)
		}
	},
}

var ShowUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Show upgrade information",
	Long: `Show upgrade information. For example:

				pgo show upgrade mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("cluster name(s) required for this command")
		} else {
			showUpgrade(args)
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
		if len(args) == 0 {
			log.Error("cluster name(s) required for this command")
		} else {
			showBackup(args)
		}
	},
}

// showBackrestCmd represents the show backrest command
var ShowBackrestCmd = &cobra.Command{
	Use:   "backrest",
	Short: "Show backrest information",
	Long: `Show backrest information. For example:

				pgo show backrest mycluser`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("cluster name(s) required for this command")
		} else {
			showBackrest(args)
		}
	},
}

// ShowClusterCmd represents the show cluster command
var ShowClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Show cluster information",
	Long: `Show a crunchy cluster. For example:

				pgo show cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Selector == "" && len(args) == 0 {
			log.Error("cluster name(s) required for this command")
		} else {
			showCluster(args)
		}
	},
}

// ShowIngestCmd represents the show ingest command
var ShowIngestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Show ingest information",
	Long: `Show a crunchy ingest. For example:

				pgo show ingest myingest`,
	Run: func(cmd *cobra.Command, args []string) {
		if Selector == "" && len(args) == 0 {
			log.Error("ingest name(s) required for this command")
		} else {
			showIngest(args)
		}
	},
}

// ShowUserCmd represents the show user command
var ShowUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Show user information",
	Long: `Show users on a cluster. For example:

				pgo show user mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Selector == "" && len(args) == 0 {
			log.Error("cluster name(s) required for this command")
		} else {
			showUser(args)
		}
	},
}
