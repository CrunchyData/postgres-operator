package cmd

/*
 Copyright 2017 - 2023 Crunchy Data Solutions, Inc.
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
	"os"

	"github.com/crunchydata/postgres-operator/cmd/pgo/util"
	"github.com/spf13/cobra"
)

const pgBouncerPrompt = "This may cause an interruption in your pgBouncer service. Are you sure you wish to proceed?"

var (
	// DisableLogin allows a user to disable the ability for a PostgreSQL uesr to
	// log in
	DisableLogin bool
	// DisableMetrics allows a user to disable metrics collection
	DisableMetrics bool
	// DisablePGBadger allows a user to disable pgBadger
	DisablePGBadger bool
	// DisableTLS will disable TLS in a cluster
	DisableTLS bool
	// DisableTLSOnly will disable TLS enforcement in the cluster
	DisableTLSOnly bool
	// EnableLogin allows a user to enable the ability for a PostgreSQL uesr to
	// log in
	EnableLogin bool
	// EnableMetrics allows a user to enbale metrics collection
	EnableMetrics bool
	// EnablePGBadger allows a user to enbale pgBadger
	EnablePGBadger bool
	// EnableTLSOnly will enable TLS enforcement in the cluster
	EnableTLSOnly bool
	// ExpireUser sets a user to having their password expired
	ExpireUser bool
	// ExporterRotatePassword rotates the password for the designed PostgreSQL
	// user for handling metrics scraping
	ExporterRotatePassword bool
	// PgoroleChangePermissions does something with the pgouser access controls,
	// I'm not sure but I wanted this at least to be documented
	PgoroleChangePermissions bool
	// RotatePassword is a flag that allows one to specify that a password be
	// automatically rotated, such as a service account type password
	RotatePassword bool
	// DisableStandby can be used to disable standby mode when enabled in an existing cluster
	DisableStandby bool
	// EnableStandby can be used to enable standby mode in an existing cluster
	EnableStandby bool
	// Shutdown is used to indicate that the cluster should be shutdown
	Shutdown bool
	// Startup is used to indicate that the cluster should be started (assuming it is shutdown)
	Startup bool
)

func init() {
	RootCmd.AddCommand(UpdateCmd)
	UpdateCmd.AddCommand(UpdatePgBouncerCmd)
	UpdateCmd.AddCommand(UpdatePgouserCmd)
	UpdateCmd.AddCommand(UpdatePgoroleCmd)
	UpdateCmd.AddCommand(UpdateClusterCmd)
	UpdateCmd.AddCommand(UpdateUserCmd)
	UpdateCmd.AddCommand(UpdateNamespaceCmd)

	UpdateClusterCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	UpdateClusterCmd.Flags().BoolVar(&AllFlag, "all", false, "all resources.")
	UpdateClusterCmd.Flags().StringSliceVar(&Annotations, "annotation", []string{},
		"Add an Annotation to all of the managed deployments (PostgreSQL, pgBackRest, pgBouncer)\n"+
			"The format to add an annotation is \"name=value\"\n"+
			"The format to remove an annotation is \"name-\"\n\n"+
			"For example, to add two annotations: \"--annotation=hippo=awesome,elephant=cool\"")
	UpdateClusterCmd.Flags().StringSliceVar(&AnnotationsBackrest, "annotation-pgbackrest", []string{},
		"Add an Annotation specifically to pgBackRest deployments\n"+
			"The format to add an annotation is \"name=value\"\n"+
			"The format to remove an annotation is \"name-\"")
	UpdateClusterCmd.Flags().StringSliceVar(&AnnotationsPgBouncer, "annotation-pgbouncer", []string{},
		"Add an Annotation specifically to pgBouncer deployments\n"+
			"The format to add an annotation is \"name=value\"\n"+
			"The format to remove an annotation is \"name-\"")
	UpdateClusterCmd.Flags().StringSliceVar(&AnnotationsPostgres, "annotation-postgres", []string{},
		"Add an Annotation specifically to PostgreSQL deployments"+
			"The format to add an annotation is \"name=value\"\n"+
			"The format to remove an annotation is \"name-\"")
	UpdateClusterCmd.Flags().StringVar(&CPURequest, "cpu", "", "Set the number of millicores to request for the CPU, e.g. "+
		"\"100m\" or \"0.1\".")
	UpdateClusterCmd.Flags().StringVar(&CPULimit, "cpu-limit", "", "Set the number of millicores to limit for the CPU, e.g. "+
		"\"100m\" or \"0.1\".")
	UpdateClusterCmd.Flags().BoolVar(&DisableAutofailFlag, "disable-autofail", false, "Disables autofail capabitilies in the cluster.")
	UpdateClusterCmd.Flags().BoolVar(&DisableMetrics, "disable-metrics", false,
		"Disable the metrics collection sidecar. May cause brief downtime.")
	UpdateClusterCmd.Flags().BoolVar(&DisablePGBadger, "disable-pgbadger", false,
		"Disable the pgBadger sidecar. May cause brief downtime.")
	UpdateClusterCmd.Flags().BoolVar(&DisableTLS, "disable-server-tls", false, "Remove TLS from the cluster.")
	UpdateClusterCmd.Flags().BoolVar(&DisableTLSOnly, "disable-tls-only", false, "Remove TLS enforcement for the cluster.")
	UpdateClusterCmd.Flags().BoolVar(&EnableAutofailFlag, "enable-autofail", false, "Enables autofail capabitilies in the cluster.")
	UpdateClusterCmd.Flags().StringVar(&MemoryRequest, "memory", "", "Set the amount of RAM to request, e.g. "+
		"1GiB.")
	UpdateClusterCmd.Flags().StringVar(&MemoryLimit, "memory-limit", "", "Set the amount of RAM to limit, e.g. "+
		"1GiB.")
	UpdateClusterCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	UpdateClusterCmd.Flags().BoolVarP(&DisableStandby, "promote-standby", "", false,
		"Disables standby mode (if enabled) and promotes the cluster(s) specified.")
	UpdateClusterCmd.Flags().StringVar(&BackrestCPURequest, "pgbackrest-cpu", "", "Set the number of millicores to request for CPU "+
		"for the pgBackRest repository.")
	UpdateClusterCmd.Flags().StringVar(&BackrestCPULimit, "pgbackrest-cpu-limit", "", "Set the number of millicores to limit for CPU "+
		"for the pgBackRest repository.")
	UpdateClusterCmd.Flags().StringVar(&BackrestMemoryRequest, "pgbackrest-memory", "", "Set the amount of memory to request for "+
		"the pgBackRest repository.")
	UpdateClusterCmd.Flags().StringVar(&BackrestMemoryLimit, "pgbackrest-memory-limit", "", "Set the amount of memory to limit for "+
		"the pgBackRest repository.")
	UpdateClusterCmd.Flags().StringVar(&BackrestPVCSize, "pgbackrest-pvc-size", "",
		`The size of the PVC capacity for the pgBackRest repository. Overrides the value set in the storage class. This is ignored if the storage type of "posix" is not used. Must follow the standard Kubernetes format, e.g. "10.1Gi"`)
	UpdateClusterCmd.Flags().StringVar(&PVCSize, "pvc-size", "",
		`The size of the PVC capacity for primary and replica PostgreSQL instances. Must follow the standard Kubernetes format, e.g. "10.1Gi"`)
	UpdateClusterCmd.Flags().StringVar(&WALPVCSize, "wal-pvc-size", "",
		`The size of the capacity for WAL storage, which overrides any value in the storage configuration.  Must follow the standard Kubernetes format, e.g. "10.1Gi".`)
	UpdateClusterCmd.Flags().StringVar(&ExporterCPURequest, "exporter-cpu", "", "Set the number of millicores to request for CPU "+
		"for the Crunchy Postgres Exporter sidecar container, e.g. \"100m\" or \"0.1\".")
	UpdateClusterCmd.Flags().StringVar(&ExporterCPULimit, "exporter-cpu-limit", "", "Set the number of millicores to limit for CPU "+
		"for the Crunchy Postgres Exporter sidecar container, e.g. \"100m\" or \"0.1\".")
	UpdateClusterCmd.Flags().StringVar(&ExporterMemoryRequest, "exporter-memory", "", "Set the amount of memory to request for "+
		"the Crunchy Postgres Exporter sidecar container.")
	UpdateClusterCmd.Flags().StringVar(&ExporterMemoryLimit, "exporter-memory-limit", "", "Set the amount of memory to limit for "+
		"the Crunchy Postgres Exporter sidecar container.")
	UpdateClusterCmd.Flags().BoolVar(&EnableMetrics, "enable-metrics", false,
		"Enable the metrics collection sidecar. May cause brief downtime.")
	UpdateClusterCmd.Flags().BoolVar(&EnablePGBadger, "enable-pgbadger", false,
		"Enable the pgBadger sidecar. May cause brief downtime.")
	UpdateClusterCmd.Flags().BoolVar(&ExporterRotatePassword, "exporter-rotate-password", false, "Used to rotate the password for the metrics collection agent.")
	UpdateClusterCmd.Flags().BoolVarP(&EnableStandby, "enable-standby", "", false,
		"Enables standby mode in the cluster(s) specified.")
	UpdateClusterCmd.Flags().BoolVar(&EnableTLSOnly, "enable-tls-only", false, "Enforce TLS on the cluster.")
	UpdateClusterCmd.Flags().StringVar(&ReplicationTLSSecret, "replication-tls-secret", "", "The name of the secret that contains "+
		"the TLS keypair to use for enabling certificate-based authentication between PostgreSQL instances, "+
		"particularly for the purpose of replication. TLS must be enabled in the cluster.")
	UpdateClusterCmd.Flags().StringVar(&ServiceType, "service-type", "", "The Service type to use for the PostgreSQL cluster. If not set, the pgo.yaml default will be used.")
	UpdateClusterCmd.Flags().BoolVar(&Startup, "startup", false, "Restart the database cluster if it "+
		"is currently shutdown.")
	UpdateClusterCmd.Flags().BoolVar(&Shutdown, "shutdown", false, "Shutdown the database "+
		"cluster if it is currently running.")
	UpdateClusterCmd.Flags().StringSliceVar(&Tablespaces, "tablespace", []string{},
		"Add a PostgreSQL tablespace on the cluster, e.g. \"name=ts1:storageconfig=nfsstorage\". The format is "+
			"a key/value map that is delimited by \"=\" and separated by \":\". The following parameters are available:\n\n"+
			"- name (required): the name of the PostgreSQL tablespace\n"+
			"- storageconfig (required): the storage configuration to use, as specified in the list available in the "+
			"\"pgo-config\" ConfigMap (aka \"pgo.yaml\")\n"+
			"- pvcsize: the size of the PVC capacity, which overrides the value set in the specified storageconfig. "+
			"Follows the Kubernetes quantity format.\n\n"+
			"For example, to create a tablespace with the NFS storage configuration with a PVC of size 10GiB:\n\n"+
			"--tablespace=name=ts1:storageconfig=nfsstorage:pvcsize=10Gi")
	UpdateClusterCmd.Flags().StringVar(&CASecret, "server-ca-secret", "", "The name of the secret that contains "+
		"the certficate authority (CA) to use for enabling the PostgreSQL cluster to accept TLS connections. "+
		"Must be used with \"server-tls-secret\".")
	UpdateClusterCmd.Flags().StringVar(&TLSSecret, "server-tls-secret", "", "The name of the secret that contains "+
		"the TLS keypair to use for enabling the PostgreSQL cluster to accept TLS connections. "+
		"Must be used with \"server-ca-secret\"")
	UpdateClusterCmd.Flags().StringSliceVar(&Tolerations, "toleration", []string{},
		"Set Pod tolerations for each PostgreSQL instance in a cluster.\n"+
			"The general format is \"key=value:Effect\"\n"+
			"For example, to add an Exists and an Equals toleration: \"--toleration=ssd:NoSchedule,zone=east:NoSchedule\"\n"+
			"A toleration can be removed by adding a \"-\" to the end, for example:\n"+
			"--toleration=ssd:NoSchedule-")
	UpdatePgBouncerCmd.Flags().StringVar(&PgBouncerCPURequest, "cpu", "", "Set the number of millicores to request for CPU "+
		"for pgBouncer.")
	UpdatePgBouncerCmd.Flags().StringVar(&PgBouncerCPULimit, "cpu-limit", "", "Set the number of millicores to limit for CPU "+
		"for pgBouncer.")
	UpdatePgBouncerCmd.Flags().StringVar(&PgBouncerMemoryRequest, "memory", "", "Set the amount of memory to request for "+
		"pgBouncer.")
	UpdatePgBouncerCmd.Flags().StringVar(&PgBouncerMemoryLimit, "memory-limit", "", "Set the amount of memory to limit for "+
		"pgBouncer.")
	UpdatePgBouncerCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	UpdatePgBouncerCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", `The output format. Supported types are: "json"`)
	UpdatePgBouncerCmd.Flags().Int32Var(&PgBouncerReplicas, "replicas", 0, "Set the total number of pgBouncer instances to deploy. If not set, defaults to 1.")
	UpdatePgBouncerCmd.Flags().BoolVar(&RotatePassword, "rotate-password", false, "Used to rotate the pgBouncer service account password. Can cause interruption of service.")
	UpdatePgBouncerCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	UpdatePgBouncerCmd.Flags().StringVar(&ServiceType, "service-type", "", "The Service type to use for pgBouncer.")
	UpdatePgouserCmd.Flags().StringVarP(&PgouserNamespaces, "pgouser-namespaces", "", "", "The namespaces to use for updating the pgouser roles.")
	UpdatePgouserCmd.Flags().BoolVar(&AllNamespaces, "all-namespaces", false, "all namespaces.")
	UpdatePgouserCmd.Flags().StringVarP(&PgouserRoles, "pgouser-roles", "", "", "The roles to use for updating the pgouser roles.")
	UpdatePgouserCmd.Flags().StringVarP(&PgouserPassword, "pgouser-password", "", "", "The password to use for updating the pgouser password.")
	UpdatePgouserCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	UpdatePgoroleCmd.Flags().StringVarP(&Permissions, "permissions", "", "", "The permissions to use for updating the pgorole permissions.")
	UpdatePgoroleCmd.Flags().BoolVar(&NoPrompt, "no-prompt", false, "No command line confirmation.")
	// pgo update user -- flags
	UpdateUserCmd.Flags().BoolVar(&AllFlag, "all", false, "all clusters.")
	UpdateUserCmd.Flags().BoolVar(&DisableLogin, "disable-login", false, "Disables a PostgreSQL user from being able to log into the PostgreSQL cluster.")
	UpdateUserCmd.Flags().BoolVar(&EnableLogin, "enable-login", false, "Enables a PostgreSQL user to be able to log into the PostgreSQL cluster.")
	UpdateUserCmd.Flags().IntVarP(&Expired, "expired", "", 0, "Updates passwords that will expire in X days using an autogenerated password.")
	UpdateUserCmd.Flags().BoolVarP(&ExpireUser, "expire-user", "", false, "Performs expiring a user if set to true.")
	UpdateUserCmd.Flags().IntVarP(&PasswordAgeDays, "valid-days", "", 0, "Sets the number of days that a password is valid. Defaults to the server value.")
	UpdateUserCmd.Flags().StringVarP(&Username, "username", "", "", "Updates the postgres user on selective clusters.")
	UpdateUserCmd.Flags().StringVarP(&OutputFormat, "output", "o", "", `The output format. Supported types are: "json"`)
	UpdateUserCmd.Flags().StringVarP(&Password, "password", "", "", "Specifies the user password when updating a user password or creating a new user. If --rotate-password is set as well, --password takes precedence.")
	UpdateUserCmd.Flags().IntVarP(&PasswordLength, "password-length", "", 0, "If no password is supplied, sets the length of the automatically generated password. Defaults to the value set on the server.")
	UpdateUserCmd.Flags().StringVar(&PasswordType, "password-type", "", "The type of password hashing to use."+
		"Choices are: (md5, scram-sha-256). This only takes effect if the password is being changed.")
	UpdateUserCmd.Flags().BoolVar(&PasswordValidAlways, "valid-always", false, "Sets a password to never expire based on expiration time. Takes precedence over --valid-days")
	UpdateUserCmd.Flags().BoolVar(&RotatePassword, "rotate-password", false, "Rotates the user's password with an automatically generated password. The length of the password is determine by either --password-length or the value set on the server, in that order.")
	UpdateUserCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	UpdateUserCmd.Flags().BoolVar(&ShowSystemAccounts, "set-system-account-password", false, "Allows for a system account password to be set.")
}

// UpdateCmd represents the update command
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a pgouser, pgorole, or cluster",
	Long: `The update command allows you to update a pgouser, pgorole, or cluster. For example:

	pgo update cluster --selector=name=mycluster --disable-autofail
	pgo update cluster --all --enable-autofail
	pgo update namespace mynamespace
	pgo update pgbouncer mycluster --rotate-password
	pgo update pgorole somerole --pgorole-permission="Cat"
	pgo update pgouser someuser --pgouser-password=somenewpassword
	pgo update pgouser someuser --pgouser-roles="role1,role2"
	pgo update pgouser someuser --pgouser-namespaces="pgouser2"
	pgo update pgorole somerole --pgorole-permission="Cat"
	pgo update user mycluster --username=testuser --selector=name=mycluster --password=somepassword`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println(`Error: You must specify the type of resource to update.  Valid resource types include:
	* cluster
	* namespace
	* pgbouncer
	* pgorole
	* pgouser
	* user`)
		} else {
			switch args[0] {
			case "user", "cluster", "pgbouncer", "pgouser", "pgorole", "namespace":
				break
			default:
				fmt.Println(`Error: You must specify the type of resource to update.  Valid resource types include:
	* cluster
	* namespace
	* pgbouncer
	* pgorole
	* pgouser
	* user`)
			}
		}
	},
}

var PgouserChangePassword bool

// UpdateClusterCmd ...
var UpdateClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Update a PostgreSQL cluster",
	Long: `Update a PostgreSQL cluster. For example:

    pgo update cluster mycluster --disable-autofail
    pgo update cluster mycluster myothercluster --disable-autofail
    pgo update cluster --selector=name=mycluster --disable-autofail
    pgo update cluster --all --enable-autofail`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}

		if len(args) == 0 && Selector == "" && !AllFlag {
			fmt.Println("Error: A cluster name(s) or selector or --all is required for this command.")
			os.Exit(1)
		}

		// if both --enable-autofail and --disable-autofail are true, then abort
		if EnableAutofailFlag && DisableAutofailFlag {
			fmt.Println("Error: Cannot set --enable-autofail and --disable-autofail simultaneously")
			os.Exit(1)
		}

		if EnableStandby {
			fmt.Println("Enabling standby mode will result in the deltion of all PVCs " +
				"for this cluster!\nData will only be retained if the proper retention policy " +
				"is configured for any associated storage classes and/or persistent volumes.\n" +
				"Please proceed with caution.")
		}

		if DisableStandby {
			fmt.Println("Disabling standby mode will enable database writes for this " +
				"cluster.\nPlease ensure the cluster this standby cluster is replicating " +
				"from has been properly shutdown before proceeding!")
		}

		if EnableMetrics || DisableMetrics {
			fmt.Println("Adding or removing a metrics collection sidecar can cause downtime.")
		}

		if EnablePGBadger || DisablePGBadger {
			fmt.Println("Adding or removing a pgBadger sidecar can cause downtime.")
		}

		if len(Tablespaces) > 0 {
			fmt.Println("Adding tablespaces can cause downtime.")
		}

		if CPURequest != "" || CPULimit != "" {
			fmt.Println("Updating CPU resources can cause downtime.")
		}

		if MemoryRequest != "" || MemoryLimit != "" {
			fmt.Println("Updating memory resources can cause downtime.")
		}

		if BackrestCPURequest != "" || BackrestMemoryRequest != "" ||
			BackrestCPULimit != "" || BackrestMemoryLimit != "" {
			fmt.Println("Updating pgBackRest resources can cause temporary unavailability of backups and WAL archives.")
		}

		if ExporterCPURequest != "" || ExporterMemoryRequest != "" ||
			ExporterCPULimit != "" || ExporterMemoryLimit != "" {
			fmt.Println("Updating Crunchy Postgres Exporter resources can cause downtime.")
		}

		if !util.AskForConfirmation(NoPrompt, "") {
			fmt.Println("Aborting...")
			return
		}

		updateCluster(args, Namespace)
	},
}

var UpdateUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Update a PostgreSQL user",
	Long: `Allows the ability to perform various user management functions for PostgreSQL users.

For example:

//change a password, set valid days for 40 days from now
pgo update user mycluster --username=someuser --password=foo
//expire password for a user
pgo update user mycluster --username=someuser --expire-user
//Update all passwords older than the number of days specified
pgo update user mycluster --expired=45 --password-length=8

# Disable the ability for a user to log into the PostgreSQL cluster
pgo update user mycluster --username=foobar --disable-login

# Enable the ability for a user to log into the PostgreSQL cluster
pgo update user mycluster --username=foobar --enable-login
		`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}

		// Check to see that there is an appropriate selector, be it clusters names,
		// a Kubernetes selector, or the --all flag
		if !AllFlag && Selector == "" && len(args) == 0 {
			fmt.Println("Error: You must specify a --selector, --all  or a list of clusters.")
			os.Exit(1)
		}

		// require either the "username" flag or the "expired" flag
		if Username == "" && Expired == 0 {
			fmt.Println("Error: You must specify either --username or --expired")
			os.Exit(1)
		}

		// if both --enable-login and --disable-login are true, then abort
		if EnableLogin && DisableLogin {
			fmt.Println("Error: Cannot set --enable-login and --disable-login simultaneously")
			os.Exit(1)
		}

		updateUser(args, Namespace)
	},
}

var UpdatePgBouncerCmd = &cobra.Command{
	Use:   "pgbouncer",
	Short: "Update a pgBouncer deployment for a PostgreSQL cluster",
	Long: `Used to update the pgBouncer deployment for a PostgreSQL cluster, such
	as by rotating a password. For example:

	pgo update pgbouncer hacluster --rotate-password
	`,

	Run: func(cmd *cobra.Command, args []string) {
		if !util.AskForConfirmation(NoPrompt, pgBouncerPrompt) {
			fmt.Println("Aborting...")
			return
		}

		if Namespace == "" {
			Namespace = PGONamespace
		}

		if PgBouncerReplicas < 0 {
			fmt.Println("Error: You must specify one or more replicas.")
			os.Exit(1)
		}

		updatePgBouncer(Namespace, args)
	},
}

var UpdatePgouserCmd = &cobra.Command{
	Use:   "pgouser",
	Short: "Update a pgouser",
	Long: `UPDATE allows you to update a pgo user. For example:
		pgo update pgouser myuser --pgouser-roles=somerole
		pgo update pgouser myuser --pgouser-password=somepassword --pgouser-roles=somerole
		pgo update pgouser myuser --pgouser-password=somepassword --no-prompt`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}

		if len(args) == 0 {
			fmt.Println("Error: You must specify the name of a pgouser.")
		} else {
			updatePgouser(args, Namespace)
		}
	},
}

var UpdatePgoroleCmd = &cobra.Command{
	Use:   "pgorole",
	Short: "Update a pgorole",
	Long: `UPDATE allows you to update a pgo role. For example:
		pgo update pgorole somerole  --permissions="Cat,Ls`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}

		if len(args) == 0 {
			fmt.Println("Error: You must specify the name of a pgorole.")
		} else {
			updatePgorole(args, Namespace)
		}
	},
}

var UpdateNamespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Update a namespace, applying Operator RBAC",
	Long: `UPDATE allows you to update a Namespace. For example:
		pgo update namespace mynamespace`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Error: You must specify the name of a Namespace.")
		} else {
			updateNamespace(args)
		}
	},
}
