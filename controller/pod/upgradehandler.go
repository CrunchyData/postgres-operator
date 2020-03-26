package pod

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
	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	apiv1 "k8s.io/api/core/v1"

	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	log "github.com/sirupsen/logrus"
)

// handleUpgradePodUpdate is responsible for handling updates to pods that occur as part of an
// upgrade
func (c *Controller) handleUpgradePodUpdate(newPod *apiv1.Pod, pgcluster *crv1.Pgcluster) error {

	// have a pod coming back up from upgrade and is ready - time to kick off the next pod.
	if isUpgradedPostgresPod(newPod) {
		upgradeTaskName := pgcluster.Name + "-" + config.LABEL_MINOR_UPGRADE
		clusteroperator.ProcessNextUpgradeItem(c.PodClientset, c.PodClient, *pgcluster, upgradeTaskName,
			newPod.ObjectMeta.Namespace)
	}

	return nil
}

// isUpgradedPostgresPod determines if the pod is one that could be getting a minor upgrade
func isUpgradedPostgresPod(newPod *apiv1.Pod) bool {

	clusterName := newPod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]
	replicaServiceName := clusterName + "-replica"

	// eliminate anything we don't care about - it will be most things
	if newPod.ObjectMeta.Labels[config.LABEL_JOB_NAME] != "" {
		log.Debugf("job pod found [%s]", newPod.Name)
		return false
	}

	if newPod.ObjectMeta.Labels[config.LABEL_NAME] == "postgres-operator" {
		log.Debugf("postgres-operator-pod found [%s]", newPod.Name)
		return false
	}
	if newPod.ObjectMeta.Labels[config.LABEL_PGBOUNCER] == "true" {
		log.Debugf("pgbouncer pod found [%s]", newPod.Name)
		return false
	}

	// look for specific pods that could have just gone through upgrade

	if newPod.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] == "true" {
		log.Debugf("Minor Upgrade: upgraded pgo-backrest-repo found %s", newPod.Name)
		return true
	}

	// primary identified by service-name being same as cluster name
	if newPod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == clusterName {
		log.Debugf("Minor Upgrade: upgraded primary found %s", newPod.Name)
		return true
	}

	if newPod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == replicaServiceName {
		log.Debugf("Minor Upgrade: upgraded replica found %s", newPod.Name)
		return true
	}

	// This indicates there is a pod we didn't account for - shouldn't be the case
	log.Debugf(" **** Minor Upgrade: unexpected isUpgraded pod found: [%s] ****", newPod.Name)

	return false
}
