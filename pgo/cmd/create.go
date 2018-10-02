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
var BackrestFlag, ArchiveFlag, AutofailFlag, PgpoolFlag, PgbouncerFlag, MetricsFlag, BadgerFlag bool
var PgpoolSecret string
var PgbouncerSecret string
var CCPImageTag string
var Password string
var SecretFrom, BackupPath, BackupPVC string
var PoliciesFlag, PolicyFile, PolicyURL string
var NodeLabel string
var UserLabels string
var IngestConfig string
var ServiceType string
var Schedule string
var ScheduleOptions string
var ScheduleType string
var PGBackRestType string
var Secret string

var Series int

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Cluster, Ingest, PGBouncer, PGPool, Policy, Schedule, or User",
	Long: `CREATE allows you to create a new Cluster, Ingest, PGBouncer, PGPool, Policy, Schedule or User. For example: 

    pgo create cluster
    pgo create ingest
    pgo create pgbouncer
    pgo create pgpool
    pgo create policy
    pgo create user`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create called")
		if len(args) == 0 || (args[0] != "cluster" && args[0] != "policy") && args[0] != "user" {
			fmt.Println(`Error: You must specify the type of resource to create.  Valid resource types include:
    * cluster
    * ingest
    * pgbouncer
    * pgpool
    * policy
    * user`)
		}
	},
}

// createClusterCmd ...
var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create a PostgreSQL cluster",
	Long: `Create a PostgreSQL cluster consisting of a primary and a number of replica backends. For example:

    pgo create cluster mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create cluster called")
		if BackupPath != "" || BackupPVC != "" {
			if SecretFrom == "" || BackupPath == "" || BackupPVC == "" {
				fmt.Println(`Error: The --secret-from, --backup-path, and --backup-pvc flags are all required to perform a restore.`)
				return
			}
		}

		if len(args) == 0 {
			fmt.Println(`Error: A cluster name is required for this command.`)
		} else {
			createCluster(args)
		}
	},
}

// createPolicyCmd ...
var createPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Create a SQL policy",
	Long: `Create a policy. For example:

    pgo create policy mypolicy --in-file=/tmp/mypolicy.sql`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create policy called ")
		if PolicyFile == "" && PolicyURL == "" {
			//log.Error("--in-file or --url is required to create a policy")
			fmt.Println(`Error: The --in-file or --url flags are required to create a policy.`)
			return
		}

		if len(args) == 0 {
			fmt.Println(`Error: A policy name is required for this command.`)
		} else {
			createPolicy(args)
		}
	},
}

// createPgbouncerCmd ...
var createPgbouncerCmd = &cobra.Command{
	Use:   "pgbouncer",
	Short: "Create a pgbouncer ",
	Long: `Create a pgbouncer. For example:

	pgo create pgbouncer mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create pgbouncer called ")

		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: A cluster name or selector is required for this command.`)
		} else {
			createPgbouncer(args)
		}
	},
}

// createPgpoolCmd ...
var createPgpoolCmd = &cobra.Command{
	Use:   "pgpool",
	Short: "Create a pgpool ",
	Long: `Create a pgpool. For example:

    pgo create pgpool mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create pgpool called ")

		if len(args) == 0 && Selector == "" {
			fmt.Println(`Error: A cluster name or selector is required for this command.`)
		} else {
			createPgpool(args)
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
			fmt.Println(`Error: An ingest name is required for this command.`)
		} else {
			if IngestConfig == "" {
				fmt.Println("Error: You must specify the ingest-config flag.")
				return
			}
			createIngest(args)
		}
	},
}

// createScheduleCmd ...
var createScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Create a cron-like scheduled task",
	Long: `SCHEDULE creates a cron-like schedule task.  For example:

    pgo create schedule --schedule=”* * * * *” --selector=env=research --schedule-type=pgbackrest`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create schedule called ")
		if len(args) == 0 && Selector == "" {
			fmt.Println("Error: The --selector flag or a cluster name is required to create a schedule.")
			return
		}
		createSchedule(args)
	},
}

// createUserCmd ...
var createUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Create a PostgreSQL user",
	Long: `Create a postgres user. For example:

    pgo create user user1 --selector=name=mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("create user called ")
		if Selector == "" {
			fmt.Println(`Error: The --selector flag is required to create a user.`)
			return
		}

		if len(args) == 0 {
			fmt.Println(`Error: A user name is required for this command.`)
		} else {
			createUser(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(CreateCmd)
	CreateCmd.AddCommand(createClusterCmd)
	CreateCmd.AddCommand(createIngestCmd)
	CreateCmd.AddCommand(createPolicyCmd)
	CreateCmd.AddCommand(createPgbouncerCmd)
	CreateCmd.AddCommand(createPgpoolCmd)
	//CreateCmd.AddCommand(createScheduleCmd)
	CreateCmd.AddCommand(createUserCmd)

	createClusterCmd.Flags().BoolVarP(&BackrestFlag, "pgbackrest", "", false, "Enables a pgBackRest volume for the database pod.")
	createClusterCmd.Flags().BoolVarP(&BadgerFlag, "pgbadger", "", false, "Adds the crunchy-pgbadger container to the database pod.")
	createClusterCmd.Flags().BoolVarP(&PgpoolFlag, "pgpool", "", false, "Adds the crunchy-pgpool container to the database pod.")
	createClusterCmd.Flags().BoolVarP(&PgbouncerFlag, "pgbouncer", "", false, "Adds a crunchy-pgbouncer deployment to the cluster.")
	createClusterCmd.Flags().BoolVarP(&ArchiveFlag, "archive", "", false, "Enables archive logging for the database cluster.")
	createClusterCmd.Flags().StringVarP(&PgpoolSecret, "pgpool-secret", "", "", "The name of a pgpool secret to use for the pgpool configuration.")
	createClusterCmd.Flags().BoolVarP(&MetricsFlag, "metrics", "", false, "Adds the crunchy-collect container to the database pod.")
	createClusterCmd.Flags().BoolVarP(&AutofailFlag, "autofail", "", false, "If set, will cause autofailover to be enabled on this cluster.")
	createClusterCmd.Flags().StringVarP(&CustomConfig, "custom-config", "", "", "The name of a configMap that holds custom PostgreSQL configuration files used to override defaults.")
	createClusterCmd.Flags().StringVarP(&StorageConfig, "storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster storage.")
	createClusterCmd.Flags().StringVarP(&ReplicaStorageConfig, "replica-storage-config", "", "", "The name of a Storage config in pgo.yaml to use for the cluster replica storage.")
	createClusterCmd.Flags().StringVarP(&NodeLabel, "node-label", "", "", "The node label (key=value) to use in placing the primary database. If not set, any node is used.")
	createClusterCmd.Flags().StringVarP(&ServiceType, "service-type", "", "", "The Service type to use for the PostgreSQL cluster. If not set, the pgo.yaml default will be used.")
	createClusterCmd.Flags().StringVarP(&Password, "password", "w", "", "The password to use for initial database users.")
	createClusterCmd.Flags().StringVarP(&SecretFrom, "secret-from", "s", "", "The cluster name to use when restoring secrets.")
	createClusterCmd.Flags().StringVarP(&BackupPVC, "backup-pvc", "p", "", "The backup archive PVC to restore from.")
	createClusterCmd.Flags().StringVarP(&UserLabels, "labels", "l", "", "The labels to apply to this cluster.")
	createClusterCmd.Flags().StringVarP(&BackupPath, "backup-path", "x", "", "The backup archive path to restore from.")
	createClusterCmd.Flags().StringVarP(&PoliciesFlag, "policies", "z", "", "The policies to apply when creating a cluster, comma separated.")
	createClusterCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.")
	createClusterCmd.Flags().IntVarP(&Series, "series", "e", 1, "The number of clusters to create in a series.")
	createClusterCmd.Flags().StringVarP(&ContainerResources, "resources-config", "r", "", "The name of a container resource configuration in pgo.yaml that holds CPU and memory requests and limits.")

	createIngestCmd.Flags().StringVarP(&IngestConfig, "ingest-config", "i", "", "Defines the path of an ingest configuration file.")
	createPolicyCmd.Flags().StringVarP(&PolicyURL, "url", "u", "", "The url to use for adding a policy.")
	createPolicyCmd.Flags().StringVarP(&PolicyFile, "in-file", "i", "", "The policy file path to use for adding a policy.")

	createScheduleCmd.Flags().StringVarP(&Schedule, "schedule", "", "", "The schedule assigned to the cron task.")
	createScheduleCmd.Flags().StringVarP(&ScheduleType, "schedule-type", "", "", "The type of schedule to be created (pgbackrest or pgbasebackup).")
	createScheduleCmd.Flags().StringVarP(&PGBackRestType, "pgbackrest-backup-type", "", "", "The type of pgBackRest backup to schedule (full or diff).")
	createScheduleCmd.Flags().StringVarP(&PVCName, "pvc-name", "", "", "The name of the backup PVC to use (only used in pgbasebackup schedules).")
	createScheduleCmd.Flags().StringVarP(&CCPImageTag, "ccp-image-tag", "c", "", "The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.")
	createScheduleCmd.Flags().StringVarP(&ScheduleOptions, "schedule-opts", "", "", "The custom options passed to the create schedule API.")
	createScheduleCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	createScheduleCmd.Flags().BoolVarP(&NoPrompt, "no-prompt", "n", false, "No command line confirmation.")

	createUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	createUserCmd.Flags().StringVarP(&Password, "password", "", "", "The password to use for creating a new user which overrides a generated password.")
	createUserCmd.Flags().BoolVarP(&ManagedUser, "managed", "", false, "Creates a user with secrets that can be managed by the Operator.")
	createUserCmd.Flags().StringVarP(&UserDBAccess, "db", "", "", "Grants the user access to a database.")
	createUserCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "", 30, "Sets passwords for new users to X days.")
}
