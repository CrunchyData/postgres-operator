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

var Password string
var BackupPath, BackupPVC string

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Cluster",
	Long: `CREATE allows you to create a new Cluster
For example:

pgo create cluster
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create called")
		if len(args) == 0 || args[0] != "cluster" {
			fmt.Println(`You must specify the type of resource to create.  Valid resource types include:
	* cluster`)
		}
	},
}

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
	createCmd.AddCommand(createClusterCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	//createCmd.PersistentFlags().String("from-pvc", "", "The PVC which contains a backup archive to restore from")
	//createCmd.PersistentFlags().String("from-backup", "", "The backup archive path to restore from")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	createClusterCmd.Flags().StringVarP(&Password, "password", "w", "", "The password to use for initial database users")
	createClusterCmd.Flags().StringVarP(&BackupPVC, "backup-pvc", "p", "", "The backup archive PVC to restore from")
	createClusterCmd.Flags().StringVarP(&BackupPath, "backup-path", "x", "", "The backup archive path to restore from")

}
