package pod

/*
Copyright 2017 - 2023 Crunchy Data Solutions, Inc.
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
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Controller holds the connections for the controller
type Controller struct {
	Client   *kubeapi.Client
	Informer coreinformers.PodInformer
}

// onAdd is called when a pod is added
func (c *Controller) onAdd(obj interface{}) {
	newPod := obj.(*apiv1.Pod)

	newPodLabels := newPod.GetObjectMeta().GetLabels()
	// only process pods with with vendor=crunchydata label
	if newPodLabels[config.LABEL_VENDOR] == "crunchydata" {
		log.Debugf("Pod Controller: onAdd processing the addition of pod %s in namespace %s",
			newPod.Name, newPod.Namespace)
	}

	// handle the case when a pg database pod is added
	if isPostgresPod(newPod) {
		c.labelPostgresPodAndDeployment(newPod)
		return
	}
}

// onUpdate is called when a pod is updated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	ctx := context.TODO()
	oldPod := oldObj.(*apiv1.Pod)
	newPod := newObj.(*apiv1.Pod)

	newPodLabels := newPod.GetObjectMeta().GetLabels()

	// only process pods with with vendor=crunchydata label
	if newPodLabels[config.LABEL_VENDOR] != "crunchydata" {
		return
	}

	log.Debugf("Pod Controller: onUpdate processing update for pod %s in namespace %s",
		newPod.Name, newPod.Namespace)

	// we only care about pods attached to a specific cluster, so if this one isn't (as identified
	// by the presence of the 'pg-cluster' label) then return
	if _, ok := newPodLabels[config.LABEL_PG_CLUSTER]; !ok {
		log.Debugf("Pod Controller: onUpdate ignoring update for pod %s in namespace %s since it "+
			"is not associated with a PG cluster", newPod.Name, newPod.Namespace)
		return
	}

	var clusterName, namespace string
	bootstrapCluster := newPodLabels[config.LABEL_PGHA_BOOTSTRAP]
	// Lookup the pgcluster CR for PG cluster associated with this Pod.  Typically we will use the
	// 'pg-cluster' label, but if a bootstrap pod we use the 'pgha-bootstrap' label.
	if bootstrapCluster != "" {
		clusterName = bootstrapCluster
	} else {
		clusterName = newPodLabels[config.LABEL_PG_CLUSTER]
	}
	// get the proper namespace for the pgcluster, which could be different than the pod if
	// restoring across namespaces
	bootstrapNamespace := newPodLabels[config.LABEL_PGHA_BOOTSTRAP_NAMESPACE]
	if bootstrapNamespace != "" {
		namespace = bootstrapNamespace
	} else {
		namespace = newPod.ObjectMeta.Namespace
	}
	cluster, err := c.Client.CrunchydataV1().Pgclusters(namespace).Get(ctx, clusterName,
		metav1.GetOptions{})
	if err != nil {
		log.Error(err.Error())
		return
	}

	// For the following upgrade and cluster initialization scenarios we only care about updates
	// where the database container within the pod is becoming ready.  We can therefore return
	// at this point if this condition is false.
	if cluster.Status.State != crv1.PgclusterStateInitialized &&
		(isDBContainerBecomingReady(oldPod, newPod) ||
			isBackRestRepoBecomingReady(oldPod, newPod)) {
		if err := c.handleClusterInit(newPod, cluster); err != nil {
			log.Error(err)
			return
		}
		return
	}

	// Handle the "role" label change from "replica" to "primary" following a failover.  This
	// logic is only triggered when the cluster has already been initialized, which implies
	// a failover or switchover has occurred.
	if isPromotedPostgresPod(oldPod, newPod) {
		log.Debugf("Pod Controller: pod %s in namespace %s promoted, calling pod promotion "+
			"handler", newPod.Name, newPod.Namespace)

		// update the pgcluster's current primary information to match the promotion
		setCurrentPrimary(c.Client, newPod, cluster)

		if err := c.handlePostgresPodPromotion(newPod, *cluster); err != nil {
			log.Error(err)
			return
		}
	}

	if isPromotedStandby(oldPod, newPod) {
		log.Debugf("Pod Controller: standby pod %s in namespace %s promoted, calling standby pod "+
			"promotion handler", newPod.Name, newPod.Namespace)

		if err := c.handleStandbyPromotion(newPod, *cluster); err != nil {
			log.Error(err)
			return
		}
	}
}

// setCurrentPrimary checks whether the newly promoted primary value differs from the pgcluster's
// current primary value. If different, patch the CRD's annotation to match the new value
func setCurrentPrimary(clientset pgo.Interface, newPod *apiv1.Pod, cluster *crv1.Pgcluster) {
	// if a failover has occurred and the current primary has changed, update the pgcluster CRD's annotation accordingly
	if cluster.Annotations[config.ANNOTATION_CURRENT_PRIMARY] != newPod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME] {
		err := util.CurrentPrimaryUpdate(clientset, cluster, newPod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME], newPod.Namespace)
		if err != nil {
			log.Errorf("PodController unable to patch pgcluster %s with currentprimary value %s Error: %s", cluster.Spec.ClusterName,
				newPod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME], err)
			return
		}
	}
}

// onDelete is called when a pgcluster is deleted
func (c *Controller) onDelete(obj interface{}) {
	pod := obj.(*apiv1.Pod)

	labels := pod.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("Pod Controller: onDelete skipping pod that is not crunchydata %s", pod.ObjectMeta.SelfLink)
		return
	}
}

// AddPodEventHandler adds the pod event handler to the pod informer
func (c *Controller) AddPodEventHandler() {
	c.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	log.Debugf("Pod Controller: added event handler to informer")
}

// isBackRestRepoBecomingReady checks to see if the Pod update shows that the BackRest
// repo Pod has transitioned from an 'unready' status to a 'ready' status.
func isBackRestRepoBecomingReady(oldPod, newPod *apiv1.Pod) bool {
	if !isBackRestRepoPod(newPod) {
		return false
	}
	return isContainerBecomingReady("database", oldPod, newPod)
}

// isBackRestRepoPod determines whether or not a pod is a pgBackRest repository Pod.  This is
// determined by checking to see if the 'pgo-backrest-repo' label is present on the Pod (also,
// this controller will only process pod with the 'vendor=crunchydata' label, so that label is
// assumed to be present), specifically because this label will only be included on pgBackRest
// repository Pods.
func isBackRestRepoPod(newpod *apiv1.Pod) bool {
	_, backrestRepoLabelExists := newpod.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO]

	return backrestRepoLabelExists
}

// isContainerBecomingReady determines whether or not that container specified is moving
// from an unready status to a ready status.
func isContainerBecomingReady(containerName string, oldPod, newPod *apiv1.Pod) bool {
	var oldContainerStatus bool
	// first see if the old version of the container was not ready
	for _, v := range oldPod.Status.ContainerStatuses {
		if v.Name == containerName {
			oldContainerStatus = v.Ready
			break
		}
	}
	// if the old version of the container was not ready, now check if the
	// new version is ready
	if !oldContainerStatus {
		for _, v := range newPod.Status.ContainerStatuses {
			if v.Name == containerName {
				if v.Ready {
					return true
				}
			}
		}
	}
	return false
}

// isDBContainerBecomingReady checks to see if the Pod update shows that the Pod has
// transitioned from an 'unready' status to a 'ready' status.
func isDBContainerBecomingReady(oldPod, newPod *apiv1.Pod) bool {
	if !isPostgresPod(newPod) {
		return false
	}
	return isContainerBecomingReady("database", oldPod, newPod)
}

// isPostgresPod determines whether or not a pod is a PostreSQL Pod, specifically either the
// primary or a replica pod within a PG cluster.  This is determined by checking to see if the
// 'pgo-pg-database' label is present on the Pod (also, this controller will only process pod with
// the 'vendor=crunchydata' label, so that label is assumed to be present), specifically because
// this label will only be included on primary and replica PostgreSQL database pods (and will be
// present as soon as the deployment and pod is created).
func isPostgresPod(newpod *apiv1.Pod) bool {
	_, pgDatabaseLabelExists := newpod.ObjectMeta.Labels[config.LABEL_PG_DATABASE]

	return pgDatabaseLabelExists
}

// isPromotedPostgresPod determines if the Pod update is the result of the promotion of the pod
// from a replica to the primary within a PG cluster.  This is determined by comparing the 'role'
// label from the old Pod to the 'role' label in the New pod, specifically to determine if the
// label has changed from "promoted" to "primary" (this is the label change that will be performed
// by Patroni when promoting a pod).
func isPromotedPostgresPod(oldPod, newPod *apiv1.Pod) bool {
	if !isPostgresPod(newPod) {
		return false
	}
	if oldPod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == "promoted" &&
		newPod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == config.LABEL_PGHA_ROLE_PRIMARY {
		return true
	}
	return false
}

// isPromotedStandby determines if the Pod update is the result of the promotion of the standby pod
// from a replica to the primary within a PG cluster.  This is determined by comparing the 'role'
// label from the old Pod to the 'role' label in the New pod, specifically to determine if the
// label has changed from "standby_leader" to "primary" (this is the label change that will be
// performed by Patroni when promoting a pod).
func isPromotedStandby(oldPod, newPod *apiv1.Pod) bool {
	if !isPostgresPod(newPod) {
		return false
	}

	oldStatus := oldPod.Annotations["status"]
	newStatus := newPod.Annotations["status"]
	if strings.Contains(oldStatus, "\"role\":\"standby_leader\"") &&
		strings.Contains(newStatus, "\"role\":\"master\"") {
		return true
	}
	return false
}
