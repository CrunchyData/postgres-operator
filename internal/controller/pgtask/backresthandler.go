package pgtask

/*
Copyright 2020 - 2023 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/config"
	backrestoperator "github.com/crunchydata/postgres-operator/internal/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/internal/operator/cluster"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// handleBackrestRestore handles pgBackRest restores request via a pgtask
func (c *Controller) handleBackrestRestore(task *crv1.Pgtask) {
	ctx := context.TODO()
	namespace := task.GetNamespace()
	clusterName := task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_FROM_CLUSTER]

	cluster, err := c.Client.CrunchydataV1().Pgclusters(namespace).
		Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("pgtask Controller: %s", err.Error())
		return
	}

	cluster, err = backrestoperator.PrepareClusterForRestore(c.Client, cluster, task)
	if err != nil {
		log.Errorf("pgtask Controller: %s", err.Error())
		return
	}
	log.Debugf("pgtask Controller: finished preparing cluster %s for restore", clusterName)

	backrestoperator.UpdatePGClusterSpecForRestore(c.Client, cluster, task)
	log.Debugf("pgtask Controller: finished updating %s spec for restore", clusterName)

	if err := clusteroperator.AddClusterBootstrap(c.Client, cluster); err != nil {
		log.Errorf("pgtask Controller: %s", err.Error())
		return
	}
	log.Debugf("pgtask Controller: added restore job for cluster %s", clusterName)

	backrestoperator.PublishRestore(clusterName, task.ObjectMeta.Labels[config.LABEL_PGOUSER], namespace)

	err = backrestoperator.UpdateWorkflow(c.Client, task.Spec.Parameters[crv1.PgtaskWorkflowID],
		namespace, crv1.PgtaskWorkflowBackrestRestoreJobCreatedStatus)
	if err != nil {
		log.Errorf("pgtask Controller: %s", err.Error())
	}
}
