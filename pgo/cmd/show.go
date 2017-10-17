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
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

// TreeBranch represents the command line branch display char
const TreeBranch = "├── "

// TreeTrunk represents the command line tree trunk display char
const TreeTrunk = "└── "

// PostgresVersion postgres version flag
var PostgresVersion string

// ShowPVC show pvc flag
var ShowPVC bool

// ShowSecrets show secrets flag
var ShowSecrets bool

// PVCRoot pvc root flag
var PVCRoot string

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show a description of a cluster",
	Long: `show allows you to show the details of a policy, backup, pvc, or cluster.
For example:

	pgo show policy policy1
	pgo show pvc mypvc
	pgo show backup mycluster
	pgo show cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println(`You must specify the type of resource to show.  
Valid resource types include:
	* cluster
	* pvc
	* policy
	* upgrade
	* backup`)
		} else {
			switch args[0] {
			case "cluster":
			case "pvc":
			case "policy":
			case "upgrade":
			case "backup":
				break
			default:
				fmt.Println(`You must specify the type of resource to show.  
Valid resource types include:
	* cluster
	* pvc
	* policy
	* upgrade
	* backup`)
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(showCmd)
	showCmd.AddCommand(showClusterCmd)
	showCmd.AddCommand(showBackupCmd)
	showCmd.AddCommand(showPolicyCmd)
	showCmd.AddCommand(showPVCCmd)
	showCmd.AddCommand(showUpgradeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// showCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// showCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	showClusterCmd.Flags().BoolVarP(&ShowSecrets, "show-secrets", "s", false, "Show secrets ")
	showClusterCmd.Flags().StringVarP(&PostgresVersion, "version", "v", "", "The postgres version to filter on")
	showPVCCmd.Flags().StringVarP(&PVCRoot, "pvc-root", "r", "", "The PVC directory to list")

	showBackupCmd.Flags().BoolVarP(&ShowPVC, "show-pvc", "p", false, "Show backup archive PVC listing ")

}

var showPolicyCmd = &cobra.Command{
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

var showPVCCmd = &cobra.Command{
	Use:   "pvc",
	Short: "Show pvc information",
	Long: `Show pvc information. For example:

				pgo show pvc mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("PVC name(s) required for this command")
		} else {
			showPVC(args)
		}
	},
}

var showUpgradeCmd = &cobra.Command{
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
var showBackupCmd = &cobra.Command{
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

var showClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Show cluster information",
	Long: `Show a crunchy cluster. For example:

				pgo show cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if Labelselector == "" && len(args) == 0 {
			log.Error("cluster name(s) required for this command")
		} else {
			showCluster(args)
		}
	},
}
