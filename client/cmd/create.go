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

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Database, Cluster, or Backup",
	Long: `CREATE allows you to create a new Database, Cluster, or Backup
For example:

crunchy create database
crunchy create cluster
crunchy create backup mydatabase
.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create called")
		if len(args) == 0 {
			fmt.Println(`You must specify the type of resource to create.  Valid resource types include:
	* database
	* cluster
	* backup`)
		}
	},
}

var createBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a new backup",
	Long: `Create a backup of a database or cluster
For example:

crunchy create backup mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		createBackup(args)
	},
}

var createDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Create a new database",
	Long: `Create a crunchy database which consists of a Service and Pod
For example:

crunchy create database mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		createDatabase(args)
	},
}

// createClusterCmd represents the create database command
var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create a database cluster",
	Long: `Create a crunchy cluster which consist of a
master and a number of replica backends. For example:

crunchy create cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create cluster called")
		createCluster(args)
	},
}

func init() {
	RootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createDatabaseCmd)
	createCmd.AddCommand(createClusterCmd)
	createCmd.AddCommand(createBackupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
