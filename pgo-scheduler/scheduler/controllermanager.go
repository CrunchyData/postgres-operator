package scheduler

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
	"sync"

	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"

	kubeinformers "k8s.io/client-go/informers"
)

// ControllerManager manages a map of controller groups, each of which is comprised of the various
// controllers needed to handle events within a specific namespace.  Only one controllerGroup is
// allowed per namespace.
type ControllerManager struct {
	context     context.Context
	cancelFunc  context.CancelFunc
	mgrMutex    sync.Mutex
	controllers map[string]*controllerGroup
	Scheduler   *Scheduler
}

// controllerGroup is a struct for managing the various controllers created to handle events
// in a specific namespace
type controllerGroup struct {
	context             context.Context
	cancelFunc          context.CancelFunc
	instanceMutex       sync.Mutex
	started             bool
	kubeInformerFactory kubeinformers.SharedInformerFactory
}

// NewControllerManager returns a new ControllerManager comprised of controllerGroups for each
// namespace included in the 'namespaces' parameter.
func NewControllerManager(namespaces []string, scheduler *Scheduler) (*ControllerManager, error) {

	ctx, cancelFunc := context.WithCancel(context.Background())

	controllerManager := ControllerManager{
		context:     ctx,
		cancelFunc:  cancelFunc,
		controllers: make(map[string]*controllerGroup),
		Scheduler:   scheduler,
	}

	// create controller groups for each namespace provided
	for _, ns := range namespaces {
		if err := controllerManager.AddControllerGroup(ns); err != nil {
			log.Error(err)
			return nil, err
		}
	}

	return &controllerManager, nil
}

// AddControllerGroup adds a new controller group for the namespace specified.  Each controller
// group is comprised of controllers for the following resources:
// - configmaps
// Two SharedInformerFactory's are utilized (one for Kube resources and one for PosgreSQL Operator
// resources) to create and track the informers for each type of resource, while any controllers
// utilizing worker queues are also tracked (this allows all informers and worker queues to be
// easily started as needed). Each controller group also recieves its own clients, which can then
// be utilized by the various controllers within that controller group.
func (c *ControllerManager) AddControllerGroup(namespace string) error {

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()
	if _, ok := c.controllers[namespace]; ok {
		return nil
	}

	// create a client for kube resources
	clients, err := kubeapi.NewControllerClients()
	if err != nil {
		log.Error(err)
		return err
	}

	kubeClientset := clients.Kubeclientset

	ctx, cancelFunc := context.WithCancel(c.context)

	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClientset, 0,
		kubeinformers.WithNamespace(namespace))

	configmapController := &Controller{
		ConfigmapClientset: kubeClientset,
		Informer:           kubeInformerFactory.Core().V1().ConfigMaps(),
	}

	// add the proper event handler to the informer in each controller
	configmapController.AddConfigMapEventHandler()

	group := &controllerGroup{
		context:             ctx,
		cancelFunc:          cancelFunc,
		kubeInformerFactory: kubeInformerFactory,
	}

	c.controllers[namespace] = group

	return nil
}

// AddAndRunControllerGroup is a convenience function that adds a controller group for the
// namespace specified, and then immediately runs the controllers in that group.
func (c *ControllerManager) AddAndRunControllerGroup(namespace string) {
	c.AddControllerGroup(namespace)
	c.RunGroup(namespace)
}

// RunAll runs all controllers across all controller groups managed by the controller manager.
func (c *ControllerManager) RunAll() {
	for ns := range c.controllers {
		c.RunGroup(ns)
	}
}

// RunGroup runs the controllers within the controller group for the namespace specified.
func (c *ControllerManager) RunGroup(namespace string) {

	instance := c.controllers[namespace]

	instance.instanceMutex.Lock()
	defer instance.instanceMutex.Unlock()

	if instance.started {
		return
	}

	instance.kubeInformerFactory.Start(instance.context.Done())
}

// StopAll stops all controllers across all controller groups managed by the controller manager.
func (c *ControllerManager) StopAll() {
	c.cancelFunc()
}

// StopGroup stops the controllers within the controller group for the namespace specified.
func (c *ControllerManager) StopGroup(namespace string) {
	c.controllers[namespace].cancelFunc()
}

// RemoveAll removes all controller groups managed by the controller manager, first stopping all
// controllers within each controller group managed by the controller manager.
func (c *ControllerManager) RemoveAll() {
	c.StopAll()
	c.controllers = make(map[string]*controllerGroup)
}

// RemoveGroup removes the controller group for the namespace specified, first stopping all
// controllers within that group
func (c *ControllerManager) RemoveGroup(namespace string) {
	c.StopGroup(namespace)
	delete(c.controllers, namespace)
}
