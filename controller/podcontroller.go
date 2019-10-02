package controller

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
	"context"
	"sync"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"

	backrestoperator "github.com/crunchydata/postgres-operator/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	taskoperator "github.com/crunchydata/postgres-operator/operator/task"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// PodController holds the connections for the controller
type PodController struct {
	PodClient          *rest.RESTClient
	PodClientset       *kubernetes.Clientset
	Ctx                context.Context
	informerNsMutex    sync.Mutex
	InformerNamespaces map[string]struct{}
}

// Run starts an pod resource controller
func (c *PodController) Run() error {

	err := c.watchPods(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for pod resource: %v", err)
		return err
	}

	<-c.Ctx.Done()
	return c.Ctx.Err()
}

// watchPods is the event loop for pod resources
func (c *PodController) watchPods(ctx context.Context) error {
	nsList := ns.GetNamespaces(c.PodClientset, operator.InstallationName)

	for i := 0; i < len(nsList); i++ {
		log.Infof("starting pod controller on ns [%s]", nsList[i])
		c.SetupWatch(nsList[i])
	}
	return nil
}

// onAdd is called when a pgcluster is added or
// if a pgo-backrest-repo pod is added
func (c *PodController) onAdd(obj interface{}) {
	newpod := obj.(*apiv1.Pod)

	labels := newpod.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("PodController: onAdd skipping pod that is not crunchydata %s", newpod.ObjectMeta.SelfLink)
		return
	}

	log.Debugf("[PodController] OnAdd ns=%s %s", newpod.ObjectMeta.Namespace, newpod.ObjectMeta.SelfLink)

	//handle the case when a pg database pod is added
	if isPostgresPod(newpod) {
		c.checkPostgresPods(newpod, newpod.ObjectMeta.Namespace)
		return
	}
}

// onUpdate is called when a pgcluster is updated
func (c *PodController) onUpdate(oldObj, newObj interface{}) {
	oldpod := oldObj.(*apiv1.Pod)
	newpod := newObj.(*apiv1.Pod)

	labels := newpod.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("PodController: onUpdate skipping pod that is not crunchydata %s", newpod.ObjectMeta.SelfLink)
		return
	}

	log.Debugf("[PodController] onUpdate ns=%s %s", newpod.ObjectMeta.Namespace, newpod.ObjectMeta.SelfLink)

	//look up the pgcluster CRD for this pod's cluster
	clusterName := newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]
	pgcluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(c.PodClient, &pgcluster, clusterName, newpod.ObjectMeta.Namespace)
	if !found || err != nil {
		log.Error(err.Error())
		log.Error("you should not get a not found in the onUpdate in PodController")
		return
	}

	// check here if cluster has an upgrade in progress flag set. 
	clusterInMinorUpgrade := pgcluster.Labels[config.LABEL_MINOR_UPGRADE] == config.LABEL_UPGRADE_IN_PROGRESS 
	// log.Debugf("Cluster: %s Minor Upgrade: %s ", clusterName, clusterInMinorUpgrade)

	// have a pod coming back up from upgrade and is ready - time to kick off the next pod.
	if clusterInMinorUpgrade && isUpgradedPostgresPod(newpod, oldpod) {
		upgradeTaskName := clusterName + "-" + config.LABEL_MINOR_UPGRADE
		clusteroperator.ProcessNextUpgradeItem(c.PodClientset, c.PodClient, clusterName, upgradeTaskName, newpod.ObjectMeta.Namespace)
	}

	//handle the case when a pg database pod is updated
	if isPostgresPod(newpod) {
		//only check the status of primary pods
		if newpod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == clusterName {
			c.checkReadyStatus(oldpod, newpod, &pgcluster)
		}
		return
	}
}

// onDelete is called when a pgcluster is deleted
func (c *PodController) onDelete(obj interface{}) {
	pod := obj.(*apiv1.Pod)

	labels := pod.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("PodController: onDelete skipping pod that is not crunchydata %s", pod.ObjectMeta.SelfLink)
		return
	}

	//	log.Debugf("[PodController] onDelete ns=%s %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.SelfLink)
}

func (c *PodController) checkReadyStatus(oldpod, newpod *apiv1.Pod, cluster *crv1.Pgcluster) {
	//handle the case of a service-name re-label
	if newpod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] !=
		oldpod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] {
		log.Debug("the pod was updated and the service names were changed in this pod update, not going to check the ReadyStatus")
		return
	}

	//handle the case of a database pod going to Not Ready that has
	//autofail enabled
	autofailEnabled := c.checkAutofailLabel(newpod, newpod.ObjectMeta.Namespace)
	clusterName := newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]
	if newpod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == clusterName &&
		clusterName != "" && autofailEnabled {
		log.Debugf("autofail pg-cluster %s updated!", clusterName)
		var oldStatus = false
		for _, v := range oldpod.Status.ContainerStatuses {
			if v.Name == "database" {
				oldStatus = v.Ready
			}
		}
		for _, v := range newpod.Status.ContainerStatuses {
			if v.Name == "database" {
				if !v.Ready && oldStatus {
					log.Debugf("podController autofail enabled pod went from ready to not ready pod name %s", newpod.Name)
					publishPrimaryNotReady(clusterName, newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], newpod.ObjectMeta.Labels[config.LABEL_PGOUSER], newpod.ObjectMeta.Namespace)
					clusteroperator.AutofailBase(c.PodClientset, c.PodClient, v.Ready, clusterName, newpod.ObjectMeta.Namespace)
				}
				//clusteroperator.AutofailBase(c.PodClientset, c.PodClient, v.Ready, clusterName, newpod.ObjectMeta.Namespace)
			}
		}
	}

	//handle applying policies, and updating workflow after a database  pod
	//is made Ready, in the case of backrest, create the create stanza job
	if newpod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == clusterName {
		var oldDatabaseStatus bool
		for _, v := range oldpod.Status.ContainerStatuses {
			if v.Name == "database" {
				oldDatabaseStatus = v.Ready
			}
		}
		for _, v := range newpod.Status.ContainerStatuses {
			if v.Name == "database" {
				//see if there are pgtasks for adding a policy
				if oldDatabaseStatus == false && v.Ready {
					log.Debugf("%s went to Ready from Not Ready, apply policies...", clusterName)
					taskoperator.ApplyPolicies(clusterName, c.PodClientset, c.PodClient, newpod.ObjectMeta.Namespace)

					taskoperator.CompleteCreateClusterWorkflow(clusterName, c.PodClientset, c.PodClient, newpod.ObjectMeta.Namespace)

					//publish event for cluster complete
					publishClusterComplete(clusterName, newpod.ObjectMeta.Namespace, cluster)
					//

					if cluster.Labels[config.LABEL_BACKREST] == "true" {
						tmptask := crv1.Pgtask{}
						found, err := kubeapi.Getpgtask(c.PodClient, &tmptask, clusterName+"-stanza-create", newpod.ObjectMeta.Namespace)
						if !found && err != nil {
							backrestoperator.StanzaCreate(newpod.ObjectMeta.Namespace, clusterName, c.PodClientset, c.PodClient)
						}
					}

					// if this is a pgbouncer enabled cluster, add authorizations to the database.
					if cluster.Labels[config.LABEL_PGBOUNCER] == "true" {
						taskoperator.UpdatePgBouncerAuthorizations(clusterName, c.PodClientset, c.PodClient, newpod.ObjectMeta.Namespace,
							newpod.Status.PodIP)
					}

				}
			}
		}
	}

}

// checkPostgresPods
// see if this is a primary or replica being created
// update service-name label on the pod for each case
// to match the correct Service selector for the PG cluster
func (c *PodController) checkPostgresPods(newpod *apiv1.Pod, ns string) {

	depName := newpod.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME]

	replica := false
	pgreplica := crv1.Pgreplica{}
	found, err := kubeapi.Getpgreplica(c.PodClient, &pgreplica, depName, ns)
	if found {
		replica = true
	}
	log.Debugf("checkPostgresPods --- dep %s replica %t", depName, replica)

	var dep *v1.Deployment
	dep, _, err = kubeapi.GetDeployment(c.PodClientset, depName, ns)
	if err != nil {
		log.Errorf("could not get Deployment on pod Add %s", newpod.Name)
		return
	}

	serviceName := ""

	if dep.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] != "" {
		log.Debug("this means the deployment was already labeled")
		log.Debug("which means its pod was restarted for some reason")
		log.Debug("we will use the service name on the deployment")
		serviceName = dep.ObjectMeta.Labels[config.LABEL_SERVICE_NAME]
	} else if replica == false {
		log.Debugf("primary pod ADDED %s service-name=%s", newpod.Name, newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER])
		//add label onto pod "service-name=clustername"
		serviceName = newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]
	} else if replica == true {
		log.Debugf("replica pod ADDED %s service-name=%s", newpod.Name, newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]+"-replica")
		//add label onto pod "service-name=clustername-replica"
		serviceName = newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] + "-replica"
	}

	err = kubeapi.AddLabelToPod(c.PodClientset, newpod, config.LABEL_SERVICE_NAME, serviceName, ns)
	if err != nil {
		log.Error(err)
		log.Errorf(" could not add pod label for pod %s and label %s ...", newpod.Name, serviceName)
		return
	}

	//add the service name label to the Deployment
	err = kubeapi.AddLabelToDeployment(c.PodClientset, dep, config.LABEL_SERVICE_NAME, serviceName, ns)

	if err != nil {
		log.Error("could not add label to deployment on pod add")
		return
	}

}

//check for the autofail flag on the pgcluster CRD
func (c *PodController) checkAutofailLabel(newpod *apiv1.Pod, ns string) bool {
	clusterName := newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]

	pgcluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(c.PodClient, &pgcluster, clusterName, ns)
	if !found {
		return false
	} else if err != nil {
		log.Error(err)
		return false
	}

	if pgcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] == "true" {
		log.Debugf("autofail is on for this pod %s", newpod.Name)
		return true
	}
	return false

}

func isPostgresPod(newpod *apiv1.Pod) bool {
	if newpod.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] == "true" {
		log.Debugf("pgo-backrest-repo found %s", newpod.Name)
		return false
	}
	if newpod.ObjectMeta.Labels[config.LABEL_JOB_NAME] != "" {
		log.Debugf("job pod found [%s]", newpod.Name)
		return false
	}
	if newpod.ObjectMeta.Labels[config.LABEL_NAME] == "postgres-operator" {
		log.Debugf("postgres-operator-pod found [%s]", newpod.Name)
		return false
	}
	if newpod.ObjectMeta.Labels[config.LABEL_PGPOOL_POD] == "true" {
		log.Debugf("pgpool pod found [%s]", newpod.Name)
		return false
	}
	if newpod.ObjectMeta.Labels[config.LABEL_PGBOUNCER] == "true" {
		log.Debugf("pgbouncer pod found [%s]", newpod.Name)
		return false
	}
	return true
}
// isUpgradedPostgresPod - determines if the pod is one that could be getting a minor upgrade
// differs from above check in that the backrest repo pod is upgradeable. 
func isUpgradedPostgresPod(newpod *apiv1.Pod, oldPod *apiv1.Pod) bool {


	clusterName := newpod.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]
	replicaServiceName := clusterName + "-replica"

	var podIsReady bool
	for _, v := range newpod.Status.ContainerStatuses {
		if v.Name == "database" {
			podIsReady = v.Ready
		}
	}

	var oldPodStatus bool 
	for _, v := range oldPod.Status.ContainerStatuses {
		if v.Name == "database" {
			oldPodStatus = v.Ready
		}
	}

	log.Debugf("[isUpgradedPostgesPod] oldstatus: %s newstatus: %s ", oldPodStatus, podIsReady)

	// only care about pods that have changed from !ready to ready
	if podIsReady && !oldPodStatus {

		// eliminate anything we don't care about - it will be most things
		if newpod.ObjectMeta.Labels[config.LABEL_JOB_NAME] != "" {
			log.Debugf("job pod found [%s]", newpod.Name)
			return false
		}

		if newpod.ObjectMeta.Labels[config.LABEL_NAME] == "postgres-operator" {
			log.Debugf("postgres-operator-pod found [%s]", newpod.Name)
			return false
		}
		if newpod.ObjectMeta.Labels[config.LABEL_PGPOOL_POD] == "true" {
			log.Debugf("pgpool pod found [%s]", newpod.Name)
			return false
		}
		if newpod.ObjectMeta.Labels[config.LABEL_PGBOUNCER] == "true" {
			log.Debugf("pgbouncer pod found [%s]", newpod.Name)
			return false
		}

		// look for specific pods that could have just gone through upgrade

		if newpod.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] == "true"  {
			log.Debugf("Minor Upgrade: upgraded pgo-backrest-repo found %s", newpod.Name)
			return true
		}

		// primary identified by service-name being same as cluster name
		if newpod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == clusterName {
			log.Debugf("Minor Upgrade: upgraded primary found %s", newpod.Name)
			return true
		}

		if newpod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == replicaServiceName {
			log.Debugf("Minor Upgrade: upgraded replica found %s", newpod.Name)
			return true
		}

		// This indicates there is a pod we didn't account for - shouldn't be the case
		log.Debugf(" **** Minor Upgrade: unexpected isUpgraded pod found: [%s] ****", newpod.Name)
	}
	return false
}

func publishClusterComplete(clusterName, namespace string, cluster *crv1.Pgcluster) error {
	//capture the cluster creation event
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventCreateClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  cluster.Spec.UserLabels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventCreateClusterCompleted,
		},
		Clustername:       clusterName,
		Clusteridentifier: cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER],
		WorkflowID:        cluster.Spec.UserLabels[config.LABEL_WORKFLOW_ID],
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return err

}

func (c *PodController) SetupWatch(ns string) {

	// don't create informer for namespace if one has already been created
	c.informerNsMutex.Lock()
	defer c.informerNsMutex.Unlock()
	if _, ok := c.InformerNamespaces[ns]; ok {
		return
	}
	c.InformerNamespaces[ns] = struct{}{}

	source := cache.NewListWatchFromClient(
		c.PodClientset.CoreV1().RESTClient(),
		"pods",
		ns,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&apiv1.Pod{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})

	go controller.Run(c.Ctx.Done())
	log.Debugf("PodController created informer for namespace %s", ns)
}

func publishPrimaryNotReady(clusterName, identifier, username, namespace string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventPrimaryNotReadyFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventPrimaryNotReady,
		},
		Clustername:       clusterName,
		Clusteridentifier: identifier,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}
}
