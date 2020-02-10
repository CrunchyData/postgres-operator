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

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
	backrestoperator "github.com/crunchydata/postgres-operator/operator/backrest"
	pgbasebackupoperator "github.com/crunchydata/postgres-operator/operator/backup"
	benchmarkoperator "github.com/crunchydata/postgres-operator/operator/benchmark"
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	pgdumpoperator "github.com/crunchydata/postgres-operator/operator/pgdump"
	taskoperator "github.com/crunchydata/postgres-operator/operator/task"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// PgtaskController holds connections for the controller
type PgtaskController struct {
	PgtaskConfig       *rest.Config
	PgtaskClient       *rest.RESTClient
	PgtaskScheme       *runtime.Scheme
	PgtaskClientset    *kubernetes.Clientset
	Queue              workqueue.RateLimitingInterface
	Ctx                context.Context
	informerNsMutex    sync.Mutex
	InformerNamespaces map[string]struct{}
}

// Run starts an pgtask resource controller
func (c *PgtaskController) Run() error {
	log.Debug("Watch Pgtask objects")

	//shut down the work queue to cause workers to end
	defer c.Queue.ShutDown()

	// Watch Example objects
	err := c.watchPgtasks(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgtask resource: %v", err)
		return err
	}

	<-c.Ctx.Done()
	return c.Ctx.Err()
}

// watchPgtasks watches the pgtask resource catching events
func (c *PgtaskController) watchPgtasks(ctx context.Context) error {
	nsList := ns.GetNamespaces(c.PgtaskClientset, operator.InstallationName)

	for i := 0; i < len(nsList); i++ {
		log.Infof("starting pgtask controller on ns [%s]", nsList[i])

		c.SetupWatch(nsList[i])
	}
	return nil
}

func (c *PgtaskController) RunWorker() {

	//process the 'add' work queue forever
	for c.processNextItem() {
	}
}

func (c *PgtaskController) processNextItem() bool {
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

	tmpTask := crv1.Pgtask{}
	found, err := kubeapi.Getpgtask(c.PgtaskClient, &tmpTask, keyResourceName, keyNamespace)
	if !found {
		log.Errorf("ERROR onAdd getting pgtask : %s", err.Error())
		return false
	}

	//update pgtask
	state := crv1.PgtaskStateProcessed
	message := "Successfully processed Pgtask by controller"
	err = kubeapi.PatchpgtaskStatus(c.PgtaskClient, state, message, &tmpTask, keyNamespace)
	if err != nil {
		log.Errorf("ERROR onAdd updating pgtask status: %s", err.Error())
		return false
	}

	//process the incoming task
	switch tmpTask.Spec.TaskType {
	case crv1.PgtaskMinorUpgrade:
		log.Debug("delete minor upgrade task added")
		clusteroperator.AddUpgrade(c.PgtaskClientset, c.PgtaskClient, &tmpTask, keyNamespace)
	case crv1.PgtaskDeletePgbouncer:
		log.Debug("delete pgbouncer task added")
		clusteroperator.DeletePgbouncerFromTask(c.PgtaskClientset, c.PgtaskClient, &tmpTask, keyNamespace)
	case crv1.PgtaskReconfigurePgbouncer:
		log.Debug("Reconfiguredelete pgbouncer task added")
		clusteroperator.ReconfigurePgbouncerFromTask(c.PgtaskClientset, c.PgtaskClient, &tmpTask, keyNamespace)
	case crv1.PgtaskAddPgbouncer:
		log.Debug("add pgbouncer task added")
		clusteroperator.AddPgbouncerFromTask(c.PgtaskClientset, c.PgtaskClient, &tmpTask, keyNamespace)
	case crv1.PgtaskFailover:
		log.Debug("failover task added")
		if !dupeFailover(c.PgtaskClient, &tmpTask, keyNamespace) {
			clusteroperator.FailoverBase(keyNamespace, c.PgtaskClientset, c.PgtaskClient, &tmpTask, c.PgtaskConfig)
		} else {
			log.Debug("skipping duplicate onAdd failover task %s/%s", keyNamespace, keyResourceName)
		}

	case crv1.PgtaskDeleteData:
		log.Debug("delete data task added")
		if !dupeDeleteData(c.PgtaskClient, &tmpTask, keyNamespace) {
			taskoperator.RemoveData(keyNamespace, c.PgtaskClientset, c.PgtaskClient, &tmpTask)
		} else {
			log.Debug("skipping duplicate onAdd delete data task %s/%s", keyNamespace, keyResourceName)
		}
	case crv1.PgtaskDeleteBackups:
		log.Debug("delete backups task added")
		taskoperator.RemoveBackups(keyNamespace, c.PgtaskClientset, &tmpTask)
	case crv1.PgtaskBackrest:
		log.Debug("backrest task added")
		backrestoperator.Backrest(keyNamespace, c.PgtaskClientset, &tmpTask)
	case crv1.PgtaskBackrestRestore:
		log.Debug("backrest restore task added")
		backrestoperator.Restore(c.PgtaskClient, keyNamespace, c.PgtaskClientset, &tmpTask)

	case crv1.PgtaskpgDump:
		log.Debug("pgDump task added")
		pgdumpoperator.Dump(keyNamespace, c.PgtaskClientset, c.PgtaskClient, &tmpTask)
	case crv1.PgtaskpgRestore:
		log.Debug("pgDump restore task added")
		pgdumpoperator.Restore(keyNamespace, c.PgtaskClientset, c.PgtaskClient, &tmpTask)

	case crv1.PgtaskpgBasebackupRestore:
		log.Debug("pgbasebackup restore task added")
		pgbasebackupoperator.Restore(c.PgtaskClient, keyNamespace, c.PgtaskClientset, &tmpTask)

	case crv1.PgtaskAutoFailover:
		log.Debugf("autofailover task added %s", keyResourceName)
	case crv1.PgtaskWorkflow:
		log.Debugf("workflow task added [%s] ID [%s]", keyResourceName, tmpTask.Spec.Parameters[crv1.PgtaskWorkflowID])

	case crv1.PgtaskBenchmark:
		log.Debug("benchmark task added")
		benchmarkoperator.Create(keyNamespace, c.PgtaskClientset, c.PgtaskClient, &tmpTask)

	case crv1.PgtaskUpdatePgbouncerAuths:
		log.Debug("Pgbouncer update credential task was found...will be handled by pod controller when ready")

	case crv1.PgtaskCloneStep1, crv1.PgtaskCloneStep2, crv1.PgtaskCloneStep3:
		log.Debug("clone task added [%s]", keyResourceName)
		clusteroperator.Clone(c.PgtaskClientset, c.PgtaskClient, keyNamespace, &tmpTask)

	default:
		log.Debugf("unknown task type on pgtask added [%s]", tmpTask.Spec.TaskType)
	}

	return true

}

// onAdd is called when a pgtask is added
func (c *PgtaskController) onAdd(obj interface{}) {
	task := obj.(*crv1.Pgtask)
	//	log.Debugf("[PgtaskController] onAdd ns=%s %s", task.ObjectMeta.Namespace, task.ObjectMeta.SelfLink)

	//handle the case of when the operator restarts, we do not want
	//to process pgtasks already processed
	if task.Status.State == crv1.PgtaskStateProcessed {
		log.Debug("pgtask " + task.ObjectMeta.Name + " already processed")
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		log.Debugf("task putting key in queue %s", key)
		c.Queue.Add(key)
	}

}

// onUpdate is called when a pgtask is updated
func (c *PgtaskController) onUpdate(oldObj, newObj interface{}) {
	//task := newObj.(*crv1.Pgtask)
	//	log.Debugf("[PgtaskController] onUpdate ns=%s %s", task.ObjectMeta.Namespace, task.ObjectMeta.SelfLink)
}

// onDelete is called when a pgtask is deleted
func (c *PgtaskController) onDelete(obj interface{}) {
	//task := obj.(*crv1.Pgtask)
	//	log.Debugf("[PgtaskController] onDelete ns=%s %s", task.ObjectMeta.Namespace, task.ObjectMeta.SelfLink)
}

func (c *PgtaskController) SetupWatch(ns string) {

	// don't create informer for namespace if one has already been created
	c.informerNsMutex.Lock()
	defer c.informerNsMutex.Unlock()
	if _, ok := c.InformerNamespaces[ns]; ok {
		return
	}
	c.InformerNamespaces[ns] = struct{}{}

	source := cache.NewListWatchFromClient(
		c.PgtaskClient,
		crv1.PgtaskResourcePlural,
		ns,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&crv1.Pgtask{},

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
	log.Debugf("PgtaskController created informer for namespace %s", ns)
}

//de-dupe logic for a failover, if the failover started
//parameter is set, it means a failover has already been
//started on this
func dupeFailover(restClient *rest.RESTClient, task *crv1.Pgtask, ns string) bool {
	tmp := crv1.Pgtask{}

	found, _ := kubeapi.Getpgtask(restClient, &tmp, task.Spec.Name, ns)
	if !found {
		//a big time error if this occurs
		return false
	}

	if tmp.Spec.Parameters[config.LABEL_FAILOVER_STARTED] == "" {
		return false
	}

	return true
}

//de-dupe logic for a delete data, if the delete data job started
//parameter is set, it means a delete data job has already been
//started on this
func dupeDeleteData(restClient *rest.RESTClient, task *crv1.Pgtask, ns string) bool {
	tmp := crv1.Pgtask{}

	found, _ := kubeapi.Getpgtask(restClient, &tmp, task.Spec.Name, ns)
	if !found {
		//a big time error if this occurs
		return false
	}

	if tmp.Spec.Parameters[config.LABEL_DELETE_DATA_STARTED] == "" {
		return false
	}

	return true
}
