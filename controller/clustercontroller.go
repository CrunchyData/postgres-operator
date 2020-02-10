package controller

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
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"

	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// PgclusterController holds the connections for the controller
type PgclusterController struct {
	PgclusterClient    *rest.RESTClient
	PgclusterScheme    *runtime.Scheme
	PgclusterClientset *kubernetes.Clientset
	Queue              workqueue.RateLimitingInterface
	Ctx                context.Context
	informerNsMutex    sync.Mutex
	InformerNamespaces map[string]struct{}
}

// Run starts an pgcluster resource controller
func (c *PgclusterController) Run() error {
	log.Debug("Watch Pgcluster objects")

	//shut down the work queue to cause workers to end
	defer c.Queue.ShutDown()

	err := c.watchPgclusters(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgcluster resource: %v", err)
		return err
	}

	<-c.Ctx.Done()

	return c.Ctx.Err()
}

// watchPgclusters is the event loop for pgcluster resources
func (c *PgclusterController) watchPgclusters(ctx context.Context) error {
	nsList := ns.GetNamespaces(c.PgclusterClientset, operator.InstallationName)

	for i := 0; i < len(nsList); i++ {
		log.Infof("starting pgcluster controller for ns [%s]", nsList[i])
		c.SetupWatch(nsList[i])
	}
	return nil
}

// onAdd is called when a pgcluster is added
func (c *PgclusterController) onAdd(obj interface{}) {
	cluster := obj.(*crv1.Pgcluster)
	log.Debugf("[PgclusterController] ns %s onAdd %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	//handle the case when the operator restarts and don't
	//process already processed pgclusters
	if cluster.Status.State == crv1.PgclusterStateProcessed {
		log.Debug("pgcluster " + cluster.ObjectMeta.Name + " already processed")
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		log.Debugf("cluster putting key in queue %s", key)
		c.Queue.Add(key)
	}

}

func (c *PgclusterController) RunWorker() {

	//process the 'add' work queue forever
	for c.processNextItem() {
	}
}

func (c *PgclusterController) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	log.Debugf("working on %s", key.(string))
	keyParts := strings.Split(key.(string), "/")
	keyNamespace := keyParts[0]
	keyResourceName := keyParts[1]

	log.Debugf("cluster add queue got key ns=[%s] resource=[%s]", keyNamespace, keyResourceName)

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.Queue.Done(key)

	// Invoke the method containing the business logic
	// for pgbackups, the convention is the CRD name is always
	// the same as the pg-cluster label value

	// in this case, the de-dupe logic is to test whether a cluster
	// deployment exists , if so, then we don't create another
	_, found, err := kubeapi.GetDeployment(c.PgclusterClientset, keyResourceName, keyNamespace)

	if found {
		log.Debugf("cluster add - dep already found, not creating again")
		return true
	}

	//get the pgcluster
	cluster := crv1.Pgcluster{}
	found, err = kubeapi.Getpgcluster(c.PgclusterClient, &cluster, keyResourceName, keyNamespace)
	if !found {
		log.Debugf("cluster add - pgcluster not found, this is invalid")
		return false
	}

	addIdentifier(&cluster)

	state := crv1.PgclusterStateProcessed
	message := "Successfully processed Pgcluster by controller"
	err = kubeapi.PatchpgclusterStatus(c.PgclusterClient, state, message, &cluster, keyNamespace)
	if err != nil {
		log.Errorf("ERROR updating pgcluster status on add: %s", err.Error())
		return false
	}

	log.Debugf("pgcluster added: %s", cluster.ObjectMeta.Name)

	clusteroperator.AddClusterBase(c.PgclusterClientset, c.PgclusterClient, &cluster, cluster.ObjectMeta.Namespace)

	return true
}

// onUpdate is called when a pgcluster is updated
func (c *PgclusterController) onUpdate(oldObj, newObj interface{}) {
	oldcluster := oldObj.(*crv1.Pgcluster)
	newcluster := newObj.(*crv1.Pgcluster)
	//	log.Debugf("pgcluster ns=%s %s onUpdate", newcluster.ObjectMeta.Namespace, newcluster.ObjectMeta.Name)

	// check to see if the "autofail" label on the pgcluster CR has been changed from either true to false, or from
	// false to true.  If it has been changed to false, autofail will then be disabled in the pg cluster.  If has
	// been changed to true, autofail will then be enabled in the pg cluster
	if newcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] != "" {
		autofailEnabledOld, err := strconv.ParseBool(oldcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL])
		if err != nil {
			log.Error(err)
			return
		}
		autofailEnabledNew, err := strconv.ParseBool(newcluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL])
		if err != nil {
			log.Error(err)
			return
		}
		if autofailEnabledNew != autofailEnabledOld {
			util.ToggleAutoFailover(c.PgclusterClientset, autofailEnabledNew,
				newcluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE],
				newcluster.ObjectMeta.Namespace)
		}

	}

}

// onDelete is called when a pgcluster is deleted
func (c *PgclusterController) onDelete(obj interface{}) {
	//cluster := obj.(*crv1.Pgcluster)
	//	log.Debugf("[PgclusterController] ns=%s onDelete %s", cluster.ObjectMeta.Namespace, cluster.ObjectMeta.SelfLink)

	//handle pgcluster cleanup
	//	clusteroperator.DeleteClusterBase(c.PgclusterClientset, c.PgclusterClient, cluster, cluster.ObjectMeta.Namespace)
}

func GetPrimaryPodStatus(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster, ns string) (error, bool) {
	var ready bool
	var err error

	selector := config.LABEL_SERVICE_NAME + "=" + cluster.Name
	pods, err := kubeapi.GetPods(clientset, selector, ns)
	if err != nil {
		return err, ready
	}
	if len(pods.Items) == 0 {
		log.Error("GetPrimaryPodStatus found no primary pod for %s using %s", cluster.Name, selector)
		return err, ready
	}
	if len(pods.Items) > 1 {
		log.Error("GetPrimaryPodStatus found more than 1 primary pod for %s using %s", cluster.Name, selector)
		return err, ready
	}

	pod := pods.Items[0]
	var readyStatus string
	readyStatus, ready = getReadyStatus(&pod)
	log.Debugf("readyStatus found to be %s", readyStatus)
	return err, ready

}

//this code is taken from apiserver/cluster/clusterimpl.go, need
//to refactor into a higher level package to share the code
func getReadyStatus(pod *v1.Pod) (string, bool) {
	equal := false
	readyCount := 0
	containerCount := 0
	for _, stat := range pod.Status.ContainerStatuses {
		containerCount++
		if stat.Ready {
			readyCount++
		}
	}
	if readyCount == containerCount {
		equal = true
	}
	return fmt.Sprintf("%d/%d", readyCount, containerCount), equal

}

func (c *PgclusterController) SetupWatch(ns string) {

	// don't create informer for namespace if one has already been created
	c.informerNsMutex.Lock()
	defer c.informerNsMutex.Unlock()
	if _, ok := c.InformerNamespaces[ns]; ok {
		return
	}
	c.InformerNamespaces[ns] = struct{}{}

	source := cache.NewListWatchFromClient(
		c.PgclusterClient,
		crv1.PgclusterResourcePlural,
		ns,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&crv1.Pgcluster{},

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
	log.Debugf("PgclusterController created informer for namespace %s", ns)
}

func addIdentifier(clusterCopy *crv1.Pgcluster) {
	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error(err)
	}

	clusterCopy.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = string(u[:len(u)-1])
}
