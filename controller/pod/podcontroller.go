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
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Controller holds the connections for the controller
type Controller struct {
	PodClient    *rest.RESTClient
	PodClientset *kubernetes.Clientset
	PodConfig    *rest.Config
	Informer     coreinformers.PodInformer
}

// onAdd is called when a pod is added
func (c *Controller) onAdd(obj interface{}) {

	newPod := obj.(*apiv1.Pod)

	newPodLabels := newPod.GetObjectMeta().GetLabels()
	//only process pods with with vendor=crunchydata label
	if newPodLabels[config.LABEL_VENDOR] == "crunchydata" {
		log.Debugf("Pod Controller: onAdd processing the addition of pod %s in namespace %s",
			newPod.Name, newPod.Namespace)
	}

	//handle the case when a pg database pod is added
	if isPostgresPod(newPod) {
		c.labelPostgresPodAndDeployment(newPod)
		return
	}
}

// onUpdate is called when a pod is updated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {

	oldPod := oldObj.(*apiv1.Pod)
	newPod := newObj.(*apiv1.Pod)

	newPodLabels := newPod.GetObjectMeta().GetLabels()

	//only process pods with with vendor=crunchydata label
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

	// Lookup the pgcluster CR for PG cluster associated with this Pod.  Since a 'pg-cluster'
	// label was found on updated Pod, this lookup should always succeed.
	clusterName := newPodLabels[config.LABEL_PG_CLUSTER]
	cluster := crv1.Pgcluster{}
	_, err := kubeapi.Getpgcluster(c.PodClient, &cluster, clusterName,
		newPod.ObjectMeta.Namespace)
	if err != nil {
		log.Error(err.Error())
		return
	}

	// Handle the "role" label change from "replica" to "master" following a failover.  This
	// logic is only triggered when the cluster has already been initialized, which implies
	// a failover or switchove has ocurred.
	if isPromotedPostgresPod(oldPod, newPod) {
		log.Debugf("Pod Controller: pod %s in namespace %s promoted, calling pod promotion "+
			"handler", newPod.Name, newPod.Namespace)
		if err := c.handlePostgresPodPromotion(newPod, cluster); err != nil {
			log.Error(err)
			return
		}
	}

	if cluster.Status.State == crv1.PgclusterStateInitialized &&
		isPromotedStandby(oldPod, newPod) {
		log.Debugf("Pod Controller: standby pod %s in namespace %s promoted, calling standby pod "+
			"promotion handler", newPod.Name, newPod.Namespace)
		if err := c.handleStandbyPromotion(newPod, cluster); err != nil {
			log.Error(err)
			return
		}
	}

	// For the following upgrade and cluster initialization scenarios we only care about updates
	// where the database container within the pod is becoming ready.  We can therefore return
	// at this point if this condition is false.
	if !isDBContainerBecomingReady(oldPod, newPod) {
		return
	}

	// First handle pod update as needed if the update was part of an ongoing upgrade
	if cluster.Labels[config.LABEL_MINOR_UPGRADE] == config.LABEL_UPGRADE_IN_PROGRESS {
		log.Debugf("Pod Controller: upgrade pod %s now ready, calling pod upgrade "+
			"handler", newPod.Name, newPod.Namespace)
		if err := c.handleUpgradePodUpdate(newPod, &cluster); err != nil {
			log.Error(err)
			return
		}
	}

	// Handle postgresql pod updates as needed for cluster initialization
	if cluster.Status.State != crv1.PgclusterStateInitialized && isPostgresPrimaryPod(newPod) {
		log.Debugf("Pod Controller: pg pod %s now ready in an unintialized cluster, calling "+
			"cluster init handler", newPod.Name, newPod.Namespace)
		if err := c.handleClusterInit(newPod, &cluster); err != nil {
			log.Error(err)
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

// isDBContainerBecomingReady checks to see if the Pod update shows that the Pod has
// transitioned from an 'unready' status to a 'ready' status.
func isDBContainerBecomingReady(oldPod, newPod *apiv1.Pod) bool {
	if !isPostgresPod(newPod) {
		return false
	}
	var oldDatabaseStatus bool
	// first see if the old version of the pod was not ready
	for _, v := range oldPod.Status.ContainerStatuses {
		if v.Name == "database" {
			oldDatabaseStatus = v.Ready
			break
		}
	}
	// if the old version of the pod was not ready, now check if the
	// new version is ready
	if !oldDatabaseStatus {
		for _, v := range newPod.Status.ContainerStatuses {
			if v.Name == "database" {
				if v.Ready {
					return true
				}
			}
		}
	}
	return false
}

// isPostgresPrimaryPod determines whether or not the specific Pod provided is the primary database
// Pod within a PG cluster.  This is done by checking to see if the "role" label for the Pod is set
// to either "master", as set by Patroni to identify the current primary, or "promoted", as set by
// Patroni when promoting a replica to be the new primary.
func isPostgresPrimaryPod(newPod *apiv1.Pod) bool {
	if !isPostgresPod(newPod) {
		return false
	}
	if newPod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == "master" ||
		newPod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == "promoted" {
		return true
	}
	return false
}

// isPromotedPostgresPod determines if the Pod update is the result of the promotion of the pod
// from a replica to the primary within a PG cluster.  This is determined by comparing the 'role'
// label from the old Pod to the 'role' label in the New pod, specifically to determine if the
// label has changed from "promoted" to "master" (this is the label change that will be performed
// by Patroni when promoting a pod).
func isPromotedPostgresPod(oldPod, newPod *apiv1.Pod) bool {
	if !isPostgresPod(newPod) {
		return false
	}
	if oldPod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == "promoted" &&
		newPod.ObjectMeta.Labels[config.LABEL_PGHA_ROLE] == "master" {
		return true
	}
	return false
}

// isPromotedPostgresPod determines if the Pod update is the result of the promotion of the pod
// from a replica to the primary within a PG cluster.  This is determined by comparing the 'role'
// label from the old Pod to the 'role' label in the New pod, specifically to determine if the
// label has changed from "promoted" to "master" (this is the label change that will be performed
// by Patroni when promoting a pod).
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
