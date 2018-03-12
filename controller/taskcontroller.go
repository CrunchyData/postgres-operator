package controller

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	//apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	taskoperator "github.com/crunchydata/postgres-operator/operator/task"
)

// PgtaskController holds connections for the controller
type PgtaskController struct {
	PgtaskClient    *rest.RESTClient
	PgtaskScheme    *runtime.Scheme
	PgtaskClientset *kubernetes.Clientset
	Namespace       string
}

// Run starts an pgtask resource controller
func (c *PgtaskController) Run(ctx context.Context) error {
	log.Info("Watch Pgtask objects")

	// Watch Example objects
	_, err := c.watchPgtasks(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgtask resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchPgtasks watches the pgtask resource catching events
func (c *PgtaskController) watchPgtasks(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.PgtaskClient,
		crv1.PgtaskResourcePlural,
		//apiv1.NamespaceAll,
		c.Namespace,
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

	go controller.Run(ctx.Done())
	return controller, nil
}

// onAdd is called when a pgtask is added
func (c *PgtaskController) onAdd(obj interface{}) {
	task := obj.(*crv1.Pgtask)
	log.Errorf("[PgtaskCONTROLLER] OnAdd %s\n", task.ObjectMeta.SelfLink)
	if task.Status.State == crv1.PgtaskStateProcessed {
		log.Info("pgtask " + task.ObjectMeta.Name + " already processed")
		return
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use taskScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj, err := c.PgtaskScheme.Copy(task)
	if err != nil {
		log.Errorf("ERROR creating a deep copy of task object: %v\n", err)
		return
	}

	//update the status of the task as completed
	taskCopy := copyObj.(*crv1.Pgtask)
	taskCopy.Status = crv1.PgtaskStatus{
		State:   crv1.PgtaskStateProcessed,
		Message: "Successfully processed Pgtask by controller",
	}

	err = c.PgtaskClient.Put().
		Name(task.ObjectMeta.Name).
		Namespace(task.ObjectMeta.Namespace).
		Resource(crv1.PgtaskResourcePlural).
		Body(taskCopy).
		Do().
		Error()

	if err != nil {
		log.Errorf("ERROR updating status: %v\n", err)
	} else {
		log.Errorf("UPDATED status: %#v\n", taskCopy)
	}

	//process the incoming task
	switch task.Spec.TaskType {
	case crv1.PgtaskDeleteData:
		log.Info("delete data task added")
		log.Info("pvc is " + task.Spec.Parameters)
		log.Info("dbname is " + task.Spec.Name)
		taskoperator.RemoveData(task.ObjectMeta.Namespace, c.PgtaskClientset, task)
	default:
		log.Info("unknown task type on pgtask added")
	}

	//for now, remove the pgtask in all cases
	err = c.PgtaskClient.Delete().
		Name(task.ObjectMeta.Name).
		Namespace(task.ObjectMeta.Namespace).
		Resource(crv1.PgtaskResourcePlural).
		Body(taskCopy).
		Do().
		Error()

	if err != nil {
		log.Errorf("ERROR deleting pgtask status: %s %v\n", task.ObjectMeta.Name, err)
	} else {
		log.Errorf("UPDATED deleted pgtask %s\n", task.ObjectMeta.Name)
	}

}

// onUpdate is called when a pgtask is updated
func (c *PgtaskController) onUpdate(oldObj, newObj interface{}) {
}

// onDelete is called when a pgtask is deleted
func (c *PgtaskController) onDelete(obj interface{}) {
}
