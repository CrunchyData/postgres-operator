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

var ContainerResources string
var ReplicaStorageConfig, StorageConfig string
var CustomConfig string
var ArchiveFlag, AutofailFlag, PgpoolFlag, MetricsFlag bool
var PgpoolSecret string
var CCPImageTag string
var Password string
var SecretFrom, BackupPath, BackupPVC string
var PoliciesFlag, PolicyFile, PolicyURL string
var NodeLabel string
var UserLabels string
var IngestConfig string
var ServiceType string

//var UserLabelsMap map[string]string
var Series int

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Cluster, Policy, or User",
	Long: `CREATE allows you to create a new Cluster, Policy, User
For example:

pgo create cluster
pgo create policy
pgo create user
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create called")
		if len(args) == 0 || (args[0] != "cluster" && args[0] != "policy") && args[0] != "user" {
			fmt.Println(`You must specify the type of resource to create.  Valid resource types include:
	* cluster
	* user
	* policy`)
		}
	},
}

// createClusterCmd ...
var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create a database cluster",
	Long: `Create a crunchy cluster which consist of a
primary and a number of replica backends. For example:

pgo create cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create cluster called")
		if SecretFrom != "" || BackupPath != "" || BackupPVC != "" {
			if SecretFrom == "" || BackupPath == "" || BackupPVC == "" {
				log.Error("secret-from, backup-path, backup-pvc are all required to perform a restore")
				return
			}
		}

		if UserLabels != "" {
			/**
			err = validateUserLabels()
			if err != nil {
				log.Error("invalid user labels, check --labels value")
				return
			}
			*/
		}

		if len(args) == 0 {
			log.Error("a cluster name is required for this command")
		} else {
			createCluster(args)
		}
	},
}

// createPolicyCmd ...
var createPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Create a policy",
	Long: `Create a policy. For example:
pgo create policy mypolicy --in-file=/tmp/mypolicy.sql`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create policy called ")
		if PolicyFile == "" && PolicyURL == "" {
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

// createIngestCmd ...
var createIngestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Create an ingest",
	Long: `Create an ingest. For example:
pgo create ingest myingest --ingest-config=./ingest.json`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create ingest called ")

		if len(args) == 0 {
			log.Error("an ingest name is required for this command")
		} else {
			if IngestConfig == "" {
				fmt.Println("You must specify the ingest-config flag")
				return
			}
			createIngest(args)
		}
	},
}

// createUserCmd ...
var createUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Create a postgres user",
	Long: `Create a postgres user. For example:
pgo create user user1 --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create user called ")
		if Selector == "" {
			log.Error("--selector is required to create a user")
			return
		}

		if len(args) == 0 {
			log.Error("a user name is required for this command")
		} else {
			createUser(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(CreateCmd)
	CreateCmd.AddCommand(createClusterCmd)
	CreateCmd.AddCommand(createPolicyCmd)
	CreateCmd.AddCommand(createIngestCmd)
	CreateCmd.AddCommand(createUserCmd)

	createIngestCmd.Flags().StringVarP(&IngestConfig, "ingest-config", "i", "", "The path of an ingest configuration file")
	createClusterCmd.Flags().BoolVarP(&PgpoolFlag, "pgpool", "", false, "If set, will cause the crunchy-pgpool container to be added to the database cluster")
	createClusterCmd.Flags().BoolVarP(&ArchiveFlag, "archive", "", false, "If set, will cause archive logging to be enabled for the database cluster")
	createClusterCmd.Flags().StringVarP(&PgpoolSecret, "pgpool-secret", "", "", "The name of a pgpool secret to use for the pgpool configuration.")
	createClusterCmd.Flags().BoolVarP(&MetricsFlag, "metrics", "m", false, "If set, will cause the crunchy-collect container to be added to the database pod")
	createClusterCmd.Flags().BoolVarP(&AutofailFlag, "autofail", "", false, "If set, will cause the autofailover to be enabled on this cluster")
	createClusterCmd.Flags().StringVarP(&CustomConfig, "custom-config", "g", "", "The name of a configMap that holds custom PG config files used to override the defaults")
	createClusterCmd.Flags().StringVarP(&StorageConfig, "storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster storage.")
	createClusterCmd.Flags().StringVarP(&ReplicaStorageConfig, "replica-storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster replica storage.")
	createClusterCmd.Flags().StringVarP(&NodeLabel, "node-label", "", "", "The node label (key=value) to use in placing the primary database, if not set any node is used")
	createClusterCmd.Flags().StringVarP(&ServiceType, "service-type", "", "", "The service type to use in the Service for the PG cluster, if not set the pgo.yaml default will be used.")
	createClusterCmd.Flags().StringVarP(&Password, "password", "w", "", "The password to use for initial database users")
	createClusterCmd.Flags().StringVarP(&SecretFrom, "secret-from", "s", "", "The cluster name to use when restoring secrets")
	createClusterCmd.Flags().StringVarP(&BackupPVC, "backup-pvc", "p", "", "The backup archive PVC to restore from")
	createClusterCmd.Flags().StringVarP(&UserLabels, "labels", "l", "", "The labels to apply to this cluster")
	createClusterCmd.Flags().StringVarP(&BackupPath, "backup-path", "x", "", "The backup archive path to restore from")
	createClusterCmd.Flags().StringVarP(&PoliciesFlag, "policies", "z", "", "The policies to apply when creating a cluster, comma separated")
	createClusterCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCPImageTag to use for cluster creation, if specified overrides the .pgo.yaml setting")
	createClusterCmd.Flags().IntVarP(&Series, "series", "e", 1, "The number of clusters to create in a series, defaults to 1")
	createClusterCmd.Flags().StringVarP(&ContainerResources, "resources-config", "r", "", "The name of a container resource configuration in pgo.yaml that holds CPU and memory requests and limits")
	createPolicyCmd.Flags().StringVarP(&PolicyURL, "url", "u", "", "The url to use for adding a policy")
	createPolicyCmd.Flags().StringVarP(&PolicyFile, "in-file", "i", "", "The policy file path to use for adding a policy")
	createUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to filter on clusters")
	createUserCmd.Flags().BoolVarP(&ManagedUser, "managed", "m", false, "--managed creates a user with secrets")
	createUserCmd.Flags().StringVarP(&UserDBAccess, "db", "b", "", "--db=userdb grants the user access to a database")
	createUserCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "v", 30, "--valid-days=7 sets passwords for new users to 7 days")

	//UserLabelsMap = make(map[string]string)

}
