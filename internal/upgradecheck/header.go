// Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package upgradecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	googleuuid "github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/kubernetes"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	clientHeader = "X-Crunchy-Client-Metadata"
)

var (
	// Using apimachinery's UUID package, so our deployment UUID will be a string
	deploymentID string
)

// Extensible struct for client upgrade data
type clientUpgradeData struct {
	BridgeClustersTotal int    `json:"bridge_clusters_total"`
	BuildSource         string `json:"build_source"`
	DeploymentID        string `json:"deployment_id"`
	FeatureGatesEnabled string `json:"feature_gates_enabled"`
	IsOpenShift         bool   `json:"is_open_shift"`
	KubernetesEnv       string `json:"kubernetes_env"`
	PGOClustersTotal    int    `json:"pgo_clusters_total"`
	PGOInstaller        string `json:"pgo_installer"`
	PGOInstallerOrigin  string `json:"pgo_installer_origin"`
	PGOVersion          string `json:"pgo_version"`
	RegistrationToken   string `json:"registration_token"`
}

// generateHeader aggregates data and returns a struct of that data
// If any errors are encountered, it logs those errors and uses the default values
func generateHeader(ctx context.Context, crClient crclient.Client,
	pgoVersion string, registrationToken string) *clientUpgradeData {

	return &clientUpgradeData{
		BridgeClustersTotal: getBridgeClusters(ctx, crClient),
		BuildSource:         os.Getenv("BUILD_SOURCE"),
		DeploymentID:        ensureDeploymentID(ctx, crClient),
		FeatureGatesEnabled: feature.ShowEnabled(ctx),
		IsOpenShift:         kubernetes.IsOpenShift(ctx),
		KubernetesEnv:       kubernetes.VersionString(ctx),
		PGOClustersTotal:    getManagedClusters(ctx, crClient),
		PGOInstaller:        os.Getenv("PGO_INSTALLER"),
		PGOInstallerOrigin:  os.Getenv("PGO_INSTALLER_ORIGIN"),
		PGOVersion:          pgoVersion,
		RegistrationToken:   registrationToken,
	}
}

// ensureDeploymentID checks if the UUID exists in memory or in a ConfigMap
// If no UUID exists, ensureDeploymentID creates one and saves it in memory/as a ConfigMap
// Any errors encountered will be logged and the ID result will be what is in memory
func ensureDeploymentID(ctx context.Context, crClient crclient.Client) string {
	// If there is no deploymentID in memory, generate one for possible use
	if deploymentID == "" {
		deploymentID = string(uuid.NewUUID())
	}

	cm := manageUpgradeCheckConfigMap(ctx, crClient, deploymentID)

	if cm != nil && cm.Data["deployment_id"] != "" {
		deploymentID = cm.Data["deployment_id"]
	}

	return deploymentID
}

// manageUpgradeCheckConfigMap ensures a ConfigMap exists with a UUID
// If it doesn't exist, this creates it with the in-memory ID
// If it exists and it has a valid UUID, use that to replace the in-memory ID
// If it exists but the field is blank or mangled, we update the ConfigMap with the in-memory ID
func manageUpgradeCheckConfigMap(ctx context.Context, crClient crclient.Client,
	currentID string) *corev1.ConfigMap {

	log := logging.FromContext(ctx)
	upgradeCheckConfigMapMetadata := naming.UpgradeCheckConfigMap()

	cm := &corev1.ConfigMap{
		ObjectMeta: upgradeCheckConfigMapMetadata,
		Data:       map[string]string{"deployment_id": currentID},
	}
	cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	// If no namespace is set, then log this and skip trying to set the UUID in the ConfigMap
	if upgradeCheckConfigMapMetadata.GetNamespace() == "" {
		log.V(1).Info("upgrade check issue: namespace not set")
		return cm
	}

	retrievedCM := &corev1.ConfigMap{}
	err := crClient.Get(ctx, naming.AsObjectKey(upgradeCheckConfigMapMetadata), retrievedCM)

	// If we get any error besides IsNotFound, log it, skip any ConfigMap steps,
	// and use the in-memory deploymentID
	if err != nil && !apierrors.IsNotFound(err) {
		log.V(1).Info("upgrade check issue: error retrieving configmap",
			"response", err.Error())
		return cm
	}

	// If we get a ConfigMap with a "deployment_id", check if that UUID is valid
	if retrievedCM.Data["deployment_id"] != "" {
		_, parseErr := googleuuid.Parse(retrievedCM.Data["deployment_id"])
		// No error -- the ConfigMap has a valid deploymentID, so use that
		if parseErr == nil {
			cm.Data["deployment_id"] = retrievedCM.Data["deployment_id"]
		}
	}

	err = applyConfigMap(ctx, crClient, cm, naming.ControllerPostgresCluster)
	if err != nil {
		log.V(1).Info("upgrade check issue: could not apply configmap",
			"response", err.Error())
	}
	return cm
}

// applyConfigMap is a focused version of the Reconciler.apply method,
// meant only to work with this ConfigMap
// It sends an apply patch to the Kubernetes API, with the fieldManager set to the deployment_id
// and the force parameter set to true.
// - https://docs.k8s.io/reference/using-api/server-side-apply/#managers
// - https://docs.k8s.io/reference/using-api/server-side-apply/#conflicts
func applyConfigMap(ctx context.Context, crClient crclient.Client,
	object crclient.Object, owner string) error {
	// Generate an apply-patch by comparing the object to its zero value.
	zero := &corev1.ConfigMap{}
	data, err := crclient.MergeFrom(zero).Data(object)

	if err == nil {
		apply := crclient.RawPatch(crclient.Apply.Type(), data)
		err = crClient.Patch(ctx, object, apply,
			[]crclient.PatchOption{crclient.ForceOwnership, crclient.FieldOwner(owner)}...)
	}
	return err
}

// getManagedClusters returns a count of postgres clusters managed by this PGO instance
// Any errors encountered will be logged and the count result will be 0
func getManagedClusters(ctx context.Context, crClient crclient.Client) int {
	var count int
	clusters := &v1beta1.PostgresClusterList{}
	err := crClient.List(ctx, clusters)
	if err != nil {
		log := logging.FromContext(ctx)
		log.V(1).Info("upgrade check issue: could not count postgres clusters",
			"response", err.Error())
	} else {
		count = len(clusters.Items)
	}
	return count
}

// getBridgeClusters returns a count of Bridge clusters managed by this PGO instance
// Any errors encountered will be logged and the count result will be 0
func getBridgeClusters(ctx context.Context, crClient crclient.Client) int {
	var count int
	clusters := &v1beta1.CrunchyBridgeClusterList{}
	err := crClient.List(ctx, clusters)
	if err != nil {
		log := logging.FromContext(ctx)
		log.V(1).Info("upgrade check issue: could not count bridge clusters",
			"response", err.Error())
	} else {
		count = len(clusters.Items)
	}
	return count
}

func addHeader(req *http.Request, upgradeInfo *clientUpgradeData) *http.Request {
	marshaled, _ := json.Marshal(upgradeInfo)
	req.Header.Add(clientHeader, string(marshaled))
	return req
}
