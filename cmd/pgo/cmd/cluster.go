package cmd

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/crunchydata/postgres-operator/cmd/pgo/api"
	"github.com/crunchydata/postgres-operator/cmd/pgo/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

// below are the tablespace parameters and the expected values of each
const (
	// tablespaceParamName represents the name of the PostgreSQL tablespace
	tablespaceParamName = "name"
	// tablespaceParamPVCSize represents the size of the PVC
	tablespaceParamPVCSize = "pvcsize"
	// tablespaceParamStorageConfig represents the storage config to use for the
	// tablespace
	tablespaceParamStorageConfig = "storageconfig"
)

// availableTablespaceParams is the list of acceptable parameters in the
// --tablespace flag
var availableTablespaceParams = map[string]struct{}{
	tablespaceParamName:          {},
	tablespaceParamPVCSize:       {},
	tablespaceParamStorageConfig: {},
}

// requiredTablespaceParams are the tablespace parameters that are required
var requiredTablespaceParams = []string{
	tablespaceParamName,
	tablespaceParamStorageConfig,
}

// deleteCluster will delete a PostgreSQL cluster that is managed by the
// PostgreSQL Operator
func deleteCluster(args []string, ns string) {
	log.Debugf("deleteCluster called %v", args)

	if AllFlag {
		args = make([]string, 1)
		args[0] = "all"
	}

	r := msgs.DeleteClusterRequest{}
	r.Selector = Selector
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns
	r.DeleteBackups = !KeepBackups
	r.DeleteData = !KeepData

	for _, arg := range args {
		r.Clustername = arg
		response, err := api.DeleteCluster(httpclient, &r, &SessionCredentials)

		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code == msgs.Ok {
			for _, result := range response.Results {
				fmt.Println(result)
			}
		} else {
			fmt.Println("Error: " + response.Status.Msg)
		}

	}

}

// showCluster ...
func showCluster(args []string, ns string) {

	log.Debugf("showCluster called %v", args)

	if OutputFormat != "" {
		if OutputFormat != "json" {
			fmt.Println("Error: ", "json is the only supported --output format value")
			os.Exit(2)
		}
	}

	log.Debugf("selector is %s", Selector)
	if len(args) == 0 && !AllFlag && Selector == "" {
		fmt.Println("Error: ", "--all needs to be set or a cluster name be entered or a --selector be specified")
		os.Exit(2)
	}
	if Selector != "" || AllFlag {
		args = make([]string, 1)
		args[0] = ""
	}

	r := new(msgs.ShowClusterRequest)
	r.Selector = Selector
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION

	for _, v := range args {

		r.Clustername = v
		response, err := api.ShowCluster(httpclient, &SessionCredentials, r)
		if err != nil {
			fmt.Println("Error: ", err.Error())
			os.Exit(2)
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Println(string(b))
			return
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

		if len(response.Results) == 0 {
			fmt.Println("No clusters found.")
			return
		}

		for _, clusterDetail := range response.Results {
			printCluster(&clusterDetail)
		}

	}

}

// printCluster
func printCluster(detail *msgs.ShowClusterDetail) {
	fmt.Println("")
	fmt.Println("cluster : " + detail.Cluster.Spec.Name + " (" + detail.Cluster.Spec.CCPImage + ":" + detail.Cluster.Spec.CCPImageTag + ")")

	// indicate if a standby cluster
	if detail.Standby {
		fmt.Printf("%sstandby : %t\n", TreeBranch, detail.Standby)
	}

	for _, pod := range detail.Pods {
		podType := "(" + pod.Type + ")"

		podStr := fmt.Sprintf("%spod : %s (%s) on %s (%s) %s", TreeBranch, pod.Name, string(pod.Phase), pod.NodeName, pod.ReadyStatus, podType)
		fmt.Println(podStr)
		for _, pvc := range pod.PVC {
			fmt.Println(fmt.Sprintf("%spvc: %s (%s)", TreeBranch+TreeBranch, pvc.Name, pvc.Capacity))
		}
	}

	// print out the resources
	resources := detail.Cluster.Spec.Resources

	if len(resources) > 0 {
		resourceStr := fmt.Sprintf("%sresources :", TreeBranch)

		if !resources.Cpu().IsZero() {
			resourceStr += fmt.Sprintf(" CPU: %s", resources.Cpu().String())
		}

		if !resources.Memory().IsZero() {
			resourceStr += fmt.Sprintf(" Memory: %s", resources.Memory().String())
		}

		fmt.Println(resourceStr)
	}

	// print out the limits
	limits := detail.Cluster.Spec.Limits

	if len(limits) > 0 {
		limitsStr := fmt.Sprintf("%slimits :", TreeBranch)

		if !limits.Cpu().IsZero() {
			limitsStr += fmt.Sprintf(" CPU: %s", limits.Cpu().String())
		}

		if !limits.Memory().IsZero() {
			limitsStr += fmt.Sprintf(" Memory: %s", limits.Memory().String())
		}

		fmt.Println(limitsStr)
	}

	for _, d := range detail.Deployments {
		fmt.Println(TreeBranch + "deployment : " + d.Name)
	}
	if len(detail.Deployments) > 0 {
		printPolicies(&detail.Deployments[0])
	}

	for _, service := range detail.Services {
		if service.ExternalIP == "" {
			fmt.Println(TreeBranch + "service : " + service.Name + " - ClusterIP (" + service.ClusterIP + ")" + " - Ports (" +
				strings.Trim(strings.Join(strings.Fields(fmt.Sprint(service.ClusterPorts)), ", "), "[]") + ")")
		} else {
			fmt.Println(TreeBranch + "service : " + service.Name + " - ClusterIP (" + service.ClusterIP + ") ExternalIP (" + service.ExternalIP +
				")" + " - Ports (" + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(service.ClusterPorts)), ", "), "[]") + ")")
		}
	}

	for _, replica := range detail.Replicas {
		fmt.Println(TreeBranch + "pgreplica : " + replica.Name)
	}

	fmt.Printf("%s%s", TreeBranch, "labels : ")
	for k, v := range detail.Cluster.ObjectMeta.Labels {
		fmt.Printf("%s=%s ", k, v)
	}
	fmt.Println("")

}

func printPolicies(d *msgs.ShowClusterDeployment) {
	for _, v := range d.PolicyLabels {
		fmt.Printf("%spolicy: %s\n", TreeBranch, v)
	}
}

// createCluster ....
func createCluster(args []string, ns string, createClusterCmd *cobra.Command) {
	var err error

	if len(args) != 1 {
		fmt.Println("Error: A single Cluster name argument is required.")
		return
	}

	if !util.IsValidForResourceName(args[0]) {
		fmt.Println("Error: Cluster name specified is not valid name - must be lowercase alphanumeric")
		return
	}

	r := new(msgs.CreateClusterRequest)
	r.Name = args[0]
	r.Namespace = ns
	r.ReplicaCount = ClusterReplicaCount
	r.NodeLabel = NodeLabel
	r.PasswordLength = PasswordLength
	r.PasswordSuperuser = PasswordSuperuser
	r.PasswordReplication = PasswordReplication
	r.Password = Password
	r.SecretFrom = SecretFrom
	r.UserLabels = UserLabels
	r.Policies = PoliciesFlag
	r.CCPImageTag = CCPImageTag
	r.CCPImage = CCPImage
	r.CCPImagePrefix = CCPImagePrefix
	r.PGOImagePrefix = PGOImagePrefix
	r.MetricsFlag = MetricsFlag
	r.ExporterCPURequest = ExporterCPURequest
	r.ExporterCPULimit = ExporterCPULimit
	r.ExporterMemoryRequest = ExporterMemoryRequest
	r.ExporterMemoryLimit = ExporterMemoryLimit
	r.BadgerFlag = BadgerFlag
	r.ServiceType = ServiceType
	r.AutofailFlag = !DisableAutofailFlag
	r.PgbouncerFlag = PgbouncerFlag
	r.BackrestStorageConfig = BackrestStorageConfig
	r.BackrestStorageType = BackrestStorageType
	r.CustomConfig = CustomConfig
	r.StorageConfig = StorageConfig
	r.ReplicaStorageConfig = ReplicaStorageConfig
	r.ClientVersion = msgs.PGO_VERSION
	r.PodAntiAffinity = PodAntiAffinity
	r.PodAntiAffinityPgBackRest = PodAntiAffinityPgBackRest
	r.PodAntiAffinityPgBouncer = PodAntiAffinityPgBouncer
	r.BackrestConfig = BackrestConfig
	r.BackrestS3CASecretName = BackrestS3CASecretName
	r.BackrestS3Key = BackrestS3Key
	r.BackrestS3KeySecret = BackrestS3KeySecret
	r.BackrestS3Bucket = BackrestS3Bucket
	r.BackrestS3Region = BackrestS3Region
	r.BackrestS3Endpoint = BackrestS3Endpoint
	r.BackrestS3URIStyle = BackrestS3URIStyle
	r.PVCSize = PVCSize
	r.BackrestPVCSize = BackrestPVCSize
	r.Username = Username
	r.ShowSystemAccounts = ShowSystemAccounts
	r.Database = Database
	r.TLSOnly = TLSOnly
	r.TLSSecret = TLSSecret
	r.ReplicationTLSSecret = ReplicationTLSSecret
	r.CASecret = CASecret
	r.Standby = Standby
	r.BackrestRepoPath = BackrestRepoPath
	// set the container resource requests
	r.CPURequest = CPURequest
	r.CPULimit = CPULimit
	r.MemoryRequest = MemoryRequest
	r.MemoryLimit = MemoryLimit
	r.BackrestCPURequest = BackrestCPURequest
	r.BackrestCPULimit = BackrestCPULimit
	r.BackrestMemoryRequest = BackrestMemoryRequest
	r.BackrestMemoryLimit = BackrestMemoryLimit
	r.PgBouncerCPURequest = PgBouncerCPURequest
	r.PgBouncerCPULimit = PgBouncerCPULimit
	r.PgBouncerMemoryRequest = PgBouncerMemoryRequest
	r.PgBouncerMemoryLimit = PgBouncerMemoryLimit
	r.PgBouncerReplicas = PgBouncerReplicas
	r.PgBouncerTLSSecret = PgBouncerTLSSecret
	// determine if the user wants to create tablespaces as part of this request,
	// and if so, set the values
	r.Tablespaces = getTablespaces(Tablespaces)
	r.WALStorageConfig = WALStorageConfig
	r.WALPVCSize = WALPVCSize
	r.PGDataSource.RestoreFrom = RestoreFrom
	r.PGDataSource.RestoreOpts = BackupOpts
	// set any annotations
	r.Annotations = getClusterAnnotations(Annotations, AnnotationsPostgres, AnnotationsBackrest,
		AnnotationsPgBouncer)

	// only set SyncReplication in the request if actually provided via the CLI
	if createClusterCmd.Flag("sync-replication").Changed {
		r.SyncReplication = &SyncReplication
	}
	// only set BackrestS3VerifyTLS in the request if actually provided via the CLI
	// if set, store provided value accordingly
	r.BackrestS3VerifyTLS = msgs.UpdateBackrestS3VerifyTLSDoNothing

	if createClusterCmd.Flag("pgbackrest-s3-verify-tls").Changed {
		if BackrestS3VerifyTLS {
			r.BackrestS3VerifyTLS = msgs.UpdateBackrestS3VerifyTLSEnable
		} else {
			r.BackrestS3VerifyTLS = msgs.UpdateBackrestS3VerifyTLSDisable
		}
	}

	// if the user provided resources for CPU or Memory, validate them to ensure
	// they are valid Kubernetes values
	if err := util.ValidateQuantity(r.CPURequest, "cpu"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.CPULimit, "cpu-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.MemoryRequest, "memory"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.MemoryLimit, "memory-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.BackrestCPURequest, "pgbackrest-cpu"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.BackrestCPULimit, "pgbackrest-cpu-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.BackrestMemoryRequest, "pgbackrest-memory"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.BackrestMemoryLimit, "pgbackrest-memory-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.ExporterCPURequest, "exporter-cpu"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.ExporterCPULimit, "exporter-cpu-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.ExporterMemoryRequest, "exporter-memory"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.ExporterMemoryLimit, "exporter-memory-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.PgBouncerCPURequest, "pgbouncer-cpu"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.PgBouncerCPULimit, "pgbouncer-cpu-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.PgBouncerMemoryRequest, "pgbouncer-memory"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.PgBouncerMemoryLimit, "pgbouncer-memory-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	response, err := api.CreateCluster(httpclient, &SessionCredentials, r)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(2)
	}

	if response.Status.Code == msgs.Error {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	// print out the legacy cluster information
	fmt.Println("created cluster:", response.Result.Name)
	fmt.Println("workflow id:", response.Result.WorkflowID)
	fmt.Println("database name:", response.Result.Database)
	fmt.Println("users:")

	for _, user := range response.Result.Users {
		fmt.Println("\tusername:", user.Username, "password:", user.Password)
	}
}

// getClusterAnnotations determines if there are any Annotations that were provided
// via the various `--annotation` flags.
func getClusterAnnotations(annotationsGlobal, annotationsPostgres, annotationsBackrest, annotationsPgBouncer []string) crv1.ClusterAnnotations {
	annotations := crv1.ClusterAnnotations{
		Backrest:  map[string]string{},
		Global:    map[string]string{},
		PgBouncer: map[string]string{},
		Postgres:  map[string]string{},
	}

	// go through each annotation type and attempt to populate it in the
	// structure. If the syntax is off anywhere, abort
	setClusterAnnotationGroup(annotations.Global, annotationsGlobal)
	setClusterAnnotationGroup(annotations.Postgres, annotationsPostgres)
	setClusterAnnotationGroup(annotations.Backrest, annotationsBackrest)
	setClusterAnnotationGroup(annotations.PgBouncer, annotationsPgBouncer)

	// return the annotations
	return annotations
}

// getTablespaces determines if there are any Tablespaces that were provided
// via the `--tablespace` CLI flag, and if so, process their values. If
// everything checks out, one or more tablespaces are added to the cluster
// request
func getTablespaces(tablespaceParams []string) []msgs.ClusterTablespaceDetail {
	tablespaces := []msgs.ClusterTablespaceDetail{}

	// if there are no tablespaces set in the Tablespaces slice, abort
	if len(Tablespaces) == 0 {
		return tablespaces
	}

	for _, tablespace := range tablespaceParams {
		tablespaceDetails := map[string]string{}

		// tablespaces are in the format "name=tsname:storageconfig=nfsstorage",
		// so we need to split this out in order to put that information into the
		// tablespace detail struct
		// we will do the initial split of the string, and then iterate to get the
		// key value map of the parameters, ignoring any ones that do not exist
		for _, tablespaceParamValue := range strings.Split(tablespace, ":") {
			tablespaceDetailParts := strings.Split(tablespaceParamValue, "=")

			// if the split is not 2 items, then abort, as that means this is not
			// a valid key/value pair
			if len(tablespaceDetailParts) != 2 {
				fmt.Println(`Error: Tablespace was not specified in proper format (e.g. "name=tablespacename"), aborting.`)
				os.Exit(1)
			}

			// store the param as lower case
			param := strings.ToLower(tablespaceDetailParts[0])

			// if this is not a tablespace parameter, ignore it
			if !isTablespaceParam(param) {
				continue
			}

			// alright, store this param/value in the map
			tablespaceDetails[param] = tablespaceDetailParts[1]
		}

		// determine if the required parameters are in the map. if they are not,
		// abort
		for _, requiredParam := range requiredTablespaceParams {
			_, found := tablespaceDetails[requiredParam]

			if !found {
				fmt.Printf("Error: Required tablespace parameter \"%s\" is not found, aborting\n", requiredParam)
				os.Exit(1)
			}
		}

		// create the cluster tablespace detail and append it to the slice
		clusterTablespaceDetail := msgs.ClusterTablespaceDetail{
			Name:          tablespaceDetails[tablespaceParamName],
			PVCSize:       tablespaceDetails[tablespaceParamPVCSize],
			StorageConfig: tablespaceDetails[tablespaceParamStorageConfig],
		}

		// append to the tablespaces slice, and continue
		tablespaces = append(tablespaces, clusterTablespaceDetail)
	}

	// return the tablespace list
	return tablespaces
}

// isTablespaceParam returns true if the parameter in question is acceptable for
// using with a tablespace.
func isTablespaceParam(param string) bool {
	_, found := availableTablespaceParams[param]

	return found
}

// setClusterAnnotationGroup sets up the annotations for a particular group
func setClusterAnnotationGroup(annotationGroup map[string]string, annotations []string) {
	for _, annotation := range annotations {
		// there are two types of annotations syntaxes:
		//
		// 1: key=value (adding, editing)
		// 2: key- (removing)
		if strings.HasSuffix(annotation, "-") {
			annotationGroup[strings.TrimSuffix(annotation, "-")] = ""
			continue
		}

		parts := strings.Split(annotation, "=")

		if len(parts) != 2 {
			fmt.Println(`Error: Annotation was not specified in propert format (i.e. key=value), aborting.`)
			os.Exit(1)
		}

		annotationGroup[parts[0]] = parts[1]
	}
}

// updateCluster ...
func updateCluster(args []string, ns string) {
	log.Debugf("updateCluster called %v", args)

	r := msgs.UpdateClusterRequest{}
	r.Selector = Selector
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.BackrestCPURequest = BackrestCPURequest
	r.BackrestCPULimit = BackrestCPULimit
	r.BackrestMemoryRequest = BackrestMemoryRequest
	r.BackrestMemoryLimit = BackrestMemoryLimit
	// set the Crunchy Postgres Exporter resource requests
	r.ExporterCPURequest = ExporterCPURequest
	r.ExporterCPULimit = ExporterCPULimit
	r.ExporterMemoryRequest = ExporterMemoryRequest
	r.ExporterMemoryLimit = ExporterMemoryLimit
	r.ExporterRotatePassword = ExporterRotatePassword
	r.Clustername = args
	r.Startup = Startup
	r.Shutdown = Shutdown
	// set the container resource requests
	r.CPURequest = CPURequest
	r.CPULimit = CPULimit
	r.MemoryRequest = MemoryRequest
	r.MemoryLimit = MemoryLimit
	// determine if the user wants to create tablespaces as part of this request,
	// and if so, set the values
	r.Tablespaces = getTablespaces(Tablespaces)
	// set any annotations
	r.Annotations = getClusterAnnotations(Annotations, AnnotationsPostgres, AnnotationsBackrest,
		AnnotationsPgBouncer)

	// check to see if EnableStandby or DisableStandby is set. If so,
	// set a value for Standby
	if EnableStandby {
		r.Standby = msgs.UpdateClusterStandbyEnable
	} else if DisableStandby {
		r.Standby = msgs.UpdateClusterStandbyDisable
	}

	// check to see if EnableAutofailFlag or DisableAutofailFlag is set. If so,
	// set a value for Autofail
	if EnableAutofailFlag {
		r.Autofail = msgs.UpdateClusterAutofailEnable
	} else if DisableAutofailFlag {
		r.Autofail = msgs.UpdateClusterAutofailDisable
	}

	// check to see if the metrics sidecar needs to be enabled or disabled
	if EnableMetrics {
		r.Metrics = msgs.UpdateClusterMetricsEnable
	} else if DisableMetrics {
		r.Metrics = msgs.UpdateClusterMetricsDisable
	}

	// if the user provided resources for CPU or Memory, validate them to ensure
	// they are valid Kubernetes values
	if err := util.ValidateQuantity(r.CPURequest, "cpu"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.CPULimit, "cpu-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.MemoryRequest, "memory"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.MemoryLimit, "memory-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.BackrestCPURequest, "pgbackrest-cpu"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.BackrestCPULimit, "pgbackrest-cpu-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.BackrestMemoryRequest, "pgbackrest-memory"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.BackrestMemoryLimit, "pgbackrest-memory-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.ExporterCPURequest, "exporter-cpu"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.ExporterCPULimit, "exporter-cpu-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.ExporterMemoryRequest, "exporter-memory"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := util.ValidateQuantity(r.ExporterMemoryLimit, "exporter-memory-limit"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	response, err := api.UpdateCluster(httpclient, &r, &SessionCredentials)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for _, result := range response.Results {
			fmt.Println(result)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
	}

}
