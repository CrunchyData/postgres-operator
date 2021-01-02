package scheduler

/*
Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Controller holds the client and informer for the controller, along with a pointer to a
// Scheduler.
type Controller struct {
	Informer  coreinformers.ConfigMapInformer
	Scheduler *Scheduler
}

// onAdd is called when a configMap is added
func (c *Controller) onAdd(obj interface{}) {
	cm, ok := obj.(*v1.ConfigMap)
	if !ok {
		log.WithFields(log.Fields{}).Error("Could not convert runtime object to configmap..")
	}

	if _, ok := cm.Labels["crunchy-scheduler"]; !ok {
		return
	}

	if err := c.Scheduler.AddSchedule(cm); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to add schedules")
	}
}

// onDelete is called when a configMap is deleted
func (c *Controller) onDelete(obj interface{}) {
	cm, ok := obj.(*v1.ConfigMap)
	if !ok {
		log.WithFields(log.Fields{}).Error("Could not convert runtime object to configmap..")
	}

	if _, ok := cm.Labels["crunchy-scheduler"]; !ok {
		return
	}
	c.Scheduler.DeleteSchedule(cm)
}

// AddConfigMapEventHandler adds the pgcluster event handler to the pgcluster informer
func (c *Controller) AddConfigMapEventHandler() {
	c.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		DeleteFunc: c.onDelete,
	})

	log.Debugf("ConfigMap Controller: added event handler to informer")
}
