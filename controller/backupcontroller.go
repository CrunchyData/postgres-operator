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
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	//	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
	backupoperator "github.com/crunchydata/postgres-operator/operator/backup"
	"k8s.io/client-go/util/workqueue"
)

// PgbackupController holds connections required by the controller
type PgbackupController struct {
	PgbackupClient     *rest.RESTClient
	PgbackupScheme     *runtime.Scheme
	PgbackupClientset  *kubernetes.Clientset
	Ctx                context.Context
	Queue              workqueue.RateLimitingInterface
	UpdateQueue        workqueue.RateLimitingInterface
	informerNsMutex    sync.Mutex
	InformerNamespaces map[string]struct{}
}

// Run starts controller
func (c *PgbackupController) Run() error {
	log.Debugf("Watch Pgbackup objects")

	//shut down the work queue to cause workers to end
	defer c.Queue.ShutDown()
	defer c.UpdateQueue.ShutDown()

	err := c.watchPgbackups(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgbackup resource: %v", err)
		return err
	}

	<-c.Ctx.Done()

	log.Debugf("Watch Pgbackup ending")

	return c.Ctx.Err()
}

// watchPgbackups will watch events for the pgbackups
func (c *PgbackupController) watchPgbackups(ctx context.Context) error {
	nsList := ns.GetNamespaces(c.PgbackupClientset, operator.InstallationName)

	for i := 0; i < len(nsList); i++ {
		log.Infof("starting pgbackup controller on ns [%s]", nsList[i])

		c.SetupWatch(nsList[i])
	}
	return nil
}

func (c *PgbackupController) RunWorker() {

	//process the 'add' work queue forever
	for c.processNextItem() {
	}
}

func (c *PgbackupController) RunUpdateWorker() {

	//process the 'add' work queue forever
	for c.processNextUpdateItem() {
	}
}

func (c *PgbackupController) processNextUpdateItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.UpdateQueue.Get()
	if quit {
		return false
	}

	log.Debugf("working on %s", key.(string))
	keyParts := strings.Split(key.(string), "/")
	keyNamespace := keyParts[0]
	keyResourceName := keyParts[1]

	log.Debugf("update queue got key ns=[%s] resource=[%s]", keyNamespace, keyResourceName)

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.UpdateQueue.Done(key)

	// Invoke the method containing the business logic
	// for pgbackups, the convention is the CRD name is always
	// the same as the pg-cluster label value

	// in this case, the de-dupe logic is to test whether a backup
	// job is already running, if so, then we don't create another
	// backup job
	selector := "pg-cluster=" + keyResourceName + ",pgbackup=true"
	jobs, err := kubeapi.GetJobs(c.PgbackupClientset, selector, keyNamespace)
	if err != nil {
		log.Errorf("update working...error found " + err.Error())
		return true
	}

	jobRunning := false
	for _, j := range jobs.Items {
		if j.Status.Succeeded <= 0 {
			jobRunning = true
		}
	}

	if jobRunning {
		log.Debugf("update working...found job already, would do nothing")
	} else {
		log.Debugf("update working...no job found, means we process")
		b := crv1.Pgbackup{}
		found, err := kubeapi.Getpgbackup(c.PgbackupClient, &b, keyResourceName, keyNamespace)
		if found {
			state := crv1.PgbackupStateProcessed
			message := "Successfully processed Pgbackup by controller"
			err = kubeapi.PatchpgbackupStatus(c.PgbackupClient, state, message, &b, b.ObjectMeta.Namespace)
			if err != nil {
				log.Errorf("ERROR updating pgbackup status: %s", err.Error())
			}

			backupoperator.AddBackupBase(c.PgbackupClientset, c.PgbackupClient, &b, b.ObjectMeta.Namespace)

			//no error, tell the queue to stop tracking history
			c.UpdateQueue.Forget(key)
		}
	}

	return true
}

func (c *PgbackupController) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	log.Debugf("working on %s", key.(string))
	keyParts := strings.Split(key.(string), "/")
	keyNamespace := keyParts[0]
	keyResourceName := keyParts[1]

	log.Debugf("queue got key ns=[%s] resource=[%s]", keyNamespace, keyResourceName)

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.Queue.Done(key)

	// Invoke the method containing the business logic
	// for pgbackups, the convention is the CRD name is always
	// the same as the pg-cluster label value

	// in this case, the de-dupe logic is to test whether a backup
	// job is already running, if so, then we don't create another
	// backup job
	selector := "pg-cluster=" + keyResourceName + ",pgbackup=true"
	jobs, err := kubeapi.GetJobs(c.PgbackupClientset, selector, keyNamespace)
	if err != nil {
		log.Errorf("working...error found " + err.Error())
		return true
	}

	jobRunning := false
	for _, j := range jobs.Items {
		if j.Status.Succeeded <= 0 {
			jobRunning = true
		}
	}

	if jobRunning {
		log.Debugf("working...found job already, would do nothing")
	} else {
		log.Debugf("working...no job found, means we process")
		b := crv1.Pgbackup{}
		found, err := kubeapi.Getpgbackup(c.PgbackupClient, &b, keyResourceName, keyNamespace)
		if found {
			state := crv1.PgbackupStateProcessed
			message := "Successfully processed Pgbackup by controller"
			err = kubeapi.PatchpgbackupStatus(c.PgbackupClient, state, message, &b, b.ObjectMeta.Namespace)
			if err != nil {
				log.Errorf("ERROR updating pgbackup status: %s", err.Error())
			}
			backupoperator.AddBackupBase(c.PgbackupClientset, c.PgbackupClient, &b, b.ObjectMeta.Namespace)

			//no error, tell the queue to stop tracking history
			c.Queue.Forget(key)
		}
	}
	return true
}

// onAdd is called when a pgbackup is added
func (c *PgbackupController) onAdd(obj interface{}) {

	backup := obj.(*crv1.Pgbackup)
	log.Debugf("[PgbackupController] ns=%s onAdd %s", backup.ObjectMeta.Namespace, backup.ObjectMeta.SelfLink)

	//the case when the operator starts up, we disregard any
	//pgbackups that have already been processed
	if backup.Status.State == crv1.PgbackupStateProcessed {
		log.Debug("pgbackup " + backup.ObjectMeta.Name + " already processed")
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		log.Debugf("[PgbackupController] putting key in queue %s", key)
		c.Queue.Add(key)
	}

}

// onUpdate is called when a pgbackup is updated
func (c *PgbackupController) onUpdate(oldObj, newObj interface{}) {
	oldBackup := oldObj.(*crv1.Pgbackup)
	backup := newObj.(*crv1.Pgbackup)
	log.Debugf("[PgbackupController] ns=%s onUpdate %s", backup.ObjectMeta.Namespace, backup.ObjectMeta.SelfLink)

	if oldBackup.Spec.BackupStatus != crv1.PgBackupJobReSubmitted &&
		backup.Spec.BackupStatus == crv1.PgBackupJobReSubmitted {
		log.Debugf("[PgbackupController] ns=%s onUpdate %s re-submitted", backup.ObjectMeta.Namespace, backup.ObjectMeta.SelfLink)

		key, err := cache.MetaNamespaceKeyFunc(oldObj)
		if err == nil {
			log.Debugf("[PgbackupController] putting key in update queue %s", key)
			c.UpdateQueue.Add(key)
		}
	}

}

// onDelete is called when a pgbackup is deleted
func (c *PgbackupController) onDelete(obj interface{}) {
	backup := obj.(*crv1.Pgbackup)
	log.Debugf("[PgbackupController] ns=%s onDelete %s", backup.ObjectMeta.Namespace, backup.ObjectMeta.SelfLink)
}

func (c *PgbackupController) SetupWatch(ns string) {

	// don't create informer for namespace if one has already been created
	c.informerNsMutex.Lock()
	defer c.informerNsMutex.Unlock()
	if _, ok := c.InformerNamespaces[ns]; ok {
		return
	}
	c.InformerNamespaces[ns] = struct{}{}

	source := cache.NewListWatchFromClient(
		c.PgbackupClient,
		crv1.PgbackupResourcePlural,
		ns,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&crv1.Pgbackup{},

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
	log.Debugf("PgbackupController: created informer for namespace %s", ns)
}
