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
	"encoding/json"
	"os"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/ns"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// defaultRegistry is the default registry to pull the container images from
	defaultRegistry = "registry.developers.crunchydata.com/crunchydata"
)

var CRUNCHY_DEBUG bool
var NAMESPACE string

var InstallationName string
var PgoNamespace string
var EventTCPAddress = "localhost:4150"

var Pgo config.PgoConfig

// ContainerImageOverrides contains a list of container images that are
// overridden by the RELATED_IMAGE_* environmental variables that can be set by
// people deploying the Operator
var ContainerImageOverrides = map[string]string{}

// NamespaceOperatingMode defines the namespace operating mode for the cluster,
// e.g. "dynamic", "readonly" or "disabled".  See type NamespaceOperatingMode
// for detailed explanations of each mode available.
var namespaceOperatingMode ns.NamespaceOperatingMode

type containerResourcesTemplateFields struct {
	// LimitsMemory and LimitsCPU detemrine the memory/CPU limits
	LimitsMemory, LimitsCPU string
	// RequestsMemory and RequestsCPU determine how much memory/CPU resources to
	// request
	RequestsMemory, RequestsCPU string
}

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

	var err error

	err = Pgo.GetConfig(clientset, PgoNamespace)
	if err != nil {
		log.Error(err)
		log.Error("pgo-config files and templates did not load")
		os.Exit(2)
	}

	log.Printf("PrimaryStorage=%v\n", Pgo.Storage["storage1"])

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

	if Pgo.Cluster.PgmonitorPassword == "" {
		log.Debug("pgo.yaml PgmonitorPassword not set, using default")
		Pgo.Cluster.PgmonitorPassword = "password"
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
		// add any supplemental groups that the user passed in
		SupplementalGroups: supplementalGroups,
	}

	// determine if we should use the PostgreSQL FSGroup.
	if !Pgo.Cluster.DisableFSGroup {
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
		config.ContainerResourcesTemplate.Execute(os.Stdout, fields)
	}

	return doc.String()
}

// GetRepoType returns the proper repo type to set in container based on the
// backrest storage type provided
func GetRepoType(backrestStorageType string) string {
	if backrestStorageType != "" && backrestStorageType == "s3" {
		return "s3"
	} else {
		return "posix"
	}
}

// IsLocalAndS3Storage a boolean indicating whether or not local and s3 storage should
// be enabled for pgBackRest based on the backrestStorageType string provided
func IsLocalAndS3Storage(backrestStorageType string) bool {
	if backrestStorageType != "" && strings.Contains(backrestStorageType, "s3") &&
		strings.Contains(backrestStorageType, "local") {
		return true
	}
	return false
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
