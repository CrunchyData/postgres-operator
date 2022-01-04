package pgpolicy

/*
Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	informers "github.com/crunchydata/postgres-operator/pkg/generated/informers/externalversions/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"
)

// Controller holds connections for the controller
type Controller struct {
	Clientset kubeapi.Interface
	Informer  informers.PgpolicyInformer
}

// onAdd is called when a pgpolicy is added
func (c *Controller) onAdd(obj interface{}) {
	policy := obj.(*crv1.Pgpolicy)
	log.Debugf("[pgpolicy Controller] onAdd ns=%s %s", policy.ObjectMeta.Namespace, policy.ObjectMeta.SelfLink)

	//handle the case of when a pgpolicy is already processed, which
	//is the case when the operator restarts
	if policy.Status.State == crv1.PgpolicyStateProcessed {
		log.Debug("pgpolicy " + policy.ObjectMeta.Name + " already processed")
		return
	}

	patch, err := json.Marshal(map[string]interface{}{
		"status": crv1.PgpolicyStatus{
			State:   crv1.PgpolicyStateProcessed,
			Message: "Successfully processed Pgpolicy by controller",
		},
	})
	if err == nil {
		_, err = c.Clientset.CrunchydataV1().Pgpolicies(policy.Namespace).Patch(policy.Name, types.MergePatchType, patch)
	}
	if err != nil {
		log.Errorf("ERROR updating pgpolicy status: %s", err.Error())
	}

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPolicy

	f := events.EventCreatePolicyFormat{
		EventHeader: events.EventHeader{
			Namespace: policy.ObjectMeta.Namespace,
			Username:  policy.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventCreatePolicy,
		},
		Policyname: policy.ObjectMeta.Name,
	}

	err = events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}

// onUpdate is called when a pgpolicy is updated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {
}

// onDelete is called when a pgpolicy is deleted
func (c *Controller) onDelete(obj interface{}) {
	policy := obj.(*crv1.Pgpolicy)
	log.Debugf("[pgpolicy Controller] onDelete ns=%s %s", policy.ObjectMeta.Namespace, policy.ObjectMeta.SelfLink)

	log.Debugf("DELETED pgpolicy %s", policy.ObjectMeta.Name)

	//publish event
	topics := make([]string, 1)
	topics[0] = events.EventTopicPolicy

	f := events.EventDeletePolicyFormat{
		EventHeader: events.EventHeader{
			Namespace: policy.ObjectMeta.Namespace,
			Username:  policy.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventDeletePolicy,
		},
		Policyname: policy.ObjectMeta.Name,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}

// AddPGPolicyEventHandler adds the pgpolicy event handler to the pgpolicy informer
func (c *Controller) AddPGPolicyEventHandler() {

	c.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	log.Debugf("pgpolicy Controller: added event handler to informer")
}
