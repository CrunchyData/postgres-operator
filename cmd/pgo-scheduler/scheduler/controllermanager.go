package scheduler

/*
Copyright 2020 - 2023 Crunchy Data Solutions, Inc.
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
	"errors"
	"fmt"
	"sync"

	"github.com/crunchydata/postgres-operator/internal/controller"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/ns"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ControllerManager manages a map of controller groups, each of which is comprised of the various
// controllers needed to handle events within a specific namespace.  Only one controllerGroup is
// allowed per namespace.
type ControllerManager struct {
	mgrMutex               sync.Mutex
	controllers            map[string]*controllerGroup
	installationName       string
	namespaceOperatingMode ns.NamespaceOperatingMode
	Scheduler              *Scheduler
	sem                    *semaphore.Weighted
}

// controllerGroup is a struct for managing the various controllers created to handle events
// in a specific namespace
type controllerGroup struct {
	stopCh              chan struct{}
	started             bool
	kubeInformerFactory kubeinformers.SharedInformerFactory
	informerSyncedFuncs []cache.InformerSynced
	clientset           kubernetes.Interface
}

// NewControllerManager returns a new ControllerManager comprised of controllerGroups for each
// namespace included in the 'namespaces' parameter.
func NewControllerManager(namespaces []string, scheduler *Scheduler, installationName string,
	namespaceOperatingMode ns.NamespaceOperatingMode) (*ControllerManager, error) {
	controllerManager := ControllerManager{
		controllers:            make(map[string]*controllerGroup),
		installationName:       installationName,
		namespaceOperatingMode: namespaceOperatingMode,
		Scheduler:              scheduler,
		sem:                    semaphore.NewWeighted(1),
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
// group is comprised of a controller for the following resource:
// - configmaps
// One SharedInformerFactory is utilized, specifically for Kube resources, to create and track the
// informers for this resource.  Each controller group also receives its own clients, which can then
// be utilized by the controller within the controller group.
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
	if c.controllers[namespace] != nil {
		// first try to clean if one is not already in progress
		if err := c.clean(namespace); err != nil {
			log.Infof("Controller Manager: %s", err.Error())
		}

		// if we just cleaned the current namespace's controller, then return
		if _, ok := c.controllers[namespace]; !ok {
			log.Infof("Controller Manager: controller group for namespace %s has already "+
				"been cleaned", namespace)
			return nil
		}
	}

	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	// only return an error if not a group already exists error
	if err := c.addControllerGroup(namespace); err != nil &&
		!errors.Is(err, controller.ErrControllerGroupExists) {
		return err
	}

	if err := c.runControllerGroup(namespace); err != nil {
		return err
	}

	return nil
}

// RemoveAll removes all controller groups managed by the controller manager, first stopping all
// controllers within each controller group managed by the controller manager.
func (c *ControllerManager) RemoveAll() {
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	for ns := range c.controllers {
		c.removeControllerGroup(ns)
	}

	log.Debug("Controller Manager: all contollers groups have been removed")
}

// RemoveGroup removes the controller group for the namespace specified, first stopping all
// controllers within that group
func (c *ControllerManager) RemoveGroup(namespace string) {
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	c.removeControllerGroup(namespace)
}

// RunAll runs all controllers across all controller groups managed by the controller manager.
func (c *ControllerManager) RunAll() error {
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	for ns := range c.controllers {
		if err := c.runControllerGroup(ns); err != nil {
			return err
		}
	}

	log.Debug("Controller Manager: all contoller groups are now running")

	return nil
}

// RunGroup runs the controllers within the controller group for the namespace specified.
func (c *ControllerManager) RunGroup(namespace string) error {
	c.mgrMutex.Lock()
	defer c.mgrMutex.Unlock()

	if _, ok := c.controllers[namespace]; !ok {
		log.Debugf("Controller Manager: unable to run controller group for namespace %s because "+
			"a controller group for this namespace does not exist", namespace)
		return nil
	}

	if err := c.runControllerGroup(namespace); err != nil {
		return err
	}

	log.Debugf("Controller Manager: the controller group for ns %s is now running", namespace)

	return nil
}

// addControllerGroup adds a new controller group for the namespace specified
func (c *ControllerManager) addControllerGroup(namespace string) error {
	if _, ok := c.controllers[namespace]; ok {
		log.Debugf("Controller Manager: a controller for namespace %s already exists", namespace)
		return controller.ErrControllerGroupExists
	}

	// create a client for kube resources
	client, err := kubeapi.NewClient()
	if err != nil {
		log.Error(err)
		return err
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(client, 0,
		kubeinformers.WithNamespace(namespace))

	configmapController := &Controller{
		Informer:  kubeInformerFactory.Core().V1().ConfigMaps(),
		Scheduler: c.Scheduler,
	}

	// add the proper event handler to the informer in each controller
	configmapController.AddConfigMapEventHandler()

	group := &controllerGroup{
		clientset:           client,
		stopCh:              make(chan struct{}),
		kubeInformerFactory: kubeInformerFactory,
		informerSyncedFuncs: []cache.InformerSynced{
			kubeInformerFactory.Core().V1().ConfigMaps().Informer().HasSynced,
		},
	}

	c.controllers[namespace] = group

	log.Debugf("Controller Manager: added controller group for namespace %s", namespace)

	return nil
}

// clean removes and controller groups that no longer correspond to a valid namespace within
// the Kubernetes cluster, e.g. in the event that a namespace has been deleted.
func (c *ControllerManager) clean(namespace string) error {
	if !c.sem.TryAcquire(1) {
		return fmt.Errorf("controller group clean already in progress, namespace %s will not "+
			"clean", namespace)
	}
	defer c.sem.Release(1)

	log.Debugf("Controller Manager: namespace %s acquired clean lock and will clean the "+
		"controller groups", namespace)

	nsList, err := ns.GetCurrentNamespaceList(c.controllers[namespace].clientset,
		c.installationName, c.namespaceOperatingMode)
	if err != nil {
		log.Errorf(err.Error())
	}

	for controlledNamespace := range c.controllers {
		cleanNamespace := true
		for _, currNamespace := range nsList {
			if controlledNamespace == currNamespace {
				cleanNamespace = false
				break
			}
		}
		if cleanNamespace {
			log.Debugf("Controller Manager: removing controller group for namespace %s",
				controlledNamespace)
			c.removeControllerGroup(controlledNamespace)
		}
	}

	return nil
}

// hasListerPrivs verifies the Operator has the privileges required to start the controllers
// for the namespace specified.
func (c *ControllerManager) hasListerPrivs(namespace string) bool {
	controllerGroup := c.controllers[namespace]

	var err error
	var hasCorePrivs bool

	hasCorePrivs, err = ns.CheckAccessPrivs(controllerGroup.clientset,
		map[string][]string{"configmaps": {"list"}},
		"", namespace)
	if err != nil {
		log.Errorf(err.Error())
	} else if !hasCorePrivs {
		log.Errorf("Controller Manager: Controller Group for namespace %s does not have the "+
			"required list privileges for resource %s in the Core API",
			namespace, "configmaps")
	}

	return hasCorePrivs
}

// runControllerGroup is responsible running the controllers for the controller group corresponding
// to the namespace provided
func (c *ControllerManager) runControllerGroup(namespace string) error {
	controllerGroup := c.controllers[namespace]

	hasListerPrivs := c.hasListerPrivs(namespace)
	switch {
	case c.controllers[namespace].started && hasListerPrivs:
		log.Debugf("Controller Manager: controller group for namespace %s is already running",
			namespace)
		return nil
	case c.controllers[namespace].started && !hasListerPrivs:
		c.removeControllerGroup(namespace)
		return fmt.Errorf("Controller Manager: removing the running controller group for "+
			"namespace %s because it no longer has the required privs, will attempt to "+
			"restart on the next ns refresh interval", namespace)
	}

	controllerGroup.kubeInformerFactory.Start(controllerGroup.stopCh)

	if ok := cache.WaitForNamedCacheSync(namespace, controllerGroup.stopCh,
		controllerGroup.informerSyncedFuncs...); !ok {
		return fmt.Errorf("Controller Manager: failed to wait for caches to sync")
	}

	controllerGroup.started = true

	log.Debugf("Controller Manager: controller group for namespace %s is now running", namespace)

	return nil
}

// removeControllerGroup removes the controller group for the namespace specified.  Any worker
// queues associated with the controllers inside of the controller group are first shutdown
// prior to removing the controller group.
func (c *ControllerManager) removeControllerGroup(namespace string) {
	if _, ok := c.controllers[namespace]; !ok {
		log.Debugf("Controller Manager: no controller group to remove for ns %s", namespace)
		return
	}

	c.stopControllerGroup(namespace)
	delete(c.controllers, namespace)

	log.Debugf("Controller Manager: the controller group for ns %s has been removed", namespace)
}

// stopControllerGroup stops the controller group associated with the namespace specified.  This is
// done by calling the ShutdownWorker function associated with the controller.  If the controller
// does not have a ShutdownWorker function then no action is taken.
func (c *ControllerManager) stopControllerGroup(namespace string) {
	if _, ok := c.controllers[namespace]; !ok {
		log.Debugf("Controller Manager: unable to stop controller group for namespace %s because "+
			"a controller group for this namespace does not exist", namespace)
		return
	}

	controllerGroup := c.controllers[namespace]

	// close the stop channel to stop all informers and instruct the workers queues to shutdown
	close(controllerGroup.stopCh)

	controllerGroup.started = false

	log.Debugf("Controller Manager: the controller group for ns %s has been stopped", namespace)
}
