package configmap

/*
Copyright 2020 Crunchy Data Solutions, Inc.
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
	"sync"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	cfg "github.com/crunchydata/postgres-operator/operator/config"
)

// handleConfigMapSync is responsible for syncing a configMap resource that has obtained from
// the ConfigMap controller's worker queue
func (c *Controller) handleConfigMapSync(key string) error {

	log.Debugf("ConfigMap Controller: handling a configmap sync for key %s", key)

	namespace, configMapName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	configMap, err := c.cmLister.ConfigMaps(namespace).Get(configMapName)
	if err != nil {
		return err
	}
	clusterName := configMap.GetObjectMeta().GetLabels()[config.LABEL_PG_CLUSTER]

	cluster, err := c.pgclusterLister.Pgclusters(namespace).Get(clusterName)
	if err != nil && kerrors.IsNotFound(err) {
		log.Debugf("ConfigMap Controller: cannot find pgcluster for configMap %s (namespace %s),"+
			"ignoring", configMapName, namespace)
	} else if err != nil {
		return err
	}

	// disable syncing when the cluster isn't currently initialized
	if cluster.Status.State != crv1.PgclusterStateInitialized {
		return nil
	}

	c.syncPGHAConfig(c.createPGHAConfigs(configMap, clusterName))

	return nil
}

// createConfigurerMap creates the configs needed to sync the PGHA configMap
func (c *Controller) createPGHAConfigs(configMap *corev1.ConfigMap,
	clusterName string) []cfg.Syncer {

	var configSyncers []cfg.Syncer

	configSyncers = append(configSyncers, cfg.NewDCS(configMap, c.kubeclientset))

	localDBConfig, err := cfg.NewLocalDB(configMap, c.cmRESTConfig, c.kubeclientset,
		c.cmRESTClient)
	// Just log the error and don't add to the map so a sync can still be attempted with
	// any other configurers
	if err != nil {
		log.Error(err)
	} else {
		configSyncers = append(configSyncers, localDBConfig)
	}

	return configSyncers
}

// syncAllConfigs takes a map of configurers and runs their sync functions concurrently
func (c *Controller) syncPGHAConfig(configSyncers []cfg.Syncer) {

	var wg sync.WaitGroup

	for _, configSyncer := range configSyncers {

		wg.Add(1)

		go func(syncer cfg.Syncer) {
			if err := syncer.Sync(); err != nil {
				log.Error(err)
			}
			wg.Done()
		}(configSyncer)
	}

	wg.Wait()
}
