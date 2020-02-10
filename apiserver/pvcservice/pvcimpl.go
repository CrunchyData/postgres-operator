package pvcservice

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

	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
)

// ShowPVC ...
func ShowPVC(allflag bool, clusterName, ns string) ([]msgs.ShowPVCResponseResult, error) {
	pvcList := []msgs.ShowPVCResponseResult{}
	// note to a future editor...all of our managed PVCs have a label called
	// called "pgremove"
	selector := fmt.Sprintf("%s=%s", config.LABEL_PGREMOVE, "true")

	// if allflag is not set to true, then update the selector to target the
	// specific PVCs for a specific cluster
	if !allflag {
		selector += fmt.Sprintf(",%s=%s", config.LABEL_PG_CLUSTER, clusterName)
	}

	pvcs, err := kubeapi.GetPVCs(apiserver.Clientset, selector, ns)
	if err != nil {
		return pvcList, err
	}

	log.Debugf("got %d PVCs from ShowPVC query", len(pvcs.Items))
	for _, p := range pvcs.Items {
		pvcResult := msgs.ShowPVCResponseResult{
			ClusterName: p.ObjectMeta.Labels[config.LABEL_PG_CLUSTER],
			PVCName:     p.Name,
		}
		pvcList = append(pvcList, pvcResult)
	}

	return pvcList, nil
}
