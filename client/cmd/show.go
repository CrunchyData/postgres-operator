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
package cmd

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

const TREE_BRANCH = "├── "
const TREE_TRUNK = "└── "

// ShowCmd represents the show command
var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "show a description of a database or cluster",
	Long: `show allows you to show the details of a database, backup, pvc, or cluster.
For example:

	pgo show database mydatabase
	pgo show pvc mypvc
	pgo show backup mydatabase
	pgo show cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println(`You must specify the type of resource to show.  
Valid resource types include:
	* database
	* cluster
	* pvc
	* upgrade
	* backup`)
		} else {
			switch args[0] {
			case "database":
			case "cluster":
			case "pvc":
			case "upgrade":
			case "backup":
				break
			default:
				fmt.Println(`You must specify the type of resource to show.  
Valid resource types include:
	* database
	* cluster
	* pvc
	* upgrade
	* backup`)
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(ShowCmd)
	ShowCmd.AddCommand(ShowDatabaseCmd)
	ShowCmd.AddCommand(ShowClusterCmd)
	ShowCmd.AddCommand(ShowBackupCmd)
	ShowCmd.AddCommand(ShowPVCCmd)
	ShowCmd.AddCommand(ShowUpgradeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ShowCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ShowCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

var ShowPVCCmd = &cobra.Command{
	Use:   "pvc",
	Short: "Show pvc information",
	Long: `Show pvc information. For example:

				pgo show pvc mydatabase`,
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

				pgo show upgrade mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("database or cluster name(s) required for this command")
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

				pgo show backup mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("database or cluster name(s) required for this command")
		} else {
			showBackup(args)
		}
	},
}

// showDatbaseCmd represents the show database command
var ShowDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Show database information",
	Long: `Show a crunchy database. For example:

				pgo show database mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("database name(s) required for this command")
		} else {
			showDatabase(args)
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
		if len(args) == 0 {
			log.Error("cluster name(s) required for this command")
		} else {
			showCluster(args)
		}
	},
}
