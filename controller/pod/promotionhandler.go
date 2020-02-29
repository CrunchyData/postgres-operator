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
	"fmt"

	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator/backrest"
	apiv1 "k8s.io/api/core/v1"
)

// handlePostgresPodPromotion is responsible for handling updates to PG pods the occur as a result
// of a failover.  Specifically, this handler is triggered when a replica has been promoted, and
// it now has either the "promoted" or "master" role label.
func (c *Controller) handlePostgresPodPromotion(newPod *apiv1.Pod, clusterName string) error {

	//look up the backrest-repo pod name
	selector := fmt.Sprintf("%s=%s,%s=true", config.LABEL_PG_CLUSTER,
		clusterName, config.LABEL_PGO_BACKREST_REPO)
	pods, err := kubeapi.GetPods(c.PodClientset, selector, newPod.ObjectMeta.Namespace)
	if len(pods.Items) != 1 {
		return fmt.Errorf("pods len != 1 for cluster %s", clusterName)
	} else if err != nil {
		return err
	}

	err = backrest.CleanBackupResources(c.PodClient, c.PodClientset,
		newPod.ObjectMeta.Namespace, clusterName)
	if err != nil {
		return err
	}
	_, err = backrest.CreatePostFailoverBackup(c.PodClient,
		newPod.ObjectMeta.Namespace, clusterName, pods.Items[0].Name)
	if err != nil {
		return err
	}

	return nil
}
