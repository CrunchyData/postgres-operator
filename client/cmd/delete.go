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
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a database, cluster, or backup",
	Long: `delete allows you to delete a database, cluster, or backup
For example:

pgo delete database mydatabase
pgo delete cluster mycluster
pgo delete backup mycluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			fmt.Println(`You must specify the type of resource to delete.  Valid resource types include:
	* database
	* cluster
	* backup`)
		} else {
			switch args[0] {
			case "database":
			case "cluster":
			case "backup":
				break
			default:
				fmt.Println(`You must specify the type of resource to delete.  Valid resource types include: 
	* database
	* cluster
	* backup`)
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteDatabaseCmd)
	deleteCmd.AddCommand(deleteClusterCmd)
	deleteCmd.AddCommand(deleteBackupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deleteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deleteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

var deleteBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "delete a backup",
	Long: `delete a backup. For example:
	pgo delete backup mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		deleteBackup(args)
	},
}

var deleteDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "delete a database",
	Long: `delete a crunchy database. For example:
	pgo delete database mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		deleteDatabase(args)
	},
}

var deleteClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "delete a cluster",
	Long: `delete a crunchy cluster. For example:
	pgo delete cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		deleteCluster(args)
	},
}
