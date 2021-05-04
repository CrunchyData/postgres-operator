package cmd

/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/cmd/pgo/api"
	"github.com/crunchydata/postgres-operator/cmd/pgo/util"
	pgoutil "github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
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

		for i := range response.Results {
			printCluster(&response.Results[i])
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
			fmt.Printf("%spvc: %s (%s)\n", TreeBranch+TreeBranch, pvc.Name, pvc.Capacity)
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

	labels := pgoutil.GetCustomLabels(&detail.Cluster)
	for k, v := range detail.Cluster.ObjectMeta.Labels {
		labels[k] = v
	}

	fmt.Printf("%s%s", TreeBranch, "labels : ")
	for k, v := range labels {
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
	r.NodeAffinityType = getNodeAffinityType(NodeLabel, NodeAffinityType)
	r.NodeLabel = NodeLabel
	r.PasswordLength = PasswordLength
	r.PasswordSuperuser = PasswordSuperuser
	r.PasswordReplication = PasswordReplication
	r.Password = Password
	r.PasswordType = PasswordType
	r.SecretFrom = SecretFrom
	r.UserLabels = getLabels(UserLabels)
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
	r.ServiceType = v1.ServiceType(ServiceType)
	r.PgBouncerServiceType = v1.ServiceType(ServiceTypePgBouncer)
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
	r.BackrestGCSBucket = BackrestGCSBucket
	r.BackrestGCSEndpoint = BackrestGCSEndpoint
	r.BackrestGCSKeyType = BackrestGCSKeyType
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
	r.PGDataSource.Namespace = RestoreFromNamespace
	r.PGDataSource.RestoreOpts = BackupOpts
	// set any annotations
	r.Annotations = getClusterAnnotations(Annotations, AnnotationsPostgres, AnnotationsBackrest,
		AnnotationsPgBouncer)
	// set any tolerations
	r.Tolerations = getClusterTolerations(Tolerations, false)

	// only set SyncReplication in the request if actually provided via the CLI
	if createClusterCmd.Flag("sync-replication").Changed {
		r.SyncReplication = &SyncReplication

		// if it is true, ensure there is at least one replica
		if r.SyncReplication != nil && *r.SyncReplication && r.ReplicaCount < 1 {
			r.ReplicaCount = 1
		}
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

	// r.BackrestGCSKey = BackrestGCSKey
	// if a GCS key is provided, it is a path to the file, so we need to see if
	// the file exists. and if it does, load it in
	if BackrestGCSKey != "" {
		gcsKeyFile, err := filepath.Abs(BackrestGCSKey)

		if err != nil {
			fmt.Println("invalid filename for --pgbackrest-gcs-key: ", err.Error())
			os.Exit(1)
		}

		gcsKey, err := ioutil.ReadFile(gcsKeyFile)

		if err != nil {
			fmt.Println("could not read GCS Key from file: ", err.Error())
			os.Exit(1)
		}

		// now we have a value that can be sent to the API server
		r.BackrestGCSKey = base64.StdEncoding.EncodeToString(gcsKey)
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

// getClusterTolerations determines if there are any Pod tolerations to set
// and converts from the defined string form to the standard Toleration object
//
// The strings should follow the following formats:
//
// Operator - rule:Effect
//
// Exists:
//   - key
//   - key:Effect
//
// Equals:
//   - key=value
//   - key=value:Effect
//
// If the remove flag is set to true, check for a trailing "-" at the end of
// each item, as this will be a remove list. Otherwise, only consider
// tolerations that are not being removed
func getClusterTolerations(tolerationList []string, remove bool) []v1.Toleration {
	tolerations := make([]v1.Toleration, 0)

	// if no tolerations, exit early
	if len(tolerationList) == 0 {
		return tolerations
	}

	// begin the joys of parsing
	for _, t := range tolerationList {
		toleration := v1.Toleration{}
		ruleEffect := strings.Split(t, ":")

		// if we don't have exactly two items, then error
		if len(ruleEffect) < 1 || len(ruleEffect) > 2 {
			fmt.Printf("invalid format for toleration: %q\n", t)
			os.Exit(1)
		}

		// for ease of reading
		rule, effectStr := ruleEffect[0], ""
		// effect string is only set if ruleEffect is of length 2
		if len(ruleEffect) == 2 {
			effectStr = ruleEffect[1]
		}

		// determine if the effect is for removal or not, as we will continue the
		// loop based on that.
		//
		// In other words, skip processing the value if either:
		// - This *is* removal mode AND the value *does not* have the removal suffix "-"
		// - This *is not* removal mode AND the value *does* have the removal suffix "-"
		if (remove && !strings.HasSuffix(effectStr, "-") && !strings.HasSuffix(rule, "-")) ||
			(!remove && (strings.HasSuffix(effectStr, "-") || strings.HasSuffix(rule, "-"))) {
			continue
		}

		// no matter what we can trim any trailing "-" off of the string, and cast
		// it as a TaintEffect
		rule = strings.TrimSuffix(rule, "-")
		effect := v1.TaintEffect(strings.TrimSuffix(effectStr, "-"))

		// see if the effect is a valid effect
		if !isValidTaintEffect(effect) {
			fmt.Printf("invalid taint effect for toleration: %q\n", effect)
			os.Exit(1)
		}

		toleration.Effect = effect

		// determine if the rule is an Exists or Equals operation
		keyValue := strings.Split(rule, "=")

		if len(keyValue) < 1 || len(keyValue) > 2 {
			fmt.Printf("invalid rule for toleration: %q\n", rule)
			os.Exit(1)
		}

		// no matter what we have a key
		toleration.Key = keyValue[0]

		// the following determine the operation to use for the toleration and if
		// we should assign a value
		if len(keyValue) == 1 {
			toleration.Operator = v1.TolerationOpExists
		} else {
			toleration.Operator = v1.TolerationOpEqual
			toleration.Value = keyValue[1]
		}

		// and append to the list of tolerations
		tolerations = append(tolerations, toleration)
	}

	return tolerations
}

// isValidTaintEffect returns true if the effect passed in is a valid
// TaintEffect, otherwise false
func isValidTaintEffect(taintEffect v1.TaintEffect) bool {
	return (taintEffect == v1.TaintEffectNoSchedule ||
		taintEffect == v1.TaintEffectPreferNoSchedule ||
		taintEffect == v1.TaintEffectNoExecute ||
		taintEffect == "")
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
	r.Clustername = args
	r.Selector = Selector
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.BackrestCPURequest = BackrestCPURequest
	r.BackrestCPULimit = BackrestCPULimit
	r.BackrestMemoryRequest = BackrestMemoryRequest
	r.BackrestMemoryLimit = BackrestMemoryLimit
	r.BackrestPVCSize = BackrestPVCSize
	// set the Crunchy Postgres Exporter resource requests
	r.ExporterCPURequest = ExporterCPURequest
	r.ExporterCPULimit = ExporterCPULimit
	r.ExporterMemoryRequest = ExporterMemoryRequest
	r.ExporterMemoryLimit = ExporterMemoryLimit
	r.ExporterRotatePassword = ExporterRotatePassword
	r.PVCSize = PVCSize
	r.ServiceType = v1.ServiceType(ServiceType)
	r.Startup = Startup
	r.Shutdown = Shutdown
	r.WALPVCSize = WALPVCSize
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
	r.Tolerations = getClusterTolerations(Tolerations, false)
	r.TolerationsDelete = getClusterTolerations(Tolerations, true)

	// most of the TLS settings
	if DisableTLS {
		r.DisableTLS = DisableTLS
	} else {
		r.TLSSecret = TLSSecret
		r.ReplicationTLSSecret = ReplicationTLSSecret
		r.CASecret = CASecret

		// check to see if we need to enable/disable TLS only
		if EnableTLSOnly {
			r.TLSOnly = msgs.UpdateClusterTLSOnlyEnable
		} else if DisableTLSOnly {
			r.TLSOnly = msgs.UpdateClusterTLSOnlyDisable
		}
	}

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

	// check to see if the pgBadger sidecar needs to be enabled or disabled
	if EnablePGBadger {
		r.PGBadger = msgs.UpdateClusterPGBadgerEnable
	} else if DisablePGBadger {
		r.PGBadger = msgs.UpdateClusterPGBadgerDisable
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
