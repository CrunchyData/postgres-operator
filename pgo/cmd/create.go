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
var PoliciesFlag, PolicyFile, PolicyURL string
var NodeName string
var UserLabels string
var UserLabelsMap map[string]string
var Series int

var CreateCmd = &cobra.Command{
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
		var err error
		log.Debug("create cluster called")
		err = validateConfigPolicies()
		if err != nil {
			return
		}
		if SecretFrom != "" || BackupPath != "" || BackupPVC != "" {
			if SecretFrom == "" || BackupPath == "" || BackupPVC == "" {
				log.Error("secret-from, backup-path, backup-pvc are all required to perform a restore")
				return
			}
		}

		//always have a valid NodeName
		if NodeName == "" {
			//NodeName = getValidNodeName()
		} else {
			err = validateNodeName(NodeName)
			if err != nil {
				log.Error(err)
				return
			}

		}

		if UserLabels != "" {
			err = validateUserLabels()
			if err != nil {
				log.Error("invalid user labels, check --labels value")
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
		if PolicyFile != "" && PolicyURL != "" {
			log.Error("--in-file or --url is required to create a policy")
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
	RootCmd.AddCommand(CreateCmd)
	CreateCmd.AddCommand(createClusterCmd)
	CreateCmd.AddCommand(createPolicyCmd)

	createClusterCmd.Flags().StringVarP(&NodeName, "node-name", "n", "", "The node on which to place the master database")
	createClusterCmd.Flags().StringVarP(&Password, "password", "w", "", "The password to use for initial database users")
	createClusterCmd.Flags().StringVarP(&SecretFrom, "secret-from", "s", "", "The cluster name to use when restoring secrets")
	createClusterCmd.Flags().StringVarP(&BackupPVC, "backup-pvc", "p", "", "The backup archive PVC to restore from")
	createClusterCmd.Flags().StringVarP(&UserLabels, "labels", "l", "", "The labels to apply to this cluster")
	createClusterCmd.Flags().StringVarP(&BackupPath, "backup-path", "x", "", "The backup archive path to restore from")
	createClusterCmd.Flags().StringVarP(&PoliciesFlag, "policies", "z", "", "The policies to apply when creating a cluster, comma separated")
	createClusterCmd.Flags().StringVarP(&CCP_IMAGE_TAG, "ccp-image-tag", "c", "", "The CCP_IMAGE_TAG to use for cluster creation, if specified overrides the .pgo.yaml setting")
	createClusterCmd.Flags().IntVarP(&Series, "series", "e", 1, "The number of clusters to create in a series, defaults to 1")
	createPolicyCmd.Flags().StringVarP(&PolicyURL, "url", "u", "", "The url to use for adding a policy")
	createPolicyCmd.Flags().StringVarP(&PolicyFile, "in-file", "i", "", "The policy file path to use for adding a policy")
	UserLabelsMap = make(map[string]string)

}
