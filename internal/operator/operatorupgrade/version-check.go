package operatorupgrade

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"

	"github.com/crunchydata/postgres-operator/internal/config"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ErrUnsuccessfulVersionCheck defines the error string that is displayed when a pgcluster
	// version check in a target namespace is unsuccessful
	ErrUnsuccessfulVersionCheck = "unsuccessful pgcluster version check"
)

// CheckVersion looks at the Postgres Operator version information for existing pgclusters and replicas
// if the Operator version listed does not match the current Operator version, create an annotation indicating
// it has not been upgraded
func CheckVersion(clientset pgo.Interface, ns string) error {
	ctx := context.TODO()

	// get all pgclusters
	clusterList, err := clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("%s: %w", ErrUnsuccessfulVersionCheck, err)
	}

	// where the Operator versions do not match, label the pgclusters accordingly
	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]
		if msgs.PGO_VERSION != cluster.Spec.UserLabels[config.LABEL_PGO_VERSION] {
			log.Infof("operator version check - pgcluster %s version is currently %s, current version is %s", cluster.Name, cluster.Spec.UserLabels[config.LABEL_PGO_VERSION], msgs.PGO_VERSION)
			// check if the annotations map has been created
			if cluster.Annotations == nil {
				// if not, create the map
				cluster.Annotations = map[string]string{}
			}
			cluster.Annotations[config.ANNOTATION_IS_UPGRADED] = config.ANNOTATIONS_FALSE
			if _, err := clientset.CrunchydataV1().Pgclusters(ns).Update(ctx, cluster, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("%s: %w", ErrUnsuccessfulVersionCheck, err)
			}
		}
	}

	// update pgreplica CRD userlabels["pgo-version"] to current version
	replicaList, err := clientset.CrunchydataV1().Pgreplicas(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err)
		return fmt.Errorf("%s: %w", ErrUnsuccessfulVersionCheck, err)
	}

	// where the Operator versions do not match, label the replicas accordingly
	for i := range replicaList.Items {
		replica := &replicaList.Items[i]
		if msgs.PGO_VERSION != replica.Spec.UserLabels[config.LABEL_PGO_VERSION] {
			log.Infof("operator version check - pgcluster replica %s version is currently %s, current version is %s", replica.Name, replica.Spec.UserLabels[config.LABEL_PGO_VERSION], msgs.PGO_VERSION)
			// check if the annotations map has been created
			if replica.Annotations == nil {
				// if not, create the map
				replica.Annotations = map[string]string{}
			}
			replica.Annotations[config.ANNOTATION_IS_UPGRADED] = config.ANNOTATIONS_FALSE
			if _, err := clientset.CrunchydataV1().Pgreplicas(ns).Update(ctx, replica, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("%s: %w", ErrUnsuccessfulVersionCheck, err)
			}
		}
	}
	return err
}
