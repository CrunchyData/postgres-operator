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

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
	backupoperator "github.com/crunchydata/postgres-operator/operator/backup"
)

// PgbackupController holds connections required by the controller
type PgbackupController struct {
	PgbackupClient    *rest.RESTClient
	PgbackupScheme    *runtime.Scheme
	PgbackupClientset *kubernetes.Clientset
	Ctx               context.Context
}

// Run starts controller
func (c *PgbackupController) Run() error {
	log.Debugf("Watch Pgbackup objects")

	err := c.watchPgbackups(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for Pgbackup resource: %v", err)
		return err
	}

	<-c.Ctx.Done()
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

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use backupScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj := backup.DeepCopyObject()

	backupCopy := copyObj.(*crv1.Pgbackup)

	state := crv1.PgbackupStateProcessed
	message := "Successfully processed Pgbackup by controller"
	err := kubeapi.PatchpgbackupStatus(c.PgbackupClient, state, message, backupCopy, backup.ObjectMeta.Namespace)
	if err != nil {
		log.Errorf("ERROR updating pgbackup status: %s", err.Error())
	}

	//handle new pgbackups
	backupoperator.AddBackupBase(c.PgbackupClientset, c.PgbackupClient, backupCopy, backup.ObjectMeta.Namespace)
}

// onUpdate is called when a pgbackup is updated
func (c *PgbackupController) onUpdate(oldObj, newObj interface{}) {
	oldBackup := oldObj.(*crv1.Pgbackup)
	backup := newObj.(*crv1.Pgbackup)
	log.Debugf("[PgbackupController] ns=%s onUpdate %s", backup.ObjectMeta.Namespace, backup.ObjectMeta.SelfLink)

	if oldBackup.Spec.BackupStatus != crv1.PgBackupJobReSubmitted &&
		backup.Spec.BackupStatus == crv1.PgBackupJobReSubmitted {
		log.Debugf("[PgbackupController] ns=%s onUpdate %s re-submitted", backup.ObjectMeta.Namespace, backup.ObjectMeta.SelfLink)
		backupoperator.AddBackupBase(c.PgbackupClientset, c.PgbackupClient, backup, backup.ObjectMeta.Namespace)
	}
}

// onDelete is called when a pgbackup is deleted
func (c *PgbackupController) onDelete(obj interface{}) {
	backup := obj.(*crv1.Pgbackup)
	log.Debugf("[PgbackupController] ns=%s onDelete %s", backup.ObjectMeta.Namespace, backup.ObjectMeta.SelfLink)
}

func (c *PgbackupController) SetupWatch(ns string) {
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
}
