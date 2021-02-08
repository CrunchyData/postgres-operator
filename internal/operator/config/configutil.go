package config

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
)

const (
	// pghaConfigMapName represents the name of the PGHA configMap created for each cluster, which
	// has the name "<clustername>-pgha-config"
	// pghaConfigMapName = "%s-pgha-config"
	// pghaDCSConfigName represents the name of the DCS configuration stored in the
	// "<clustername>-pgha-config" configMap, which is "<clustername>-dcs-config"
	// PGHADCSConfigName = "%s-dcs-config"
	// pghaLocalConfigName represents the name of the local configuration stored for each database
	// server in the "<clustername>-pgha-config" configMap, which is "<clustername>-local-config"
	// pghaLocalConfigName = "%s-local-config"
	//
	pghLocalConfigSuffix = "-local-config"
)

// ErrMissingClusterConfig is the error thrown when configuration is missing from a configMap
var ErrMissingClusterConfig error = errors.New("Configuration is missing from configMap")

// Syncer defines a resource that is able to sync its configuration stored configuration with a
// service, application, etc.
type Syncer interface {
	Sync() error
}

// PatchConfigMapData replaces the configuration stored the configuration specified with the
// provided content
func PatchConfigMapData(kubeclientset kubernetes.Interface, configMap *corev1.ConfigMap,
	configName string, content []byte) error {
	ctx := context.TODO()

	jsonOpBytes, err := kubeapi.NewJSONPatch().Replace("data", configName)(string(content)).Bytes()
	if err != nil {
		return err
	}

	log.Debugf("patching configmap %s: %s", configMap.GetName(), jsonOpBytes)
	_, err = kubeclientset.CoreV1().ConfigMaps(configMap.GetNamespace()).
		Patch(ctx, configMap.GetName(), types.JSONPatchType, jsonOpBytes, metav1.PatchOptions{})

	return err
}
