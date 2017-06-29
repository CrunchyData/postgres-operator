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

var CCP_IMAGE_TAG string
var Password string
var SecretFrom, BackupPath, BackupPVC string
var PolicyFile, PolicyRepo string

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Cluster or Policy",
	Long: `CREATE allows you to create a new Cluster or Policy
For example:

pgo create cluster
pgo create policy
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create called")
		if len(args) == 0 || (args[0] != "cluster" && args[0] != "policy") {
			fmt.Println(`You must specify the type of resource to create.  Valid resource types include:
	* cluster
	* policy`)
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
		err := validateConfigPolicies()
		if err != nil {
			return
		}
		if SecretFrom != "" || BackupPath != "" || BackupPVC != "" {
			if SecretFrom == "" || BackupPath == "" || BackupPVC == "" {
				log.Error("secret-from, backup-path, backup-pvc are all required to perform a restore")
				return
			}
		}

		if len(args) == 0 {
			log.Error("a cluster name is required for this command")
		} else {
			createCluster(args)
		}
	},
}

var createPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Create a policy",
	Long: `Create a policy. For example:
pgo create policy mypolicy --in-file=/tmp/mypolicy.sql`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create policy called")
		if PolicyFile != "" && PolicyRepo != "" {
			log.Error("in-file or repo is required to create a policy")
			return
		}

		if len(args) == 0 {
			log.Error("a policy name is required for this command")
		} else {
			createPolicy(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createClusterCmd)
	createCmd.AddCommand(createPolicyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	createClusterCmd.Flags().StringVarP(&Password, "password", "w", "", "The password to use for initial database users")
	createClusterCmd.Flags().StringVarP(&SecretFrom, "secret-from", "s", "", "The cluster name to use when restoring secrets")
	createClusterCmd.Flags().StringVarP(&BackupPVC, "backup-pvc", "p", "", "The backup archive PVC to restore from")
	createClusterCmd.Flags().StringVarP(&BackupPath, "backup-path", "x", "", "The backup archive path to restore from")
	createClusterCmd.Flags().StringVarP(&CCP_IMAGE_TAG, "ccp-image-tag", "c", "", "The CCP_IMAGE_TAG to use for cluster creation, if specified overrides the .pgo.yaml setting")
	createPolicyCmd.Flags().StringVarP(&PolicyRepo, "repo", "r", "", "The git repo to use for the policy , if specified overrides the .pgo.yaml setting")
	createPolicyCmd.Flags().StringVarP(&PolicyFile, "in-file", "i", "", "The policy file path to use for the policy , if specified overrides the .pgo.yaml setting")

}
