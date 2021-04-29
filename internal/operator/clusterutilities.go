package operator

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

// PGHAConfigMapSuffix defines the suffix for the name of the PGHA configMap created for each PG
// cluster
const PGHAConfigMapSuffix = "pgha-config"

// the following constants define the settings in the PGHA configMap that is created for each PG
// cluster
const (
	// PGHAConfigInitSetting determines whether or not initialization logic should be run in the
	// crunchy-postgres-ha (or GIS equivilaent) container
	PGHAConfigInitSetting = "init"
	// PGHAConfigReplicaBootstrapRepoType defines an override for the type of repo (local, S3, etc.)
	// that should be utilized when bootstrapping a replica (i.e. it override the
	// PGBACKREST_REPO1_TYPE env var in the environment).  Allows for dynamic changing of the
	// backrest repo type without requiring container restarts (as would be required to update
	// PGBACKREST_REPO1_TYPE).
	PGHAConfigReplicaBootstrapRepoType = "replica-bootstrap-repo-type"
)

// defaultPGBackRestS3URIStyle is the default pgBackRest S3 URI style to use if a specific style is
// not provided
const defaultPGBackRestS3URIStyle = "host"

// affinityType represents the two affinity types provided by Kubernetes, specifically
// either preferredDuringSchedulingIgnoredDuringExecution or
// requiredDuringSchedulingIgnoredDuringExecution
type affinityType string

const (
	requireScheduleIgnoreExec affinityType = "requiredDuringSchedulingIgnoredDuringExecution"
	preferScheduleIgnoreExec  affinityType = "preferredDuringSchedulingIgnoredDuringExecution"
)

type podAntiAffinityTemplateFields struct {
	AffinityType            affinityType
	ClusterName             string
	PodAntiAffinityLabelKey string
	VendorLabelKey          string
	VendorLabelValue        string
}

// consolidate
type exporterTemplateFields struct {
	Name               string
	JobName            string
	PGOImageTag        string
	PGOImagePrefix     string
	PgPort             string
	ExporterPort       string
	ExporterSecretName string
	ContainerResources string
	TLSOnly            bool
}

// consolidate
type badgerTemplateFields struct {
	CCPImageTag    string
	CCPImagePrefix string
	BadgerTarget   string
	PGBadgerPort   string
}

type PgbackrestEnvVarsTemplateFields struct {
	PgbackrestStanza             string
	PgbackrestDBPath             string
	PgbackrestRepo1Path          string
	PgbackrestRepo1Host          string
	PgbackrestRepo1Type          crv1.BackrestStorageType
	PgbackrestLocalAndGCSStorage bool
	PgbackrestLocalAndS3Storage  bool
	PgbackrestPGPort             string
}

type PgbackrestGCSEnvVarsTemplateFields struct {
	PgbackrestGCSBucket   string
	PgbackrestGCSEndpoint string
	PgbackrestGCSKeyType  string
}

type PgbackrestS3EnvVarsTemplateFields struct {
	PgbackrestS3Bucket     string
	PgbackrestS3Endpoint   string
	PgbackrestS3Region     string
	PgbackrestS3Key        string
	PgbackrestS3KeySecret  string
	PgbackrestS3SecretName string
	PgbackrestS3URIStyle   string
	PgbackrestS3VerifyTLS  string
}

type PgmonitorEnvVarsTemplateFields struct {
	ExporterSecret string
}

// BootstrapJobTemplateFields defines the fields needed to populate the cluster bootstrap job
// template
type BootstrapJobTemplateFields struct {
	DeploymentTemplateFields
	// RestoreFrom defines the name of a cluster to restore from when bootstrapping from an
	// existing data source
	RestoreFrom string
	// RestoreOpts defines the command line options that should be passed to the restore utility
	// (e.g. pgBackRest) when bootstrapping the cluster from an existing data source
	RestoreOpts string
}

// DeploymentTemplateFields ...
type DeploymentTemplateFields struct {
	Name             string
	ClusterName      string
	Port             string
	CCPImagePrefix   string
	CCPImageTag      string
	CCPImage         string
	Database         string
	DeploymentLabels string
	// PodAnnotations are user-specified annotations that can be applied to a
	// Pod, e.g. annotations specific to a PostgreSQL instance
	PodAnnotations            string
	PodLabels                 string
	DataPathOverride          string
	PVCName                   string
	RootSecretName            string
	UserSecretName            string
	PrimarySecretName         string
	SecurityContext           string
	ContainerResources        string
	NodeSelector              string
	ConfVolume                string
	ExporterAddon             string
	BadgerAddon               string
	PgbackrestEnvVars         string
	PgbackrestS3EnvVars       string
	PgbackrestGCSEnvVars      string
	PgmonitorEnvVars          string
	ScopeLabel                string
	Replicas                  string
	IsInit                    bool
	ReplicaReinitOnStartFail  bool
	PasswordType              string
	PodAntiAffinity           string
	PodAntiAffinityLabelName  string
	PodAntiAffinityLabelValue string
	SyncReplication           bool
	Standby                   bool
	// A comma-separated list of tablespace names...this could be an array, but
	// given how this would ultimately be interpreted in a shell script somewhere
	// down the line, it's easier for the time being to do it this way. In the
	// future, we should consider having an array
	Tablespaces            string
	TablespaceVolumes      string
	TablespaceVolumeMounts string
	// Tolerations is an optional parameter that provides Pod tolerations that
	// have been transformed into JSON encoding from an actual Tolerations object
	Tolerations string
	// The following fields set the TLS requirements as well as provide
	// information on how to configure TLS in a PostgreSQL cluster
	// TLSEnabled enables TLS in a cluster if set to true. Only works in actuality
	// if CASecret and TLSSecret are set
	TLSEnabled bool
	// TLSOnly is set to true if the PostgreSQL cluster should only accept TLS
	// connections
	TLSOnly bool
	// TLSSecret is the name of the Secret that has the PostgreSQL server's TLS
	// keypair
	TLSSecret string
	// ReplicationTLSSecret is the name of the Secret that has the TLS keypair
	// for performing certificate-based authentication between instances
	ReplicationTLSSecret string
	// CASecret is the name of the Secret that has the trusted CA that the
	// PostgreSQL server is using
	CASecret string
}

// tablespaceVolumeFields are the fields used to create the volumes in a
// Deployment template spec or the like. These are turned into JSON.
type tablespaceVolumeFields struct {
	Name string                    `json:"name"`
	PVC  tablespaceVolumePVCFields `json:"persistentVolumeClaim"`
}

// tablespaceVolumePVCFields used for specifying the PVC that should be attached
// to the volume. These are turned into JSON
type tablespaceVolumePVCFields struct {
	PVCName string `json:"claimName"`
}

// tablespaceVolumeMountFields are the field used to create the volume mounts
// in a Deployment template spec. These are turned into JSON.
type tablespaceVolumeMountFields struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
}

// GetAnnotations returns the annotations in a JSON format can be used by the
// template. If no annotations are found, returns an empty string
func GetAnnotations(cluster *crv1.Pgcluster, annotationType crv1.ClusterAnnotationType) string {
	annotations := map[string]string{}

	// no matter what, grab any of the global annotations and put into the
	// annotations list
	for k, v := range cluster.Spec.Annotations.Global {
		annotations[k] = v
	}

	// determine if we need to add any additional annotations to the list that may
	// be pod specific
	switch annotationType {
	case crv1.ClusterAnnotationBackrest:
		for k, v := range cluster.Spec.Annotations.Backrest {
			annotations[k] = v
		}
	case crv1.ClusterAnnotationPgBouncer:
		for k, v := range cluster.Spec.Annotations.PgBouncer {
			annotations[k] = v
		}
	case crv1.ClusterAnnotationPostgres:
		for k, v := range cluster.Spec.Annotations.Postgres {
			annotations[k] = v
		}
	case crv1.ClusterAnnotationGlobal: // no-op as its handled in the loop above
	}

	// if the map is empty, return an empty string
	if len(annotations) == 0 {
		return ""
	}

	// let's try to create a JSON document out of the above
	doc, err := json.Marshal(annotations)
	// if there is an error, warn in our logs and return an empty string
	if err != nil {
		log.Errorf("could not set custom annotations: %q", err)
		return ""
	}

	return string(doc)
}

// consolidate with cluster.GetPgbackrestEnvVars
func GetPgbackrestEnvVars(cluster *crv1.Pgcluster, depName, port string) string {
	fields := PgbackrestEnvVarsTemplateFields{
		PgbackrestStanza:             "db",
		PgbackrestRepo1Host:          cluster.Name + "-backrest-shared-repo",
		PgbackrestRepo1Path:          GetPGBackRestRepoPath(cluster),
		PgbackrestDBPath:             "/pgdata/" + depName,
		PgbackrestPGPort:             port,
		PgbackrestRepo1Type:          GetRepoType(cluster),
		PgbackrestLocalAndS3Storage:  IsLocalAndS3Storage(cluster),
		PgbackrestLocalAndGCSStorage: IsLocalAndGCSStorage(cluster),
	}

	doc := bytes.Buffer{}
	if err := config.PgbackrestEnvVarsTemplate.Execute(&doc, fields); err != nil {
		log.Error(err.Error())
		return ""
	}

	return doc.String()
}

// GetPgbackrestBootstrapEnvVars returns a string containing the pgBackRest environment variables
// for a bootstrap job
func GetPgbackrestBootstrapEnvVars(cluster *crv1.Pgcluster,
	restoreFromSecret *v1.Secret) (string, error) {
	ctx := context.TODO()

	depName := cluster.GetAnnotations()[config.ANNOTATION_CURRENT_PRIMARY]
	restoreClusterName := cluster.Spec.PGDataSource.RestoreFrom

	// get the namespace for the repo we're restoring from
	restoreFromNamespace := GetBootstrapNamespace(cluster)

	// Lookup an existing Service to determine its fully qualified domain name.
	// This is inexpensive because the "net" package uses OS-level DNS caching.
	// - https://golang.org/issue/24796
	api := "kubernetes.default.svc"
	cname, err := net.DefaultResolver.LookupCNAME(ctx, api)
	if err != nil {
		return "", err
	}
	domain := strings.TrimPrefix(cname, api+".")

	// use DNS for the host for bootstrap Jobs since we might be bootstrapping across namespaces
	repoHost := fmt.Sprintf(util.BackrestRepoDeploymentName, restoreClusterName) + "." +
		restoreFromNamespace + ".svc." + domain

	fields := PgbackrestEnvVarsTemplateFields{
		PgbackrestStanza:    "db",
		PgbackrestDBPath:    fmt.Sprintf("/pgdata/%s", depName),
		PgbackrestRepo1Path: restoreFromSecret.Annotations[config.ANNOTATION_REPO_PATH],
		PgbackrestPGPort:    restoreFromSecret.Annotations[config.ANNOTATION_PG_PORT],
		PgbackrestRepo1Host: repoHost,
		PgbackrestRepo1Type: crv1.BackrestStorageTypePosix, // just set to the default, can be overridden via CLI args
	}

	var doc bytes.Buffer
	if err := config.PgbackrestEnvVarsTemplate.Execute(&doc, fields); err != nil {
		log.Error(err.Error())
		return "", err
	}
	return doc.String(), nil
}

// GetBackrestDeployment finds the pgBackRest repository Deployments for a
// PostgreQL cluster
func GetBackrestDeployment(clientset kubernetes.Interface, cluster *crv1.Pgcluster) (*apps_v1.Deployment, error) {
	ctx := context.TODO()
	// find the pgBackRest repository Deployment, which follows a known pattern
	deploymentName := fmt.Sprintf(util.BackrestRepoDeploymentName, cluster.Name)
	deployment, err := clientset.AppsV1().Deployments(cluster.Namespace).Get(ctx, deploymentName, metav1.GetOptions{})

	return deployment, err
}

// GetBadgerAddon is a legacy method that generates a JSONish string to be used
// to add a pgBadger sidecar to a PostgreSQL instance
func GetBadgerAddon(cluster *crv1.Pgcluster, target string) string {
	if !cluster.Spec.PGBadger {
		return ""
	}

	log.Debugf("pgBadger enabled for cluster %q", cluster.Name)

	badgerTemplateFields := badgerTemplateFields{
		BadgerTarget:   target,
		CCPImagePrefix: util.GetValueOrDefault(cluster.Spec.CCPImagePrefix, Pgo.Cluster.CCPImagePrefix),
		CCPImageTag:    util.GetStandardImageTag(cluster.Spec.CCPImage, cluster.Spec.CCPImageTag),
		PGBadgerPort:   cluster.Spec.PGBadgerPort,
	}

	if CRUNCHY_DEBUG {
		_ = config.BadgerTemplate.Execute(os.Stdout, badgerTemplateFields)
	}

	doc := bytes.Buffer{}
	if err := config.BadgerTemplate.Execute(&doc, badgerTemplateFields); err != nil {
		log.Error(err)
		return ""
	}

	return doc.String()
}

// GetBootstrapNamespace returns the proper namespace to use when looking up and/or creating any
// resources required to bootstrap a PostgreSQL cluster.  This includes either using the namespace
// specified by the user in the PGDataSource, or defaulting to the same namespace as the pgcluster.
func GetBootstrapNamespace(cluster *crv1.Pgcluster) string {
	var restoreFromNamespace string
	if cluster.Spec.PGDataSource.Namespace != "" {
		restoreFromNamespace = cluster.Spec.PGDataSource.Namespace
	} else {
		restoreFromNamespace = cluster.Namespace
	}
	return restoreFromNamespace
}

// GetExporterAddon returns the template used to create an exporter container
// for metrics. This is semi-legacy, but updated to match the current way of
// handling this
func GetExporterAddon(spec crv1.PgclusterSpec) string {
	// do not execute if metrics are not enabled
	if !spec.Exporter {
		return ""
	}

	exporterTemplateFields := exporterTemplateFields{
		ContainerResources: GetResourcesJSON(spec.ExporterResources, spec.ExporterLimits),
		ExporterPort:       spec.ExporterPort,
		ExporterSecretName: util.GenerateExporterSecretName(spec.ClusterName),
		JobName:            spec.Name,
		Name:               spec.Name,
		PGOImagePrefix:     util.GetValueOrDefault(spec.PGOImagePrefix, Pgo.Pgo.PGOImagePrefix),
		PGOImageTag:        Pgo.Pgo.PGOImageTag,
		PgPort:             spec.Port,
		// see if TLS only is set. however, this also requires checking to see if
		// TLS is enabled in this case. The reason is that even if TLS is only just
		// enabled, because the connection is over an internal interface, we do not
		// need to have the overhead of a TLS connection
		TLSOnly: (spec.TLS.IsTLSEnabled() && spec.TLSOnly),
	}

	if CRUNCHY_DEBUG {
		_ = config.ExporterTemplate.Execute(os.Stdout, exporterTemplateFields)
	}

	exporterDoc := bytes.Buffer{}

	if err := config.ExporterTemplate.Execute(&exporterDoc, exporterTemplateFields); err != nil {
		log.Error(err)
		return ""
	}

	return exporterDoc.String()
}

// consolidate with cluster.GetConfVolume
func GetConfVolume(clientset kubernetes.Interface, cl *crv1.Pgcluster, namespace string) string {
	ctx := context.TODO()
	var configMapStr string

	// check for user provided configmap
	if cl.Spec.CustomConfig != "" {
		_, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, cl.Spec.CustomConfig, metav1.GetOptions{})
		if err != nil {
			// you should NOT get this error because of apiserver validation of this value!
			log.Errorf("%s was not found, error, skipping user provided configMap", cl.Spec.CustomConfig)
		} else {
			log.Debugf("user provided configmap %s was used for this cluster", cl.Spec.CustomConfig)
			return "\"" + cl.Spec.CustomConfig + "\""
		}
	}

	// check for global custom configmap "pgo-custom-pg-config"
	_, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, config.GLOBAL_CUSTOM_CONFIGMAP, metav1.GetOptions{})
	if err == nil {
		return `"pgo-custom-pg-config"`
	}
	log.Debug(config.GLOBAL_CUSTOM_CONFIGMAP + " was not found, skipping global configMap")

	return configMapStr
}

// CreatePGHAConfigMap creates a configMap that will be utilized to store configuration settings
// for a PostgreSQL cluster.  Currently this configMap simply defines an "init" setting, which is
// utilized by the crunchy-postgres-ha container (or GIS equivalent) to determine whether or not
// initialization logic should be executed when the container is run.  This ensures that the
// original primary in a PostgreSQL cluster does not attempt to run any initialization logic more
// than once, such as following a restart of the container.  In the future this configMap can also
// be leveraged to manage other configuration settings for the PostgreSQL cluster and its
// associated containers.
func CreatePGHAConfigMap(clientset kubernetes.Interface, cluster *crv1.Pgcluster,
	namespace string) error {
	ctx := context.TODO()

	labels := make(map[string]string)
	labels[config.LABEL_VENDOR] = config.LABEL_CRUNCHY
	labels[config.LABEL_PG_CLUSTER] = cluster.Name
	labels[config.LABEL_PGHA_CONFIGMAP] = "true"

	data := make(map[string]string)
	// set "init" to true in the postgres-ha configMap
	data[PGHAConfigInitSetting] = "true"

	// if a standby cluster then we want to create replicas using either S3 or GCS
	// pgBackRest repository (and not the local in-cluster pgBackRest repository)
	if cluster.Spec.Standby {
		repoType := crv1.BackrestStorageTypeS3
		for _, rt := range cluster.Spec.BackrestStorageTypes {
			if rt == crv1.BackrestStorageTypeGCS {
				repoType = crv1.BackrestStorageTypeGCS
			}
		}

		data[PGHAConfigReplicaBootstrapRepoType] = string(repoType)
	}

	configmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cluster.Name + "-" + PGHAConfigMapSuffix,
			Labels: labels,
		},
		Data: data,
	}

	for k, v := range util.GetCustomLabels(cluster) {
		configmap.ObjectMeta.Labels[k] = v
	}

	if _, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configmap, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// GetNodeAffinity returns any node affinity rules for the Operator in a JSON
// string. If there is no data or there is an error, it will return an empty
// string.
func GetNodeAffinity(nodeAffinity *v1.NodeAffinity) string {
	if nodeAffinity == nil {
		return ""
	}

	data, err := json.MarshalIndent(nodeAffinity, "", " ")

	if err != nil {
		log.Warnf("could not generate node affinity: %s", err.Error())
		return ""
	}

	return string(data)
}

// GetPasswordType returns the specific password type to use as part of Postgres
// user management. If it's not "scram-sha-256" or "md5", it will just
// return "" and use the default set in Postgres
func GetPasswordType(cluster *crv1.Pgcluster) string {
	switch cluster.Spec.PasswordType {
	case "scram":
		return "scram-sha-256"
	case "scram-sha-256", "md5":
		return cluster.Spec.PasswordType
	}

	return ""
}

// GetTablespaceNamePVCMap returns a map of the tablespace name to the PVC name
func GetTablespaceNamePVCMap(clusterName string, tablespaceStorageTypeMap map[string]string) map[string]string {
	tablespacePVCMap := map[string]string{}

	// iterate through all of the tablespace mounts and match the name of the
	// tablespace to its PVC
	for tablespaceName := range tablespaceStorageTypeMap {
		tablespacePVCMap[tablespaceName] = GetTablespacePVCName(clusterName, tablespaceName)
	}

	return tablespacePVCMap
}

// GetInstanceDeployments finds the Deployments that represent PostgreSQL
// instances
func GetInstanceDeployments(clientset kubernetes.Interface, cluster *crv1.Pgcluster) (*apps_v1.DeploymentList, error) {
	ctx := context.TODO()

	// first, get a list of all of the available deployments so we can properly
	// mount the tablespace PVCs after we create them
	// NOTE: this will also get the pgBackRest deployments, but we will filter
	// these out later
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_VENDOR, config.LABEL_CRUNCHY,
		config.LABEL_PG_CLUSTER, cluster.Name)

	// get the deployments for this specific PostgreSQL luster
	clusterDeployments, err := clientset.
		AppsV1().Deployments(cluster.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	// start prepping the instance deployments
	instanceDeployments := apps_v1.DeploymentList{}

	// iterate through the list of deployments -- if it matches the definition of
	// a PostgreSQL instance deployment, then add it to the slice
	for _, deployment := range clusterDeployments.Items {
		labels := deployment.ObjectMeta.GetLabels()

		// get the name of the PostgreSQL instance. If the "deployment-name"
		// label is not present, then we know it's not a PostgreSQL cluster.
		// Otherwise, the "deployment-name" label doubles as the name of the
		// instance
		if instanceName, ok := labels[config.LABEL_DEPLOYMENT_NAME]; ok {
			log.Debugf("instance found [%s]", instanceName)

			instanceDeployments.Items = append(instanceDeployments.Items, deployment)
		}
	}

	return &instanceDeployments, nil
}

// GetTablespaceNames generates a comma-separated list of the format
// "tablespaceName1,tablespceName2" so that the PVC containing a tablespace
// can be properly mounted in the container, and the tablespace can be
// referenced by the specified human readable name.  We use a comma-separated
// list to make it "easier" to work with the shell scripts that currently setup
// the container
func GetTablespaceNames(tablespaceMounts map[string]crv1.PgStorageSpec) string {
	tablespaces := []string{}

	// iterate through the list of tablespace mounts and extract the tablespace
	// name
	for tablespaceName := range tablespaceMounts {
		tablespaces = append(tablespaces, tablespaceName)
	}

	// return the string that joins the list with the comma
	return strings.Join(tablespaces, ",")
}

// GetTablespaceStorageTypeMap returns a map of "tablespaceName => storageType"
func GetTablespaceStorageTypeMap(tablespaceMounts map[string]crv1.PgStorageSpec) map[string]string {
	tablespaceStorageTypeMap := map[string]string{}

	// iterate through all of the tablespaceMounts and extract the storage type
	for tablespaceName, storageSpec := range tablespaceMounts {
		tablespaceStorageTypeMap[tablespaceName] = storageSpec.StorageType
	}

	return tablespaceStorageTypeMap
}

// GetTablespacePVCName returns the formatted name that is used for a PVC for
// a tablespace
func GetTablespacePVCName(clusterName string, tablespaceName string) string {
	return fmt.Sprintf(config.VOLUME_TABLESPACE_PVC_NAME_FORMAT, clusterName, tablespaceName)
}

// GetTablespaceVolumeMountsJSON Creates an appendable list for the volumeMounts
// that are used to mount table spacs and returns them in a JSON-ish string
func GetTablespaceVolumeMountsJSON(tablespaceStorageTypeMap map[string]string) string {
	volumeMounts := bytes.Buffer{}

	// iterate over each table space and generate the JSON snippet that is loaded
	// into a Kubernetes Deployment template (or equivalent structure)
	for tablespaceName := range tablespaceStorageTypeMap {
		log.Debugf("generating tablespace volume mount json for %s", tablespaceName)

		volumeMountFields := tablespaceVolumeMountFields{
			Name:      GetTablespaceVolumeName(tablespaceName),
			MountPath: fmt.Sprintf("%s%s", config.VOLUME_TABLESPACE_PATH_PREFIX, tablespaceName),
		}

		// write the generated JSON into a buffer. if there is an error, log the
		// error and continue
		if err := writeTablespaceJSON(&volumeMounts, volumeMountFields); err != nil {
			log.Error(err)
			continue
		}
	}

	return volumeMounts.String()
}

// GetTablespaceVolumes Creates an appendable list for the volumes section of a
// Kubernetes pod
func GetTablespaceVolumesJSON(clusterName string, tablespaceStorageTypeMap map[string]string) string {
	volumes := bytes.Buffer{}

	// iterate over each table space and generate the JSON snippet that is loaded
	// into a Kubernetes Deployment template (or equivalent structure)
	for tablespaceName := range tablespaceStorageTypeMap {
		log.Debugf("generating tablespace volume json for %s", tablespaceName)

		volumeFields := tablespaceVolumeFields{
			Name: GetTablespaceVolumeName(tablespaceName),
			PVC: tablespaceVolumePVCFields{
				PVCName: GetTablespacePVCName(clusterName, tablespaceName),
			},
		}

		// write the generated JSON into a buffer. if there is an error, log the
		// error and continue
		if err := writeTablespaceJSON(&volumes, volumeFields); err != nil {
			log.Error(err)
			continue
		}
	}

	return volumes.String()
}

// GetTableSpaceVolumeName returns the name that is used to identify the volume
// that is used to mount the tablespace
func GetTablespaceVolumeName(tablespaceName string) string {
	return fmt.Sprintf("%s%s", config.VOLUME_TABLESPACE_NAME_PREFIX, tablespaceName)
}

// needs to be consolidated with cluster.GetLabelsFromMap
// GetLabelsFromMap ...
func GetLabelsFromMap(labels map[string]string, trimComma bool) string {
	var output string

	for key, value := range labels {
		if len(validation.IsQualifiedName(key)) == 0 && len(validation.IsValidLabelValue(value)) == 0 {
			output += fmt.Sprintf("\"%s\": \"%s\",", key, value)
		}
	}
	// removing the trailing comma from the final label, if request
	if trimComma {
		return strings.TrimSuffix(output, ",")
	}

	return output
}

// GetPodAntiAffinity returns the populated pod anti-affinity json that should be attached to
// the various pods comprising the pg cluster
func GetPodAntiAffinity(cluster *crv1.Pgcluster, deploymentType crv1.PodAntiAffinityDeployment, podAntiAffinityType crv1.PodAntiAffinityType) string {
	log.Debugf("GetPodAnitAffinity with clusterName=[%s]", cluster.Spec.Name)

	// run through the checks on the pod anti-affinity type to see if it is not
	// provided by the user, it's set by one of many defaults
	podAntiAffinityType = GetPodAntiAffinityType(cluster, deploymentType, podAntiAffinityType)

	// verify that the affinity type provided is valid (i.e. 'required' or 'preferred'), and
	// log an error and return an empty string if not
	if err := podAntiAffinityType.Validate(); err != nil {
		log.Error(fmt.Sprintf("Invalid affinity type '%s' specified when attempting to set "+
			"default pod anti-affinity for cluster %s.  Pod anti-affinity will not be applied.",
			podAntiAffinityType, cluster.Spec.Name))
		return ""
	}

	// set requiredDuringSchedulingIgnoredDuringExecution or
	// prefferedDuringSchedulingIgnoredDuringExecution depending on the pod anti-affinity type
	// specified in the pgcluster CR.  Defaults to preffered if not explicitly specified
	// in the CR or in the pgo.yaml configuration file
	templateAffinityType := preferScheduleIgnoreExec
	switch podAntiAffinityType {
	case crv1.PodAntiAffinityDisabled: // if disabled return an empty string
		log.Debugf("Default pod anti-affinity disabled for clusterName=[%s]", cluster.Spec.Name)
		return ""
	case crv1.PodAntiAffinityRequired:
		templateAffinityType = requireScheduleIgnoreExec
	case crv1.PodAntiAffinityPreffered: // no-op as its the default value
	}

	podAntiAffinityTemplateFields := podAntiAffinityTemplateFields{
		AffinityType:            templateAffinityType,
		ClusterName:             cluster.Spec.Name,
		VendorLabelKey:          config.LABEL_VENDOR,
		VendorLabelValue:        config.LABEL_CRUNCHY,
		PodAntiAffinityLabelKey: config.LABEL_POD_ANTI_AFFINITY,
	}

	var podAntiAffinityDoc bytes.Buffer
	err := config.PodAntiAffinityTemplate.Execute(&podAntiAffinityDoc,
		podAntiAffinityTemplateFields)
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	if CRUNCHY_DEBUG {
		_ = config.PodAntiAffinityTemplate.Execute(os.Stdout, podAntiAffinityTemplateFields)
	}

	return podAntiAffinityDoc.String()
}

// GetPodAntiAffinityType returns the type of pod anti-affinity to use. This is
// based on the deployment type (cluster, pgBackRest, pgBouncer), the value
// in the cluster spec, and the defaults available in pgo.yaml.
//
// In other words, the pod anti-affinity is determined by this heuristic, in
// priority order:
//
// 1. If it's pgBackRest/pgBouncer the value set by the user (available in the
//    cluster spec)
// 2. If it's pgBackRest/pgBouncer the value set in pgo.yaml
// 3. The value set in "Default" in the cluster spec
// 4. The value set for PodAntiAffinity in pgo.yaml
func GetPodAntiAffinityType(cluster *crv1.Pgcluster, deploymentType crv1.PodAntiAffinityDeployment, podAntiAffinityType crv1.PodAntiAffinityType) crv1.PodAntiAffinityType {
	// early exit: if podAntiAffinityType is already set, return
	if podAntiAffinityType != "" {
		return podAntiAffinityType
	}

	// if this is a pgBouncer or pgBackRest deployment, see if there is a value
	// set in the configuration. If there is, return that
	switch deploymentType {
	case crv1.PodAntiAffinityDeploymentPgBackRest:
		if Pgo.Cluster.PodAntiAffinityPgBackRest != "" {
			podAntiAffinityType = crv1.PodAntiAffinityType(Pgo.Cluster.PodAntiAffinityPgBackRest)

			if podAntiAffinityType != "" {
				return podAntiAffinityType
			}
		}
	case crv1.PodAntiAffinityDeploymentPgBouncer:
		if Pgo.Cluster.PodAntiAffinityPgBouncer != "" {
			podAntiAffinityType = crv1.PodAntiAffinityType(Pgo.Cluster.PodAntiAffinityPgBouncer)

			if podAntiAffinityType != "" {
				return podAntiAffinityType
			}
		}
	case crv1.PodAntiAffinityDeploymentDefault: // no-op as its the default setting
	}

	// check to see if the value for the cluster anti-affinity is set. If so, use
	// this value
	if cluster.Spec.PodAntiAffinity.Default != "" {
		return cluster.Spec.PodAntiAffinity.Default
	}

	// At this point, check the value in the configuration that is used for pod
	// anti-affinity. Ensure it is cast to be of PodAntiAffinityType
	return crv1.PodAntiAffinityType(Pgo.Cluster.PodAntiAffinity)
}

// GetPgmonitorEnvVars populates the pgmonitor env var template, which contains any
// pgmonitor env vars that need to be included in the Deployment spec for a PG cluster.
func GetPgmonitorEnvVars(cluster *crv1.Pgcluster) string {
	if !cluster.Spec.Exporter {
		return ""
	}

	fields := PgmonitorEnvVarsTemplateFields{
		ExporterSecret: util.GenerateExporterSecretName(cluster.Name),
	}

	doc := bytes.Buffer{}

	if err := config.PgmonitorEnvVarsTemplate.Execute(&doc, fields); err != nil {
		log.Error(err)
		return ""
	}

	return doc.String()
}

// GetPgbackrestGCSEnvVars retrieves the values for the various configuration
// settings required to configure pgBackRest for GCS.
func GetPgbackrestGCSEnvVars(clientset kubernetes.Interface, cluster crv1.Pgcluster) string {
	// determine if backups are enabled to be stored in GCS
	isGCS := false

	for _, storageType := range cluster.Spec.BackrestStorageTypes {
		isGCS = isGCS || (storageType == crv1.BackrestStorageTypeGCS)
	}

	if !isGCS {
		return ""
	}

	// determine the secret for getting the credentials for using GCS as a
	// pgBackRest repository. If we can't do that, then we can't move on
	if _, err := util.GetGCSCredsFromBackrestRepoSecret(clientset, cluster.Namespace, cluster.Name); err != nil {
		return ""
	}

	// populate the GCS bucket, endpoint and key type using either the values in
	// the pgcluster spec (if present), otherwise populate using the values from
	// the pgo.yaml config file
	gcsEnvVars := PgbackrestGCSEnvVarsTemplateFields{
		PgbackrestGCSBucket:   Pgo.Cluster.BackrestGCSBucket,
		PgbackrestGCSEndpoint: Pgo.Cluster.BackrestGCSEndpoint,
		PgbackrestGCSKeyType:  Pgo.Cluster.BackrestGCSKeyType,
	}

	if cluster.Spec.BackrestGCSBucket != "" {
		gcsEnvVars.PgbackrestGCSBucket = cluster.Spec.BackrestGCSBucket
	}

	if cluster.Spec.BackrestGCSEndpoint != "" {
		gcsEnvVars.PgbackrestGCSEndpoint = cluster.Spec.BackrestGCSEndpoint
	}

	if cluster.Spec.BackrestGCSKeyType != "" {
		gcsEnvVars.PgbackrestGCSKeyType = cluster.Spec.BackrestGCSKeyType
	}

	// ensure that bucket is set
	if gcsEnvVars.PgbackrestGCSBucket == "" {
		log.Error("pgBackRest GCS bucket must be set")
		return ""
	}

	// if key type is set, ensure it is of a valid value
	if gcsEnvVars.PgbackrestGCSKeyType != "" &&
		!(gcsEnvVars.PgbackrestGCSKeyType == "service" || gcsEnvVars.PgbackrestGCSKeyType == "token") {
		log.Error(`pgBackRest GCS key type must be either "service" or "token"`)
		return ""
	}

	doc := bytes.Buffer{}

	if err := config.PgbackrestGCSEnvVarsTemplate.Execute(&doc, gcsEnvVars); err != nil {
		log.Error(err.Error())
		return ""
	}

	return doc.String()
}

// GetPgbackrestS3EnvVars retrieves the values for the various configuration settings require to
// configure pgBackRest for AWS S3, including a bucket, endpoint, region, key and key secret.
// The bucket, endpoint & region are obtained from the associated parameters in the pgcluster
// CR, while the key and key secret are obtained from the backrest repository secret.  Once these
// values have been obtained, they are used to populate a template containing the various
// pgBackRest environment variables required to enable S3 support.  After the template has been
// executed with the proper values, the result is then returned a string for inclusion in the PG
// and pgBackRest deployments.
func GetPgbackrestS3EnvVars(clientset kubernetes.Interface, cluster crv1.Pgcluster) string {
	// determine if backups are enabled to be stored on S3
	isS3 := false

	for _, storageType := range cluster.Spec.BackrestStorageTypes {
		isS3 = isS3 || (storageType == crv1.BackrestStorageTypeS3)
	}

	if !isS3 {
		return ""
	}

	// determine the secret for getting the credentials for using S3 as a
	// pgBackRest repository. If we can't do that, then we can't move on
	if _, err := util.GetS3CredsFromBackrestRepoSecret(clientset, cluster.Namespace, cluster.Name); err != nil {
		return ""
	}

	// populate the S3 bucket, endpoint and region using either the values in the pgcluster
	// spec (if present), otherwise populate using the values from the pgo.yaml config file
	s3EnvVars := PgbackrestS3EnvVarsTemplateFields{
		PgbackrestS3Key:        util.BackRestRepoSecretKeyAWSS3KeyAWSS3Key,
		PgbackrestS3KeySecret:  util.BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret,
		PgbackrestS3SecretName: fmt.Sprintf("%s-%s", cluster.Name, config.LABEL_BACKREST_REPO_SECRET),
	}

	if cluster.Spec.BackrestS3Bucket != "" {
		s3EnvVars.PgbackrestS3Bucket = cluster.Spec.BackrestS3Bucket
	} else {
		s3EnvVars.PgbackrestS3Bucket = Pgo.Cluster.BackrestS3Bucket
	}

	if cluster.Spec.BackrestS3Endpoint != "" {
		s3EnvVars.PgbackrestS3Endpoint = cluster.Spec.BackrestS3Endpoint
	} else {
		s3EnvVars.PgbackrestS3Endpoint = Pgo.Cluster.BackrestS3Endpoint
	}

	if cluster.Spec.BackrestS3Region != "" {
		s3EnvVars.PgbackrestS3Region = cluster.Spec.BackrestS3Region
	} else {
		s3EnvVars.PgbackrestS3Region = Pgo.Cluster.BackrestS3Region
	}
	if cluster.Spec.BackrestS3URIStyle != "" {
		s3EnvVars.PgbackrestS3URIStyle = cluster.Spec.BackrestS3URIStyle
	} else {
		s3EnvVars.PgbackrestS3URIStyle = Pgo.Cluster.BackrestS3URIStyle
	}

	// if the URI style is not configured, set to the default value
	if s3EnvVars.PgbackrestS3URIStyle == "" {
		s3EnvVars.PgbackrestS3URIStyle = defaultPGBackRestS3URIStyle
	}
	// if set, pgBackRest URI style must be set to either 'path' or 'host'. If it is neither,
	// log an error and stop the cluster from being created.
	if s3EnvVars.PgbackrestS3URIStyle != "path" && s3EnvVars.PgbackrestS3URIStyle != "host" {
		log.Error("pgBackRest S3 URI style must be set to either \"path\" or \"host\".")
		return ""
	}

	// get the verify TLS boolean value as a string
	s3EnvVars.PgbackrestS3VerifyTLS = GetS3VerifyTLSSetting(&cluster)

	doc := bytes.Buffer{}

	if err := config.PgbackrestS3EnvVarsTemplate.Execute(&doc, s3EnvVars); err != nil {
		log.Error(err.Error())
		return ""
	}

	return doc.String()
}

// GetS3VerifyTLSSetting parses the configured value as a boolean to ensure a valid
// option is used, then returns the pgBackRest S3 configuration value to either enable
// or disable TLS verification as the expected string value.
func GetS3VerifyTLSSetting(cluster *crv1.Pgcluster) string {
	// If the pgcluster has already been set, either by the PGO client or from the
	// CRD definition, parse the boolean value given.
	// If this value is not set, then parse the value stored in the default
	// configuration and set the value accordingly
	verifyTLS, _ := strconv.ParseBool(Pgo.Cluster.BackrestS3VerifyTLS)

	if cluster.Spec.BackrestS3VerifyTLS != "" {
		verifyTLS, _ = strconv.ParseBool(cluster.Spec.BackrestS3VerifyTLS)
	}

	return strconv.FormatBool(verifyTLS)
}

// GetPgbackrestBootstrapGCSEnvVars retrieves the values for the various
// configuration settings required to configure pgBackRest for GCS, specifically
// for a bootstrap job.
func GetPgbackrestBootstrapGCSEnvVars(pgDataSourceRestoreFrom string,
	restoreFromSecret *v1.Secret) string {
	gcsEnvVars := PgbackrestGCSEnvVarsTemplateFields{
		PgbackrestGCSBucket:   restoreFromSecret.Annotations[config.ANNOTATION_GCS_BUCKET],
		PgbackrestGCSEndpoint: restoreFromSecret.Annotations[config.ANNOTATION_GCS_ENDPOINT],
		PgbackrestGCSKeyType:  restoreFromSecret.Annotations[config.ANNOTATION_GCS_KEY_TYPE],
	}

	doc := bytes.Buffer{}

	if err := config.PgbackrestGCSEnvVarsTemplate.Execute(&doc, gcsEnvVars); err != nil {
		log.Error(err)
		return ""
	}

	return doc.String()
}

// GetPgbackrestBootstrapS3EnvVars retrieves the values for the various configuration settings
// required to configure pgBackRest for AWS S3, specifically for a bootstrap job (includes a
// bucket, endpoint, region, key and key secret. The bucket, endpoint & region are obtained from
// annotations in the pgbackrest secret from the cluster being restored from during the bootstrap
// job, while the key and key secret are then obtained from the data in this same secret.  Once
// these values have been obtained, they are used to populate a template containing the various
// pgBackRest environment variables required to enable S3 support for the boostrap job.  After
// the template has been executed with the proper values, the result is then returned a string
// for inclusion in the PG and pgBackRest deployments.
func GetPgbackrestBootstrapS3EnvVars(pgDataSourceRestoreFrom string,
	restoreFromSecret *v1.Secret) string {
	s3EnvVars := PgbackrestS3EnvVarsTemplateFields{
		PgbackrestS3Key:        util.BackRestRepoSecretKeyAWSS3KeyAWSS3Key,
		PgbackrestS3KeySecret:  util.BackRestRepoSecretKeyAWSS3KeyAWSS3KeySecret,
		PgbackrestS3Bucket:     restoreFromSecret.Annotations[config.ANNOTATION_S3_BUCKET],
		PgbackrestS3Endpoint:   restoreFromSecret.Annotations[config.ANNOTATION_S3_ENDPOINT],
		PgbackrestS3Region:     restoreFromSecret.Annotations[config.ANNOTATION_S3_REGION],
		PgbackrestS3SecretName: fmt.Sprintf(util.BackrestRepoSecretName, pgDataSourceRestoreFrom),
	}

	// if the URI style annotation is empty then set the proper default
	if restoreFromSecret.Annotations[config.ANNOTATION_S3_URI_STYLE] != "" {
		s3EnvVars.PgbackrestS3URIStyle = restoreFromSecret.Annotations[config.ANNOTATION_S3_URI_STYLE]
	} else {
		s3EnvVars.PgbackrestS3URIStyle = defaultPGBackRestS3URIStyle
	}

	verifyTLS := restoreFromSecret.Annotations[config.ANNOTATION_S3_VERIFY_TLS]
	if verifyTLS != "" {
		s3EnvVars.PgbackrestS3VerifyTLS = verifyTLS
	} else {
		s3EnvVars.PgbackrestS3VerifyTLS = "true"
	}

	doc := bytes.Buffer{}

	if err := config.PgbackrestS3EnvVarsTemplate.Execute(&doc, s3EnvVars); err != nil {
		log.Error(err.Error())
		return ""
	}

	return doc.String()
}

// UpdatePGHAConfigInitFlag sets the value for the "init" setting in the PGHA configMap for the
// PG cluster to the value specified via the "initVal" parameter.  For instance, following the
// initialization of a PG cluster this function will be utilized to set the "init" value to false
// to ensure the primary does not attempt to run initialization logic in the event that it is
// restarted.
func UpdatePGHAConfigInitFlag(clientset kubernetes.Interface, initVal bool, clusterName,
	namespace string) error {
	ctx := context.TODO()

	log.Debugf("updating init value to %t in the pgha configMap for cluster %s", initVal, clusterName)

	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_PGHA_CONFIGMAP + "=true"
	configMapList, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	switch {
	case err != nil:
		return fmt.Errorf("unable to find the default pgha configMap found for cluster %s using selector %s, unable to set "+
			"init value to false", clusterName, selector)
	case len(configMapList.Items) == 0:
		return fmt.Errorf("no pgha configMaps found for cluster %s using selector %s, unable to set "+
			"init value to false", clusterName, selector)
	case len(configMapList.Items) > 1:
		return fmt.Errorf("more than one default pgha configMap found for cluster %s using selector %s, unable to set "+
			"init value to false", clusterName, selector)
	}

	configMap := &configMapList.Items[0]
	configMap.Data[PGHAConfigInitSetting] = strconv.FormatBool(initVal)

	if _, err := clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

// GetSyncReplication returns true if synchronous replication has been enabled using either the
// pgcluster CR specification or the pgo.yaml configuration file.  Otherwise, if synchronous
// mode has not been enabled, it returns false.
func GetSyncReplication(specSyncReplication *bool) bool {
	// alawys use the value from the CR if explicitly provided
	if specSyncReplication != nil {
		return *specSyncReplication
	}

	return Pgo.Cluster.SyncReplication
}

// OverrideClusterContainerImages is a helper function that provides the
// appropriate hooks to override any of the container images that might be
// deployed with a PostgreSQL cluster
func OverrideClusterContainerImages(containers []v1.Container) {
	// set the container image to an override value, if one exists, which involves
	// looping through the containers array
	for i, container := range containers {
		var containerImageName string
		// there are a few images we need to check for:
		// 1. "database" image, which is PostgreSQL or some flavor of it
		// 2. "exporter" image, which helps with monitoring
		// 3. "pgbadger" image, which helps with...pgbadger
		switch container.Name {

		case "exporter":
			containerImageName = config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_EXPORTER
		case "database":
			containerImageName = config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA
			// one more step here...determine if this is GIS enabled
			// ...yes, this is not ideal
			if strings.Contains(container.Image, "gis-ha") {
				containerImageName = config.CONTAINER_IMAGE_CRUNCHY_POSTGRES_GIS_HA
			}
		case "pgbadger":
			containerImageName = config.CONTAINER_IMAGE_CRUNCHY_PGBADGER
		}

		SetContainerImageOverride(containerImageName, &containers[i])
	}
}

// writeTablespaceJSON is a convenience function to write the tablespace JSON
// into the current buffer
func writeTablespaceJSON(w *bytes.Buffer, jsonFields interface{}) error {
	json, err := json.Marshal(jsonFields)
	// if there is an error, log the error and continue
	if err != nil {
		return err
	}

	// We are appending to the end of a list so we can always assume this comma
	// ...at least for now
	w.WriteString(",")
	w.Write(json)

	return nil
}
