package autobind

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

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GetPgAdminQueryRunner takes cluster information, identifies whether
// it has a pgAdmin deployment and provides a query runner for executing
// queries against the pgAdmin database
//
// The pointer will be nil if there is no pgAdmin deployed for the cluster
func GetPgAdminQueryRunner(clientset *kubernetes.Clientset, restconfig *rest.Config, cluster *crv1.Pgcluster) (*queryRunner, error) {
	if active, ok := cluster.Labels[config.LABEL_PGADMIN]; !ok || active != "true" {
		return nil, nil
	}

	selector := fmt.Sprintf("%s=true,%s=%s", config.LABEL_PGADMIN, config.LABEL_PG_CLUSTER, cluster.Name)

	pods, err := kubeapi.GetPods(clientset, selector, cluster.Namespace)
	if err != nil {
		log.Errorf("failed to find pgadmin pod [%v]", err)
		return nil, err
	}

	// pgAdmin deployment is single-replica, not HA, should only be one pod
	if l := len(pods.Items); l > 1 {
		log.Warnf("Unexpected number of pods for pgadmin [%d], defaulting to first", l)
	} else if l == 0 {
		err := fmt.Errorf("Unable to find pgadmin pod for cluster %s, deleting instance", cluster.Name)
		return nil, err
	}

	return NewQueryRunner(clientset, restconfig, pods.Items[0]), nil
}

// ServerEntryFromPgService populates the ServerEntry struct based on
// details of the kubernetes service, it is up to the caller to provide
// the assumed PgCluster service
func ServerEntryFromPgService(service *v1.Service, clustername string) ServerEntry {
	dbService := ServerEntry{
		Name:          clustername,
		Host:          service.Spec.ClusterIP,
		Port:          5432,
		SSLMode:       "prefer",
		MaintenanceDB: clustername,
	}

	// Set Port info
	for _, portInfo := range service.Spec.Ports {
		if portInfo.Name == "postgres" {
			dbService.Port = int(portInfo.Port)
		}
	}
	return dbService
}
