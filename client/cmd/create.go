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

var BackupPath, BackupPVC string

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Database, Cluster, Backup, or Upgrade",
	Long: `CREATE allows you to create a new Database, Cluster, Backup, or Upgrade
For example:

pgo create database
pgo create cluster
pgo create backup mydatabase
pgo create upgrade mydatabase
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create called")
		if len(args) == 0 {
			fmt.Println(`You must specify the type of resource to create.  Valid resource types include:
	* database
	* cluster
	* upgrade
	* backup`)
		}
	},
}

var createUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Create a new upgrade",
	Long: `Create an upgrade of a database or cluster
For example:

pgo create upgrade mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("a database or cluster name is required for this command")
		} else {
			createUpgrade(args)
		}
	},
}

var createBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a new backup",
	Long: `Create a backup of a database or cluster
For example:

pgo create backup mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("a database or cluster name is required for this command")
		} else {
			createBackup(args)
		}
	},
}

var createDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Create a new database",
	Long: `Create a crunchy database which consists of a Service and Pod
For example:

pgo create database mydatabase`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error("a database or cluster name is required for this command")
		} else {
			createDatabase(args)
		}
	},
}

// createClusterCmd represents the create database command
var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create a database cluster",
	Long: `Create a crunchy cluster which consist of a
master and a number of replica backends. For example:

pgo create cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create cluster called")
		if len(args) == 0 {
			log.Error("a cluster name is required for this command")
		} else {
			createCluster(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createDatabaseCmd, createClusterCmd, createBackupCmd, createUpgradeCmd)
	//createCmd.AddCommand(createClusterCmd)
	//createCmd.AddCommand(createBackupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	//createCmd.PersistentFlags().String("from-pvc", "", "The PVC which contains a backup archive to restore from")
	//createCmd.PersistentFlags().String("from-backup", "", "The backup archive path to restore from")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	createDatabaseCmd.Flags().StringVarP(&BackupPVC, "backup-pvc", "p", "", "The backup archive PVC to restore from")
	createDatabaseCmd.Flags().StringVarP(&BackupPath, "backup-path", "x", "", "The backup archive path to restore from")

}
