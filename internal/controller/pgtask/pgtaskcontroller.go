package pgtask

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
	"encoding/json"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	backrestoperator "github.com/crunchydata/postgres-operator/internal/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/internal/operator/cluster"
	pgdumpoperator "github.com/crunchydata/postgres-operator/internal/operator/pgdump"
	taskoperator "github.com/crunchydata/postgres-operator/internal/operator/task"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"
	informers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller holds connections for the controller
type Controller struct {
	Client            *kubeapi.Client
	Queue             workqueue.RateLimitingInterface
	Informer          informers.PgtaskInformer
	PgtaskWorkerCount int
}

// RunWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) RunWorker(stopCh <-chan struct{}, doneCh chan<- struct{}) {
	go c.waitForShutdown(stopCh)

	for c.processNextItem() {
	}

	log.Debug("pgtask Contoller: worker queue has been shutdown, writing to the done channel")
	doneCh <- struct{}{}
}

// waitForShutdown waits for a message on the stop channel and then shuts down the work queue
func (c *Controller) waitForShutdown(stopCh <-chan struct{}) {
	<-stopCh
	c.Queue.ShutDown()
	log.Debug("pgtask Contoller: received stop signal, worker queue told to shutdown")
}

func (c *Controller) processNextItem() bool {
	ctx := context.TODO()

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

	tmpTask, err := c.Client.CrunchydataV1().Pgtasks(keyNamespace).Get(ctx, keyResourceName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("ERROR onAdd getting pgtask : %s", err.Error())
		c.Queue.Forget(key) // NB(cbandy): This should probably be a retry.
		return true
	}

	// update pgtask
	patch, err := json.Marshal(map[string]interface{}{
		"status": crv1.PgtaskStatus{
			State:   crv1.PgtaskStateProcessed,
			Message: "Successfully processed Pgtask by controller",
		},
	})
	if err == nil {
		_, err = c.Client.CrunchydataV1().Pgtasks(keyNamespace).
			Patch(ctx, tmpTask.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Errorf("ERROR onAdd updating pgtask status: %s", err.Error())
		c.Queue.Forget(key) // NB(cbandy): This should probably be a retry.
		return true
	}

	// process the incoming task
	switch tmpTask.Spec.TaskType {
	case crv1.PgtaskPgAdminAdd:
		log.Debug("add pgadmin task added")
		clusteroperator.AddPgAdminFromPgTask(c.Client, c.Client.Config, tmpTask)
	case crv1.PgtaskPgAdminDelete:
		log.Debug("delete pgadmin task added")
		clusteroperator.DeletePgAdminFromPgTask(c.Client, c.Client.Config, tmpTask)
	case crv1.PgtaskUpgrade:
		log.Debug("upgrade task added")
		clusteroperator.AddUpgrade(c.Client, tmpTask, keyNamespace)
	case crv1.PgtaskRollingUpdate:
		log.Debug("rolling update task added")
		// first, attempt to get the pgcluster object
		clusterName := tmpTask.Spec.Parameters[config.LABEL_PG_CLUSTER]

		if cluster, err := c.Client.CrunchydataV1().Pgclusters(tmpTask.Namespace).
			Get(ctx, clusterName, metav1.GetOptions{}); err == nil {
			if err := clusteroperator.RollingUpdate(c.Client, c.Client.Config, cluster,
				func(kubeapi.Interface, *crv1.Pgcluster, *appsv1.Deployment) error { return nil }); err != nil {
				log.Errorf("rolling update failed: %q", err.Error())
			}
		} else {
			log.Debugf("rolling update failed: could not find cluster %q", clusterName)
		}

	case crv1.PgtaskDeleteData:
		log.Debug("delete data task added")
		if !dupeDeleteData(c.Client, tmpTask, keyNamespace) {
			taskoperator.RemoveData(keyNamespace, c.Client, tmpTask)
		} else {
			log.Debugf("skipping duplicate onAdd delete data task %s/%s", keyNamespace, keyResourceName)
		}
	case crv1.PgtaskBackrest:
		log.Debug("backrest task added")
		backrestoperator.Backrest(keyNamespace, c.Client, tmpTask)
	case crv1.PgtaskBackrestRestore:
		log.Debug("backrest restore task added")
		c.handleBackrestRestore(tmpTask)

	case crv1.PgtaskpgDump:
		log.Debug("pgDump task added")
		pgdumpoperator.Dump(keyNamespace, c.Client, tmpTask)
	case crv1.PgtaskpgRestore:
		log.Debug("pgDump restore task added")
		pgdumpoperator.Restore(keyNamespace, c.Client, tmpTask)
	case crv1.PgtaskWorkflow:
		log.Debugf("workflow task added [%s] ID [%s]", keyResourceName, tmpTask.Spec.Parameters[crv1.PgtaskWorkflowID])

	default:
		log.Debugf("unknown task type on pgtask added [%s]", tmpTask.Spec.TaskType)
	}

	c.Queue.Forget(key)
	return true
}

// onAdd is called when a pgtask is added
func (c *Controller) onAdd(obj interface{}) {
	task := obj.(*crv1.Pgtask)

	// handle the case of when the operator restarts, we do not want
	// to process pgtasks already processed
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
func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	// task := newObj.(*crv1.Pgtask)
	//	log.Debugf("[Controller] onUpdate ns=%s %s", task.ObjectMeta.Namespace, task.ObjectMeta.SelfLink)
}

// onDelete is called when a pgtask is deleted
func (c *Controller) onDelete(obj interface{}) {
}

// AddPGTaskEventHandler adds the pgtask event handler to the pgtask informer
func (c *Controller) AddPGTaskEventHandler() {
	c.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	log.Debugf("pgtask Controller: added event handler to informer")
}

// de-dupe logic for a delete data, if the delete data job started
// parameter is set, it means a delete data job has already been
// started on this
func dupeDeleteData(clientset pgo.Interface, task *crv1.Pgtask, ns string) bool {
	ctx := context.TODO()
	tmp, err := clientset.CrunchydataV1().Pgtasks(ns).Get(ctx, task.Spec.Name, metav1.GetOptions{})
	if err != nil {
		// a big time error if this occurs
		return false
	}

	if tmp.Spec.Parameters[config.LABEL_DELETE_DATA_STARTED] == "" {
		return false
	}

	return true
}

// WorkerCount returns the worker count for the controller
func (c *Controller) WorkerCount() int {
	return c.PgtaskWorkerCount
}
