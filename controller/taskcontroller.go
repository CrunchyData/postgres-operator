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

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
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
)

// PgtaskController holds connections for the controller
type PgtaskController struct {
	PgtaskConfig    *rest.Config
	PgtaskClient    *rest.RESTClient
	PgtaskScheme    *runtime.Scheme
	PgtaskClientset *kubernetes.Clientset
	Ctx             context.Context
}

// Run starts an pgtask resource controller
func (c *PgtaskController) Run() error {
	log.Debug("Watch Pgtask objects")

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

// onAdd is called when a pgtask is added
func (c *PgtaskController) onAdd(obj interface{}) {
	task := obj.(*crv1.Pgtask)
	log.Debugf("[PgtaskController] onAdd ns=%s %s", task.ObjectMeta.Namespace, task.ObjectMeta.SelfLink)

	//handle the case of when the operator restarts, we do not want
	//to process pgtasks already processed
	if task.Status.State == crv1.PgtaskStateProcessed {
		log.Debug("pgtask " + task.ObjectMeta.Name + " already processed")
		return
	}

	//time.Sleep(time.Second * time.Duration(2))

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use taskScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj := task.DeepCopyObject()
	taskCopy := copyObj.(*crv1.Pgtask)

	//update the status of the task as processed to prevent reprocessing
	taskCopy.Status = crv1.PgtaskStatus{
		State:   crv1.PgtaskStateProcessed,
		Message: "Successfully processed Pgtask by controller",
	}
	task.Status = crv1.PgtaskStatus{
		State:   crv1.PgtaskStateProcessed,
		Message: "Successfully processed Pgtask by controller",
	}

	//Body(taskCopy).

	//get pgtask

	tmpTask := crv1.Pgtask{}
	found, err := kubeapi.Getpgtask(c.PgtaskClient, &tmpTask, task.ObjectMeta.Name, task.ObjectMeta.Namespace)
	if !found {
		log.Errorf("ERROR onAdd getting pgtask : %s", err.Error())
		return
	}

	//update pgtask
	tmpTask.Status = crv1.PgtaskStatus{
		State:   crv1.PgtaskStateProcessed,
		Message: "Successfully processed Pgtask by controller",
	}

	err = kubeapi.Updatepgtask(c.PgtaskClient, &tmpTask, task.ObjectMeta.Name, task.ObjectMeta.Namespace)

	/**
	err = c.PgtaskClient.Put().
		Name(tmpTask.ObjectMeta.Name).
		Namespace(tmpTask.ObjectMeta.Namespace).
		Resource(crv1.PgtaskResourcePlural).
		Body(tmpTask).
		Do().
		Error()

	*/
	if err != nil {
		log.Errorf("ERROR onAdd updating pgtask status: %s", err.Error())
		return
	}

	//process the incoming task
	switch task.Spec.TaskType {
	case crv1.PgtaskMinorUpgrade:
		log.Debug("delete minor upgrade task added")
		clusteroperator.AddUpgrade(c.PgtaskClientset, c.PgtaskClient, task, task.ObjectMeta.Namespace)
	case crv1.PgtaskDeletePgbouncer:
		log.Debug("delete pgbouncer task added")
		clusteroperator.DeletePgbouncerFromTask(c.PgtaskClientset, c.PgtaskClient, task, task.ObjectMeta.Namespace)
	case crv1.PgtaskReconfigurePgbouncer:
		log.Debug("Reconfiguredelete pgbouncer task added")
		clusteroperator.ReconfigurePgbouncerFromTask(c.PgtaskClientset, c.PgtaskClient, task, task.ObjectMeta.Namespace)
	case crv1.PgtaskAddPgbouncer:
		log.Debug("add pgbouncer task added")
		clusteroperator.AddPgbouncerFromTask(c.PgtaskClientset, c.PgtaskClient, task, task.ObjectMeta.Namespace)
	case crv1.PgtaskDeletePgpool:
		log.Debug("delete pgpool task added")
		clusteroperator.DeletePgpoolFromTask(c.PgtaskClientset, c.PgtaskClient, task, task.ObjectMeta.Namespace)
	case crv1.PgtaskReconfigurePgpool:
		log.Debug("Reconfiguredelete pgpool task added")
		clusteroperator.ReconfigurePgpoolFromTask(c.PgtaskClientset, c.PgtaskClient, task, task.ObjectMeta.Namespace)
	case crv1.PgtaskAddPgpool:
		log.Debug("add pgpool task added")
		clusteroperator.AddPgpoolFromTask(c.PgtaskClientset, c.PgtaskClient, task, task.ObjectMeta.Namespace)
	case crv1.PgtaskFailover:
		log.Debug("failover task added")
		clusteroperator.FailoverBase(task.ObjectMeta.Namespace, c.PgtaskClientset, c.PgtaskClient, task, c.PgtaskConfig)

	case crv1.PgtaskDeleteData:
		log.Debug("delete data task added")
		taskoperator.RemoveData(task.ObjectMeta.Namespace, c.PgtaskClientset, task)
	case crv1.PgtaskDeleteBackups:
		log.Debug("delete backups task added")
		taskoperator.RemoveBackups(task.ObjectMeta.Namespace, c.PgtaskClientset, task)
	case crv1.PgtaskBackrest:
		log.Debug("backrest task added")
		backrestoperator.Backrest(task.ObjectMeta.Namespace, c.PgtaskClientset, task)
	case crv1.PgtaskBackrestRestore:
		log.Debug("backrest restore task added")
		backrestoperator.Restore(c.PgtaskClient, task.ObjectMeta.Namespace, c.PgtaskClientset, task)

	case crv1.PgtaskpgDump:
		log.Debug("pgDump task added")
		pgdumpoperator.Dump(task.ObjectMeta.Namespace, c.PgtaskClientset, c.PgtaskClient, task)
	case crv1.PgtaskpgRestore:
		log.Debug("pgDump restore task added")
		pgdumpoperator.Restore(task.ObjectMeta.Namespace, c.PgtaskClientset, c.PgtaskClient, task)

	case crv1.PgtaskpgBasebackupRestore:
		log.Debug("pgbasebackup restore task added")
		pgbasebackupoperator.Restore(c.PgtaskClient, task.ObjectMeta.Namespace, c.PgtaskClientset, task)

	case crv1.PgtaskAutoFailover:
		log.Debugf("autofailover task added %s", task.ObjectMeta.Name)
	case crv1.PgtaskWorkflow:
		log.Debugf("workflow task added [%s] ID [%s]", task.ObjectMeta.Name, task.Spec.Parameters[crv1.PgtaskWorkflowID])

	case crv1.PgtaskBenchmark:
		log.Debug("benchmark task added")
		benchmarkoperator.Create(task.ObjectMeta.Namespace, c.PgtaskClientset, c.PgtaskClient, task)

	case crv1.PgtaskUpdatePgbouncerAuths:
		log.Debug("Pgbouncer update credential task was found...will be handled by pod controller when ready")

	default:
		log.Debugf("unknown task type on pgtask added [%s]", task.Spec.TaskType)
	}

}

// onUpdate is called when a pgtask is updated
func (c *PgtaskController) onUpdate(oldObj, newObj interface{}) {
	task := newObj.(*crv1.Pgtask)
	log.Debugf("[PgtaskController] onUpdate ns=%s %s", task.ObjectMeta.Namespace, task.ObjectMeta.SelfLink)
}

// onDelete is called when a pgtask is deleted
func (c *PgtaskController) onDelete(obj interface{}) {
	task := obj.(*crv1.Pgtask)
	log.Debugf("[PgtaskController] onDelete ns=%s %s", task.ObjectMeta.Namespace, task.ObjectMeta.SelfLink)
}

func (c *PgtaskController) SetupWatch(ns string) {
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
}
