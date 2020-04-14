package manager

/*
Copyright 2020 Crunchy Data Solutions, Inc.
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
	"errors"
	"sync"
	"time"

	"github.com/crunchydata/postgres-operator/controller"
	"github.com/crunchydata/postgres-operator/controller/job"
	"github.com/crunchydata/postgres-operator/controller/pgcluster"
	"github.com/crunchydata/postgres-operator/controller/pgpolicy"
	"github.com/crunchydata/postgres-operator/controller/pgreplica"
	"github.com/crunchydata/postgres-operator/controller/pgtask"
	"github.com/crunchydata/postgres-operator/controller/pod"
	"github.com/crunchydata/postgres-operator/kubeapi"
	informers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/util/workqueue"
)

// ControllerManager manages a map of controller groups, each of which is comprised of the various
// controllers needed to handle events within a specific namespace.  Only one controllerGroup is
// allowed per namespace.
type ControllerManager struct {
	context     context.Context
	cancelFunc  context.CancelFunc
	mgrMutex    sync.Mutex
	controllers map[string]*controllerGroup
}

// controllerGroup is a struct for managing the various controllers created to handle events
// in a specific namespace
type controllerGroup struct {
	context                context.Context
	cancelFunc             context.CancelFunc
	started                bool
	pgoInformerFactory     informers.SharedInformerFactory
	kubeInformerFactory    kubeinformers.SharedInformerFactory
	controllersWithWorkers []controller.WorkerRunner
}

// NewControllerManager returns a new ControllerManager comprised of controllerGroups for each
// namespace included in the 'namespaces' parameter.
func NewControllerManager(namespaces []string) (*ControllerManager, error) {

	ctx, cancelFunc := context.WithCancel(context.Background())

	controllerManager := ControllerManager{
		context:     ctx,
		cancelFunc:  cancelFunc,
		controllers: make(map[string]*controllerGroup),
	}

	// create controller groups for each namespace provided
	for _, ns := range namespaces {
		if err := controllerManager.AddGroup(ns); err != nil {
			log.Error(err)
			return nil, err
		}
	}

	log.Debugf("Controller Manager: new controller manager created for namespaces %v",
		namespaces)

	return &controllerManager, nil
}

// AddGroup adds a new controller group for the namespace specified.  Each controller
// group is comprised of controllers for the following resources:
// - pods
// - jobs
// - pgclusters
// - pgpolicys
// - pgtasks
// Two SharedInformerFactory's are utilized (one for Kube resources and one for PosgreSQL Operator
// resources) to create and track the informers for each type of resource, while any controllers
// utilizing worker queues are also tracked (this allows all informers and worker queues to be
// easily started as needed). Each controller group also receives its own clients, which can then
// be utilized by the various controllers within that controller group.
func (c *ControllerManager) AddGroup(namespace string) error {

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	// only return an error if not a group already exists error
	if err := c.addControllerGroup(namespace); err != nil &&
		!errors.Is(err, controller.ErrControllerGroupExists) {
		return err
	}

	return nil
}

// AddAndRunGroup is a convenience function that adds a controller group for the
// namespace specified, and then immediately runs the controllers in that group.
func (c *ControllerManager) AddAndRunGroup(namespace string) error {

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	// only return an error if not a group already exists error
	if err := c.addControllerGroup(namespace); err != nil &&
		!errors.Is(err, controller.ErrControllerGroupExists) {
		return err
	}

	c.runControllerGroup(namespace)

	return nil
}

// RemoveAll removes all controller groups managed by the controller manager, first stopping all
// controllers within each controller group managed by the controller manager.
func (c *ControllerManager) RemoveAll() {

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	c.controllers = make(map[string]*controllerGroup)
	log.Debug("Controller Manager: all contollers groups have been removed")
}

// RemoveGroup removes the controller group for the namespace specified, first stopping all
// controllers within that group
func (c *ControllerManager) RemoveGroup(namespace string) {

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	if _, ok := c.controllers[namespace]; !ok {
		log.Debugf("Controller Manager: no controller group to remove for ns %s ", namespace)
		return
	}

	delete(c.controllers, namespace)
	log.Debugf("Controller Manager: the controller group for ns %s has been removed", namespace)
}

// RunAll runs all controllers across all controller groups managed by the controller manager.
func (c *ControllerManager) RunAll() {

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	for ns := range c.controllers {
		c.runControllerGroup(ns)
	}

	log.Debug("Controller Manager: all contoller groups are now running")
}

// RunGroup runs the controllers within the controller group for the namespace specified.
func (c *ControllerManager) RunGroup(namespace string) {

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	if _, ok := c.controllers[namespace]; !ok {
		log.Debugf("Controller Manager: unable to run controller group for namespace %s because "+
			"a controller group for this namespace does not exist", namespace)
		return
	}

	c.runControllerGroup(namespace)

	log.Debugf("Controller Manager: the controller group for ns %s is now running", namespace)
}

// StopAll stops all controllers across all controller groups managed by the controller manager.
func (c *ControllerManager) StopAll() {

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	c.cancelFunc()
	log.Debug("Controller Manager: all contoller groups are now stopped")
}

// StopGroup stops the controllers within the controller group for the namespace specified.
func (c *ControllerManager) StopGroup(namespace string) {

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	if _, ok := c.controllers[namespace]; !ok {
		log.Debugf("Controller Manager: unable to stop controller group for namespace %s because "+
			"a controller group for this namespace does not exist", namespace)
		return
	}

	controllerGroup := c.controllers[namespace]
	controllerGroup.cancelFunc()
	controllerGroup.started = false

	log.Debugf("Controller Manager: the controller group for ns %s has been stopped", namespace)
}

// addControllerGroup adds a new controller group for the namespace specified
func (c *ControllerManager) addControllerGroup(namespace string) error {

	if _, ok := c.controllers[namespace]; ok {
		log.Debugf("Controller Manager: a controller for namespace %s already exists", namespace)
		return controller.ErrControllerGroupExists
	}

	// create a client for kube resources
	clients, err := kubeapi.NewControllerClients()
	if err != nil {
		log.Error(err)
		return err
	}

	config := clients.Config
	pgoClientset := clients.PGOClientset
	pgoRESTClient := clients.PGORestclient
	kubeClientset := clients.Kubeclientset

	ctx, cancelFunc := context.WithCancel(c.context)

	pgoInformerFactory := informers.NewSharedInformerFactoryWithOptions(pgoClientset, 0,
		informers.WithNamespace(namespace))

	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClientset, 0,
		kubeinformers.WithNamespace(namespace))

	pgTaskcontroller := &pgtask.Controller{
		PgtaskConfig:    config,
		PgtaskClient:    pgoRESTClient,
		PgtaskClientset: kubeClientset,
		Queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Informer:        pgoInformerFactory.Crunchydata().V1().Pgtasks(),
	}

	pgClustercontroller := &pgcluster.Controller{
		PgclusterClient:    pgoRESTClient,
		PgclusterClientset: kubeClientset,
		PgclusterConfig:    config,
		Queue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Informer:           pgoInformerFactory.Crunchydata().V1().Pgclusters(),
	}

	pgReplicacontroller := &pgreplica.Controller{
		PgreplicaClient:    pgoRESTClient,
		PgreplicaClientset: kubeClientset,
		Queue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Informer:           pgoInformerFactory.Crunchydata().V1().Pgreplicas(),
	}

	pgPolicycontroller := &pgpolicy.Controller{
		PgpolicyClient:    pgoRESTClient,
		PgpolicyClientset: kubeClientset,
		Informer:          pgoInformerFactory.Crunchydata().V1().Pgpolicies(),
	}

	podcontroller := &pod.Controller{
		PodConfig:    config,
		PodClientset: kubeClientset,
		PodClient:    pgoRESTClient,
		Informer:     kubeInformerFactory.Core().V1().Pods(),
	}

	jobcontroller := &job.Controller{
		JobConfig:    config,
		JobClientset: kubeClientset,
		JobClient:    pgoRESTClient,
		Informer:     kubeInformerFactory.Batch().V1().Jobs(),
	}

	// add the proper event handler to the informer in each controller
	pgTaskcontroller.AddPGTaskEventHandler()
	pgClustercontroller.AddPGClusterEventHandler()
	pgReplicacontroller.AddPGReplicaEventHandler()
	pgPolicycontroller.AddPGPolicyEventHandler()
	podcontroller.AddPodEventHandler()
	jobcontroller.AddJobEventHandler()

	group := &controllerGroup{
		context:             ctx,
		cancelFunc:          cancelFunc,
		pgoInformerFactory:  pgoInformerFactory,
		kubeInformerFactory: kubeInformerFactory,
	}

	// store the controllers containing worker queues so that the queues can also be started
	// when any informers in the controller are started
	group.controllersWithWorkers = append(group.controllersWithWorkers,
		pgTaskcontroller, pgClustercontroller, pgReplicacontroller)

	c.controllers[namespace] = group

	log.Debugf("Controller Manager: added controller group for namespace %s", namespace)

	return nil
}

// runControllerGroup is responsible running the controllers for the controller group corresponding
// to the namespace provided
func (c *ControllerManager) runControllerGroup(namespace string) {

	controllerGroup := c.controllers[namespace]

	if c.controllers[namespace].started {
		log.Debugf("Controller Manager: controller group for namespace %s is already running",
			namespace)
		return
	}

	controllerGroup.kubeInformerFactory.Start(controllerGroup.context.Done())
	controllerGroup.pgoInformerFactory.Start(controllerGroup.context.Done())

	for _, worker := range c.controllers[namespace].controllersWithWorkers {
		go wait.Until(worker.RunWorker, time.Second, controllerGroup.context.Done())
	}

	controllerGroup.started = true

	log.Debugf("Controller Manager: controller group for namespace %s is now running", namespace)
}
