package operatorupgrade

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"time"
)

func OperatorUpgrade(clientset *kubernetes.Clientset, restclient *rest.RESTClient, ns []string) error {
	var err error
	log.Info("OperatorUpgrade starts")
	for i := 0; i < len(ns); i++ {
		err = updateVersion(restclient, ns[i])
		if err != nil {
			log.Error("problem running operator upgrade")
			return err
		}
	}
	log.Info("OperatorUpgrade ends")
	return err
}

func updateVersion(restclient *rest.RESTClient, ns string) error {
	var err error

	t := time.Now()
	dt := t.Format("20060102150405")
	//update pgcluster CRD userlabels["pgo-version"] to current version
	clusterList := crv1.PgclusterList{}

	err = kubeapi.Getpgclusters(restclient, &clusterList, ns)
	if err != nil {
		log.Error(err)
		return err
	}

	for _, cluster := range clusterList.Items {
		if msgs.PGO_VERSION != cluster.Spec.UserLabels[config.LABEL_PGO_VERSION] {
			cluster.Spec.UserLabels[config.LABEL_UPGRADE_DATE] = dt
			log.Infof("operator-upgrade - upgrade pgcluster %s from %s to %s on %s", cluster.Name, cluster.Spec.UserLabels[config.LABEL_PGO_VERSION], msgs.PGO_VERSION, dt)
			cluster.Spec.UserLabels[config.LABEL_PGO_VERSION] = msgs.PGO_VERSION
			err = kubeapi.Updatepgcluster(restclient, &cluster, cluster.Name, ns)
			if err != nil {
				return err
			}
		}
	}

	//update pgreplica CRD userlabels["pgo-version"] to current version
	replicaList := crv1.PgreplicaList{}

	err = kubeapi.Getpgreplicas(restclient, &replicaList, ns)
	if err != nil {
		log.Error(err)
		return err
	}

	for _, replica := range replicaList.Items {
		if msgs.PGO_VERSION != replica.Spec.UserLabels[config.LABEL_PGO_VERSION] {
			replica.Spec.UserLabels[config.LABEL_UPGRADE_DATE] = dt
			log.Infof("operator-upgrade - upgrade pgreplica %s from %s to %s on %s", replica.Name, replica.Spec.UserLabels[config.LABEL_PGO_VERSION], msgs.PGO_VERSION, dt)
			replica.Spec.UserLabels[config.LABEL_PGO_VERSION] = msgs.PGO_VERSION
			err = kubeapi.Updatepgreplica(restclient, &replica, replica.Name, ns)
			if err != nil {
				return err
			}
		}
	}
	return err
}
