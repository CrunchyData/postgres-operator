package operator

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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/ns"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	// defaultBackrestRepoPath defines the default repo1-path for pgBackRest for
	// use when a specic path is not provided in the pgcluster CR.  The '%s'
	// format verb will be replaced with the cluster name when this variable is
	// utilized
	defaultBackrestRepoPath = "/backrestrepo/%s-backrest-shared-repo"
	// defaultBackrestRepoConfigPath contains the default configuration that are used
	// to set up a pgBackRest repository
	defaultBackrestRepoConfigPath = "/default-pgo-backrest-repo/"
	// defaultRegistry is the default registry to pull the container images from
	defaultRegistry = "registry.developers.crunchydata.com/crunchydata"
)

var (
	CRUNCHY_DEBUG bool
	NAMESPACE     string
)

var (
	InstallationName string
	PgoNamespace     string
	EventTCPAddress  = "localhost:4150"
)

var Pgo config.PgoConfig

// ContainerImageOverrides contains a list of container images that are
// overridden by the RELATED_IMAGE_* environmental variables that can be set by
// people deploying the Operator
var ContainerImageOverrides = map[string]string{}

// NamespaceOperatingMode defines the namespace operating mode for the cluster,
// e.g. "dynamic", "readonly" or "disabled".  See type NamespaceOperatingMode
// for detailed explanations of each mode available.
var namespaceOperatingMode ns.NamespaceOperatingMode

// runAsNonRoot forces the Pod to run as a non-root Pod
var runAsNonRoot = true

type containerResourcesTemplateFields struct {
	// LimitsMemory and LimitsCPU detemrine the memory/CPU limits
	LimitsMemory, LimitsCPU string
	// RequestsMemory and RequestsCPU determine how much memory/CPU resources to
	// request
	RequestsMemory, RequestsCPU string
}

// defaultBackrestRepoConfigKeys are the default keys expected to be in the
// pgBackRest repo config secret
var defaultBackrestRepoConfigKeys = []string{"config", "sshd_config", "aws-s3-ca.crt"}

func Initialize(clientset kubernetes.Interface) {
	tmp := os.Getenv("CRUNCHY_DEBUG")
	if tmp == "true" {
		CRUNCHY_DEBUG = true
		log.Debug("CRUNCHY_DEBUG flag set to true")
	} else {
		CRUNCHY_DEBUG = false
		log.Info("CRUNCHY_DEBUG flag set to false")
	}

	NAMESPACE = os.Getenv("NAMESPACE")
	log.Infof("NAMESPACE %s", NAMESPACE)

	InstallationName = os.Getenv("PGO_INSTALLATION_NAME")
	log.Infof("InstallationName %s", InstallationName)
	if InstallationName == "" {
		log.Error("PGO_INSTALLATION_NAME env var is required")
		os.Exit(2)
	}

	PgoNamespace = os.Getenv("PGO_OPERATOR_NAMESPACE")
	if PgoNamespace == "" {
		log.Error("PGO_OPERATOR_NAMESPACE environment variable is not set and is required, this is the namespace that the Operator is to run within.")
		os.Exit(2)
	}

	if err := Pgo.GetConfig(clientset, PgoNamespace); err != nil {
		log.Error(err)
		log.Fatal("pgo-config files and templates did not load")
	}

	// initialize the general pgBackRest secret
	if err := initializeOperatorBackrestSecret(clientset, PgoNamespace); err != nil {
		log.Fatal(err)
	}

	if Pgo.Cluster.CCPImagePrefix == "" {
		log.Debugf("pgo.yaml CCPImagePrefix not set, using default %q", defaultRegistry)
		Pgo.Cluster.CCPImagePrefix = defaultRegistry
	} else {
		log.Debugf("pgo.yaml CCPImagePrefix set, using %s", Pgo.Cluster.CCPImagePrefix)
	}
	if Pgo.Pgo.PGOImagePrefix == "" {
		log.Debugf("pgo.yaml PGOImagePrefix not set, using default %q", defaultRegistry)
		Pgo.Pgo.PGOImagePrefix = defaultRegistry
	} else {
		log.Debugf("PGOImagePrefix set, using %s", Pgo.Pgo.PGOImagePrefix)
	}

	// In a RELATED_IMAGE_* world, this does not _need_ to be set, but our
	// installer does set it up so we could be ok...
	if Pgo.Pgo.PGOImageTag == "" {
		log.Error("pgo.yaml PGOImageTag not set, required ")
		os.Exit(2)
	}

	// initialize any container image overrides that are set by the "RELATED_*"
	// variables
	initializeContainerImageOverrides()

	tmp = os.Getenv("EVENT_TCP_ADDRESS")
	if tmp != "" {
		EventTCPAddress = tmp
	}
	log.Info("EventTCPAddress set to " + EventTCPAddress)

	// set controller refresh intervals and worker counts
	initializeControllerRefreshIntervals()
	initializeControllerWorkerCounts()
}

// GetPodSecurityContext will generate the security context required for a
// Deployment by incorporating the standard fsGroup for the user that runs the
// container (typically the "postgres" user), and adds any supplemental groups
// that may need to be added, e.g. for NFS storage.
//
// Following the legacy method, this returns a JSON string, which will be
// modified in the future. Mainly this is transitioning from the legacy function
// by adding the expected types
func GetPodSecurityContext(supplementalGroups []int64) string {
	// set up the security context struct
	securityContext := v1.PodSecurityContext{
		// we don't want to run the pods as root, so explicitly disallow this
		RunAsNonRoot: &runAsNonRoot,
		// add any supplemental groups that the user passed in
		SupplementalGroups: supplementalGroups,
	}

	// determine if we should use the PostgreSQL FSGroup.
	if !Pgo.DisableFSGroup() {
		// we store the PostgreSQL FSGroup in this constant as an int64, so it's
		// just carried over
		securityContext.FSGroup = &crv1.PGFSGroup
	}

	// ...convert to JSON. Errors are ignored
	doc, err := json.Marshal(securityContext)
	// if there happens to be an error, warn about it
	if err != nil {
		log.Warn(err)
	}

	// for debug purposes, we can look at the document
	log.Debug(doc)

	// return a string of the security context
	return string(doc)
}

// GetResourcesJSON is a pseudo-legacy method that creates JSON that applies the
// CPU and Memory settings. The settings are only included if:
// a) they exist
// b) they are nonzero
func GetResourcesJSON(resources, limits v1.ResourceList) string {
	fields := containerResourcesTemplateFields{}

	// first, if the contents of the resources list happen to be nil, exit out
	if resources == nil && limits == nil {
		return ""
	}

	if resources != nil {
		if resources.Cpu() != nil && !resources.Cpu().IsZero() {
			fields.RequestsCPU = resources.Cpu().String()
		}

		if resources.Memory() != nil && !resources.Memory().IsZero() {
			fields.RequestsMemory = resources.Memory().String()
		}
	}

	if limits != nil {
		if limits.Cpu() != nil && !limits.Cpu().IsZero() {
			fields.LimitsCPU = limits.Cpu().String()
		}

		if limits.Memory() != nil && !limits.Memory().IsZero() {
			fields.LimitsMemory = limits.Memory().String()
		}
	}

	doc := bytes.Buffer{}

	if err := config.ContainerResourcesTemplate.Execute(&doc, fields); err != nil {
		log.Error(err)
		return ""
	}

	if log.GetLevel() == log.DebugLevel {
		_ = config.ContainerResourcesTemplate.Execute(os.Stdout, fields)
	}

	return doc.String()
}

// GetPGBackRestRepoPath is responsible for determining the repo path setting
// (i.e. 'repo1-path' flag) for use by pgBackRest.  If a specific repo path has
// been defined in the pgcluster CR, then that path will be returned. Otherwise
// a default path will be returned that is generated from the cluster name
func GetPGBackRestRepoPath(cluster *crv1.Pgcluster) string {
	if cluster.Spec.BackrestRepoPath != "" {
		return cluster.Spec.BackrestRepoPath
	}
	return fmt.Sprintf(defaultBackrestRepoPath, cluster.Name)
}

// GetRepoType returns the proper repo type to set in container based on the
// backrest storage type provided
//
// If there are multiple types, the default returned is "posix". This could
// change once there is proper multi-repo support, but with proper multi-repo
// support, this function is likely annhilated.
//
// If there is nothing, the default returned is posix
func GetRepoType(cluster *crv1.Pgcluster) crv1.BackrestStorageType {
	// so...per the above comment...
	if len(cluster.Spec.BackrestStorageTypes) == 0 || len(cluster.Spec.BackrestStorageTypes) > 1 {
		return crv1.BackrestStorageTypePosix
	}

	// alright, so there is only 1. If it happens to be "local" ensure that posix
	// is returned
	if cluster.Spec.BackrestStorageTypes[0] == crv1.BackrestStorageTypeLocal {
		return crv1.BackrestStorageTypePosix
	}

	return cluster.Spec.BackrestStorageTypes[0]
}

// IsLocalAndGCSStorage a boolean indicating whether or not local and gcs
// storage should be enabled for pgBackRest based on the backrestStorageType
// string provided
func IsLocalAndGCSStorage(cluster *crv1.Pgcluster) bool {
	// this works for the time being. if the counter is two or greater, then we
	// have both local and GCS storage
	i := 0

	for _, storageType := range cluster.Spec.BackrestStorageTypes {
		switch storageType {
		default: // no-op
		case crv1.BackrestStorageTypeLocal, crv1.BackrestStorageTypePosix, crv1.BackrestStorageTypeGCS:
			i += 1
		}
	}

	return i >= 2
}

// IsLocalAndS3Storage a boolean indicating whether or not local and s3 storage should
// be enabled for pgBackRest based on the backrestStorageType string provided
func IsLocalAndS3Storage(cluster *crv1.Pgcluster) bool {
	// this works for the time being. if the counter is two or greater, then we
	// have both local and S3 storage
	i := 0

	for _, storageType := range cluster.Spec.BackrestStorageTypes {
		switch storageType {
		default: // no -oop
		case crv1.BackrestStorageTypeLocal, crv1.BackrestStorageTypePosix, crv1.BackrestStorageTypeS3:
			i += 1
		}
	}

	return i >= 2
}

// ScaleDeployment scales a deployment to a specified number of replicas. It
// will also wait to ensure that the Deployment is actually scaled down.
func ScaleDeployment(clientset kubeapi.Interface,
	deployment *appsv1.Deployment, replicas *int32) error {
	ctx := context.TODO()

	patch, _ := kubeapi.NewMergePatch().Add("spec", "replicas")(*replicas).Bytes()

	log.Debugf("patching deployment %s: %s", deployment.GetName(), patch)

	// Patch the Deployment with the updated number of replicas, which will
	// trigger the scaling operation. We store the updated deployment so the
	// object can be later updated when we scale back up
	_, err := clientset.AppsV1().Deployments(deployment.Namespace).
		Patch(ctx, deployment.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})

	return err
}

// SetContainerImageOverride determines if there is an override available for
// a container image, and sets said value on the Kubernetes Container image
// definition
func SetContainerImageOverride(containerImageName string, container *v1.Container) {
	// if a container image name override is available, set it!
	overrideImageName := ContainerImageOverrides[containerImageName]

	if overrideImageName != "" {
		log.Debugf("overriding image %s with %s", containerImageName, overrideImageName)

		container.Image = overrideImageName
	}
}

// getCandidatePod tries to get the candidate Pod for a switchover or failover.
// If "candidateName" is provided, it will seek out the specific PostgreSQL
// instance. Otherwise, it will just attempt to find a running Pod.
//
// If such a Pod cannot be found, we likely cannot use the instance for a
// switchover for failover candidate as it is not running.
func getCandidatePod(clientset kubernetes.Interface, cluster *crv1.Pgcluster, candidateName string) (*v1.Pod, error) {
	ctx := context.TODO()

	// build the label selector. we are looking for any PostgreSQL instance within
	// this cluster, so that part is easy
	labelSelector := fields.Set{
		config.LABEL_PG_CLUSTER:  cluster.Name,
		config.LABEL_PG_DATABASE: config.LABEL_TRUE,
	}

	// if a candidateName is supplied, use that as part of the label selector to
	// find the candidate Pod
	if candidateName != "" {
		labelSelector[config.LABEL_DEPLOYMENT_NAME] = candidateName
	}

	// ensure the Pod is part of the cluster and is running
	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: labelSelector.String(),
	}

	pods, err := clientset.CoreV1().Pods(cluster.Namespace).List(ctx, options)
	if err != nil {
		return nil, err
	}

	// if no Pods are found, then also return an error as we then cannot switch
	// over to this instance
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods found for instance %s", candidateName)
	}

	// the list returns multiple Pods, so just return the first one
	return &pods.Items[0], nil
}

// initializeContainerImageOverrides initializes the container image overrides
// that could be set if there are any `RELATED_IMAGE_*` environmental variables
func initializeContainerImageOverrides() {
	// the easiest way to handle this is to iterate over the RelatedImageMap,
	// check if said image exist in the environmental variable, and if it does
	// load it in as an override. Otherwise, ignore.
	for relatedImageEnvVar, imageName := range config.RelatedImageMap {
		// see if the envirionmental variable overrides the image name or not
		overrideImageName := os.Getenv(relatedImageEnvVar)

		// if it is overridden, set the image name the map
		if overrideImageName != "" {
			ContainerImageOverrides[imageName] = overrideImageName
			log.Infof("image %s overridden by: %s", imageName, overrideImageName)
		}
	}
}

// initControllerRefreshIntervals initializes the refresh intervals for any informers
// created by the Operator requiring a refresh interval.  This includes first attempting
// to utilize the refresh interval(s) defined in the pgo.yaml config file, and if not
// present then falling back to a default value.
func initializeControllerRefreshIntervals() {
	// set the namespace controller refresh interval if not provided in the pgo.yaml
	if Pgo.Pgo.NamespaceRefreshInterval == nil {
		log.Debugf("NamespaceRefreshInterval not set, defaulting to %d seconds",
			config.DefaultNamespaceRefreshInterval)
		defaultVal := int(config.DefaultNamespaceRefreshInterval)
		Pgo.Pgo.NamespaceRefreshInterval = &defaultVal
	} else {
		log.Debugf("NamespaceRefreshInterval is set, using %d seconds",
			*Pgo.Pgo.NamespaceRefreshInterval)
	}

	// set the default controller group refresh interval if not provided in the pgo.yaml
	if Pgo.Pgo.ControllerGroupRefreshInterval == nil {
		log.Debugf("ControllerGroupRefreshInterval not set, defaulting to %d seconds",
			config.DefaultControllerGroupRefreshInterval)
		defaultVal := int(config.DefaultControllerGroupRefreshInterval)
		Pgo.Pgo.ControllerGroupRefreshInterval = &defaultVal
	} else {
		log.Debugf("ControllerGroupRefreshInterval is set, using %d seconds",
			*Pgo.Pgo.ControllerGroupRefreshInterval)
	}
}

// initControllerWorkerCounts sets the number of workers that will be created for any worker
// queues created within the various controllers created by the Operator.  This includes first
// attempting to utilize the worker counts defined in the pgo.yaml config file, and if not
// present then falling back to a default value.
func initializeControllerWorkerCounts() {
	if Pgo.Pgo.ConfigMapWorkerCount == nil {
		log.Debugf("ConfigMapWorkerCount not set, defaulting to %d worker(s)",
			config.DefaultConfigMapWorkerCount)
		defaultVal := int(config.DefaultConfigMapWorkerCount)
		Pgo.Pgo.ConfigMapWorkerCount = &defaultVal
	} else {
		log.Debugf("ConfigMapWorkerCount is set, using %d worker(s)",
			*Pgo.Pgo.ConfigMapWorkerCount)
	}

	if Pgo.Pgo.NamespaceWorkerCount == nil {
		log.Debugf("NamespaceWorkerCount not set, defaulting to %d worker(s)",
			config.DefaultNamespaceWorkerCount)
		defaultVal := int(config.DefaultNamespaceWorkerCount)
		Pgo.Pgo.NamespaceWorkerCount = &defaultVal
	} else {
		log.Debugf("NamespaceWorkerCount is set, using %d worker(s)",
			*Pgo.Pgo.NamespaceWorkerCount)
	}

	if Pgo.Pgo.PGClusterWorkerCount == nil {
		log.Debugf("PGClusterWorkerCount not set, defaulting to %d worker(s)",
			config.DefaultPGClusterWorkerCount)
		defaultVal := int(config.DefaultPGClusterWorkerCount)
		Pgo.Pgo.PGClusterWorkerCount = &defaultVal
	} else {
		log.Debugf("PGClusterWorkerCount is set, using %d worker(s)",
			*Pgo.Pgo.PGClusterWorkerCount)
	}

	if Pgo.Pgo.PGReplicaWorkerCount == nil {
		log.Debugf("PGReplicaWorkerCount not set, defaulting to %d worker(s)",
			config.DefaultPGReplicaWorkerCount)
		defaultVal := int(config.DefaultPGReplicaWorkerCount)
		Pgo.Pgo.PGReplicaWorkerCount = &defaultVal
	} else {
		log.Debugf("PGReplicaWorkerCount is set, using %d worker(s)",
			*Pgo.Pgo.PGReplicaWorkerCount)
	}

	if Pgo.Pgo.PGTaskWorkerCount == nil {
		log.Debugf("PGTaskWorkerCount not set, defaulting to %d worker(s)",
			config.DefaultPGTaskWorkerCount)
		defaultVal := int(config.DefaultPGTaskWorkerCount)
		Pgo.Pgo.PGTaskWorkerCount = &defaultVal
	} else {
		log.Debugf("PGTaskWorkerCount is set, using %d worker(s)",
			*Pgo.Pgo.PGTaskWorkerCount)
	}
}

// initializeOperatorBackrestSecret ensures the generic pgBackRest configuration
// is available
func initializeOperatorBackrestSecret(clientset kubernetes.Interface, namespace string) error {
	var isNew, isModified bool

	ctx := context.TODO()

	// determine if the Secret already exists
	secret, err := clientset.
		CoreV1().Secrets(namespace).
		Get(ctx, config.SecretOperatorBackrestRepoConfig, metav1.GetOptions{})
		// if there is a true error, return. Otherwise, initialize a new Secret
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}

		secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: config.SecretOperatorBackrestRepoConfig,
				Labels: map[string]string{
					config.LABEL_VENDOR: config.LABEL_CRUNCHY,
				},
			},
			Data: map[string][]byte{},
		}
		isNew = true
	}

	// set any missing defaults
	for _, filename := range defaultBackrestRepoConfigKeys {
		// skip if there is already content
		if len(secret.Data[filename]) != 0 {
			continue
		}

		file := path.Join(defaultBackrestRepoConfigPath, filename)

		// if we can't read the contents of the file for whatever reason, warn,
		// but continue
		// otherwise, update the entry in the Secret
		if contents, err := ioutil.ReadFile(file); err != nil {
			log.Warn(err)
			continue
		} else {
			secret.Data[filename] = contents
		}

		isModified = true
	}

	// do not make any updates if the secret is not modified at all
	if !isModified {
		return nil
	}

	// make the API calls based on if we are creating or updating
	if isNew {
		_, err := clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		return err
	}

	_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})

	return err
}

// SetupNamespaces is responsible for the initial namespace configuration for the Operator
// install.  This includes setting the proper namespace operating mode, creating and/or updating
// namespaces as needed (or as permitted by the current operator mode), and returning a valid list
// of namespaces for the current Operator install.
func SetupNamespaces(clientset kubernetes.Interface) ([]string, error) {
	// First set the proper namespace operating mode for the Operator install.  The mode identified
	// determines whether or not certain namespace capabilities are enabled.
	if err := setNamespaceOperatingMode(clientset); err != nil {
		log.Errorf("Error detecting namespace operating mode: %v", err)
		return nil, err
	}
	log.Debugf("Namespace operating mode is '%s'", NamespaceOperatingMode())

	namespaceList, err := ns.GetInitialNamespaceList(clientset, NamespaceOperatingMode(),
		InstallationName, PgoNamespace)
	if err != nil {
		return nil, err
	}

	// proceed with creating and/or updating any namespaces provided for the installation
	if err := ns.ConfigureInstallNamespaces(clientset, InstallationName,
		PgoNamespace, namespaceList, NamespaceOperatingMode()); err != nil {
		log.Errorf("Unable to setup namespaces: %v", err)
		return nil, err
	}

	return namespaceList, nil
}

// setNamespaceOperatingMode set the namespace operating mode for the Operator by calling the
// proper utility function to determine which mode is applicable based on the current
// permissions assigned to the Operator Service Account.
func setNamespaceOperatingMode(clientset kubernetes.Interface) error {
	nsOpMode, err := ns.GetNamespaceOperatingMode(clientset)
	if err != nil {
		return err
	}
	namespaceOperatingMode = nsOpMode

	return nil
}

// NamespaceOperatingMode returns the namespace operating mode for the current Operator
// installation, which is stored in the "namespaceOperatingMode" variable
func NamespaceOperatingMode() ns.NamespaceOperatingMode {
	return namespaceOperatingMode
}
