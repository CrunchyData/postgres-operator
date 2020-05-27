package operatorupgrade

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http:// www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// OperatorCRPgoVersionCheck - checks the version value for any existing PG Cluster. If the values do not match the current
// Operator version, add a label to this cluster. This is called each time the operator starts up.
// (Previously, OperatorUpdateCRPgoVersion updated existing pgcluster custom resources to show current operator version.
func OperatorCRPgoVersionCheck(clientset *kubernetes.Clientset, restclient *rest.RESTClient, ns []string) error {
	var err error
	log.Info("Operator version version check starts")
	for i := 0; i < len(ns); i++ {
		err = checkVersion(restclient, ns[i])
		if err != nil {
			log.Error("problem running operator version check")
			return err
		}
	}
	log.Info("Operator version Update ends")
	return err
}

// checkVersion looks at the Postgres Operator version information for existing pgclusters and replicas
// if the Operator version listed does not match the current Operator version, create an annotation indicating
// it has not been upgraded
func checkVersion(restclient *rest.RESTClient, ns string) error {
	var err error
	clusterList := crv1.PgclusterList{}

	// get all pgclusters
	err = kubeapi.Getpgclusters(restclient, &clusterList, ns)
	if err != nil {
		log.Error(err)
		return err
	}

	// where the Operator versions do not match, label the pgclusters accordingly
	for _, cluster := range clusterList.Items {
		if msgs.PGO_VERSION != cluster.Spec.UserLabels[config.LABEL_PGO_VERSION] {
			log.Infof("operator version check - pgcluster %s version is currently %s, current version is %s", cluster.Name, cluster.Spec.UserLabels[config.LABEL_PGO_VERSION], msgs.PGO_VERSION)
			// check if the annotations map has been created
			if cluster.Annotations == nil {
				// if not, create the map
				cluster.Annotations = map[string]string{}
			}
			cluster.Annotations[config.ANNOTATION_IS_UPGRADED] = config.ANNOTATIONS_FALSE
			err = kubeapi.Updatepgcluster(restclient, &cluster, cluster.Name, ns)
			if err != nil {
				return err
			}
		}
	}

	// update pgreplica CRD userlabels["pgo-version"] to current version
	replicaList := crv1.PgreplicaList{}

	// get all replicas
	err = kubeapi.Getpgreplicas(restclient, &replicaList, ns)
	if err != nil {
		log.Error(err)
		return err
	}

	// where the Operator versions do not match, label the replicas accordingly
	for _, replica := range replicaList.Items {
		if msgs.PGO_VERSION != replica.Spec.UserLabels[config.LABEL_PGO_VERSION] {
			log.Infof("operator version check - pgcluster replica %s version is currently %s, current version is %s", replica.Name, replica.Spec.UserLabels[config.LABEL_PGO_VERSION], msgs.PGO_VERSION)
			// check if the annotations map has been created
			if replica.Annotations == nil {
				// if not, create the map
				replica.Annotations = map[string]string{}
			}
			replica.Annotations[config.ANNOTATION_IS_UPGRADED] = config.ANNOTATIONS_FALSE
			err = kubeapi.Updatepgreplica(restclient, &replica, replica.Name, ns)
			if err != nil {
				return err
			}
		}
	}
	return err
}
