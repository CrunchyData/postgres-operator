package cluster

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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	weakrand "math/rand"
	"os"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/internal/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/events"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/pgadmin"
	"github.com/crunchydata/postgres-operator/internal/util"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	defPgAdminPort   = config.DEFAULT_PGADMIN_PORT
	defSetupUsername = "pgadminsetup"
)

type pgAdminTemplateFields struct {
	Name           string
	ClusterName    string
	CCPImagePrefix string
	CCPImageTag    string
	Port           string
	ServicePort    string
	InitUser       string
	InitPass       string
	PVCName        string
}

// pgAdminDeploymentFormat is the name of the Kubernetes Deployment that
// manages pgAdmin, and follows the format "<clusterName>-pgadmin"
const pgAdminDeploymentFormat = "%s-pgadmin"

// initPassLen is the length of the one-time setup password for pgadmin
const initPassLen = 20

const (
	deployTimeout = 60
	pollInterval  = 3
)

// AddPgAdmin contains the various functions that are used to add a pgAdmin
// Deployment to a PostgreSQL cluster
//
// Any returned error is logged in the calling function
func AddPgAdmin(
	clientset *kubernetes.Clientset,
	restclient *rest.RESTClient,
	restconfig *rest.Config,
	cluster *crv1.Pgcluster,
	storageClass *crv1.PgStorageSpec) error {
	log.Debugf("adding pgAdmin")

	// first, ensure that the Cluster CR is updated to know that there is now
	// a pgAdmin associated with it. This may also include other CR updates too,
	// such as if the pgAdmin is being added via a pgtask, and as such the
	// values for memory/CPU may be set as well.
	//
	// if we cannot update this we abort
	cluster.Labels[config.LABEL_PGADMIN] = "true"

	ns := cluster.Namespace

	if err := kubeapi.Updatepgcluster(restclient, cluster, cluster.Name, ns); err != nil {
		return err
	}

	// Using deployment/service name for PVC also
	pvcName := fmt.Sprintf(pgAdminDeploymentFormat, cluster.Name)

	// create the pgAdmin storage volume
	if _, err := pvc.CreateIfNotExists(clientset, *storageClass, pvcName, cluster.Name, ns); err != nil {
		log.Errorf("Error creating PVC: %s", err.Error())
		return err
	} else {
		log.Info("created pgadmin PVC =" + pvcName + " in namespace " + ns)
	}

	// create the pgAdmin deployment
	if err := createPgAdminDeployment(clientset, cluster, pvcName); err != nil {
		return err
	}

	// create the pgAdmin service
	if err := createPgAdminService(clientset, cluster); err != nil {
		return err
	}

	log.Debugf("added pgAdmin to cluster [%s]", cluster.Name)

	return nil
}

// AddPgAdminFromPgTask is a method that helps to bring up
// the pgAdmin deployment that sits alongside a PostgreSQL cluster
func AddPgAdminFromPgTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, task *crv1.Pgtask) {
	clusterName := task.Spec.Parameters[config.LABEL_PGADMIN_TASK_CLUSTER]
	namespace := task.Spec.Namespace
	storage := task.Spec.StorageSpec

	log.Debugf("add pgAdmin from task called for cluster [%s] in namespace [%s]",
		clusterName, namespace)

	// first, check to ensure that the cluster still exosts
	cluster := crv1.Pgcluster{}

	if found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace); !found || err != nil {
		// even if it's not found, this is pretty bad and we cannot continue
		log.Error(err)
		return
	}

	// bring up the pgAdmin deployment
	if err := AddPgAdmin(clientset, restclient, restconfig, &cluster, &storage); err != nil {
		log.Error(err)
		return
	}

	// publish an event
	publishPgAdminEvent(events.EventCreatePgAdmin, task)

	// at this point, the pgtask is successful, so we can safely rvemove it
	// we can fallthrough in the event of an error, because we're returning anyway
	if err := kubeapi.Deletepgtask(restclient, task.Name, namespace); err != nil {
		log.Error(err)
	}

	deployName := fmt.Sprintf(pgAdminDeploymentFormat, clusterName)
	if err := waitForDeploymentReady(clientset, namespace, deployName, deployTimeout, pollInterval); err != nil {
		log.Error(err)
	}

	// Lock down setup user and prepopulate connections for managed users
	if err := BootstrapPgAdminUsers(clientset, restclient, restconfig, &cluster); err != nil {
		log.Error(err)
	}

	return
}

func BootstrapPgAdminUsers(
	clientset *kubernetes.Clientset,
	restclient *rest.RESTClient,
	restconfig *rest.Config,
	cluster *crv1.Pgcluster) error {

	qr, err := pgadmin.GetPgAdminQueryRunner(clientset, restconfig, cluster)
	if err != nil {
		return err
	} else if qr == nil {
		// Cluster doesn't claim to have pgAdmin setup, we're done here
		return nil
	}

	// Disables setup user and breaks the password hash value
	err = qr.Exec("UPDATE user SET active = 0, password = substr(password,1,50) WHERE id=1;")
	if err != nil {
		log.Errorf("failed to lock down pgadmin db [%v], deleting instance", err)
		return err
	}

	// Get service details and prep connection metadata
	service, svcFound, err := kubeapi.GetService(clientset, cluster.Name, cluster.Namespace)
	if err != nil {
		return err
	}

	dbService := pgadmin.ServerEntry{}
	if svcFound {
		dbService = pgadmin.ServerEntryFromPgService(service, cluster.Name)
	}

	// Get current users of cluster and add them to pgadmin's db if they
	// have kubernetes-stored passwords, using the connection info above
	//

	// Get the secrets managed by Kubernetes - any users existing only in
	// Postgres don't have their passwords available
	sel := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, cluster.Name)
	secretList, err := kubeapi.GetSecrets(clientset, sel, cluster.Namespace)
	if err != nil {
		return err
	}
	for _, secret := range secretList.Items {
		dbService.Password = ""

		uname, ok := secret.Data["username"]
		if !ok {
			continue
		}
		user := string(uname[:])
		if secret.Name != fmt.Sprintf("%s-%s-secret", cluster.Name, user) {
			// Doesn't look like the secrets we seek
			continue
		}
		if util.IsPostgreSQLUserSystemAccount(user) {
			continue
		}
		rawpass, ok := secret.Data["password"]
		if !ok {
			// password not stored in secret, can't use this one
			continue
		}

		dbService.Password = string(rawpass[:])
		err = pgadmin.SetLoginPassword(qr, user, dbService.Password)
		if err != nil {
			return err
		}

		if dbService.Name != "" {
			err = pgadmin.SetClusterConnection(qr, user, dbService)
			if err != nil {
				return err
			}
		}
	}
	//
	// Initial autobinding complete

	return nil
}

// DeletePgAdmin contains the various functions that are used to delete a
// pgAdmin Deployment for a PostgreSQL cluster
//
// Any errors that are returned should be logged in the calling function, though
// some logging occurs in this function as well
func DeletePgAdmin(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	clusterName := cluster.Name
	namespace := cluster.Namespace

	log.Debugf("delete pgAdmin from cluster [%s] in namespace [%s]", clusterName, namespace)

	// first, ensure that the Cluster CR is updated to know that there is no
	// longer a pgAdmin associated with it
	// if we cannot update this we abort
	cluster.Labels[config.LABEL_PGADMIN] = "false"

	if err := kubeapi.Updatepgcluster(restclient, cluster, clusterName, namespace); err != nil {
		return err
	}

	// delete the various Kubernetes objects associated with the pgAdmin
	// these include the Service, Deployment, and the pgAdmin data PVC
	// If these fail, we'll just pass through
	//
	// Delete the PVC, Service and Deployment, which share the same naem
	pgAdminDeploymentName := fmt.Sprintf(pgAdminDeploymentFormat, clusterName)

	if err := kubeapi.DeletePVC(clientset, pgAdminDeploymentName, namespace); err != nil {
		log.Warn(err)
	}

	if err := kubeapi.DeleteService(clientset, pgAdminDeploymentName, namespace); err != nil {
		log.Warn(err)
	}

	if err := kubeapi.DeleteDeployment(clientset, pgAdminDeploymentName, namespace); err != nil {
		log.Warn(err)
	}

	return nil
}

// DeletePgAdminFromPgTask is effectively a legacy method that helps to delete
// the pgAdmin deployment that sits alongside a PostgreSQL cluster
func DeletePgAdminFromPgTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, task *crv1.Pgtask) {
	clusterName := task.Spec.Parameters[config.LABEL_PGADMIN_TASK_CLUSTER]
	namespace := task.Spec.Namespace

	log.Debugf("delete pgAdmin from task called for cluster [%s] in namespace [%s]",
		clusterName, namespace)

	// find the pgcluster that is associated with this task
	cluster := crv1.Pgcluster{}

	if found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace); !found || err != nil {
		// if even if it's found and there is an error, it's pretty bad so abort
		log.Error(err)
		return
	}

	// attempt to delete the pgAdmin!
	if err := DeletePgAdmin(clientset, restclient, restconfig, &cluster); err != nil {
		log.Error(err)
		return
	}

	// publish an event
	publishPgAdminEvent(events.EventDeletePgAdmin, task)

	// lastly, remove the task
	if err := kubeapi.Deletepgtask(restclient, task.Name, namespace); err != nil {
		log.Warn(err)
	}
}

// createPgAdminDeployment creates the Kubernetes Deployment for pgAdmin
func createPgAdminDeployment(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster, pvcName string) error {
	log.Debugf("creating pgAdmin deployment: %s", cluster.Name)

	// derive the name of the Deployment...which is also used as the name of the
	// service
	pgAdminDeploymentName := fmt.Sprintf(pgAdminDeploymentFormat, cluster.Name)

	// Password provided to initialize pgadmin setup (admin) - credentials
	// not given to users (account gets disabled)
	//
	// This password is throwaway so low entropy genreation method is fine
	randBytes := make([]byte, initPassLen)
	// weakrand Read is always nil error
	weakrand.Read(randBytes)
	throwawayPass := base64.RawStdEncoding.EncodeToString(randBytes)

	// get the fields that will be substituted in the pgAdmin template
	fields := pgAdminTemplateFields{
		Name:           pgAdminDeploymentName,
		ClusterName:    cluster.Name,
		CCPImagePrefix: operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:    cluster.Spec.CCPImageTag,
		Port:           defPgAdminPort,
		InitUser:       defSetupUsername,
		InitPass:       throwawayPass,
		PVCName:        pvcName,
	}

	// For debugging purposes, put the template substitution in stdout
	if operator.CRUNCHY_DEBUG {
		config.PgAdminTemplate.Execute(os.Stdout, fields)
	}

	// perform the actual template substitution
	doc := bytes.Buffer{}

	if err := config.PgAdminTemplate.Execute(&doc, fields); err != nil {
		return err
	}

	// Set up the Kubernetes deployment for pgAdmin
	deployment := appsv1.Deployment{}

	if err := json.Unmarshal(doc.Bytes(), &deployment); err != nil {
		return err
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_CRUNCHY_PGADMIN,
		&deployment.Spec.Template.Spec.Containers[0])

	if err := kubeapi.CreateDeployment(clientset, &deployment, cluster.Namespace); err != nil {
		return err
	}

	return nil
}

// createPgAdminService creates the Kubernetes Service for pgAdmin
func createPgAdminService(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster) error {
	// pgAdminServiceName is the name of the Service of the pgAdmin, which
	// matches that for the Deploymnt
	pgAdminSvcName := fmt.Sprintf(pgAdminDeploymentFormat, cluster.Name)

	// get the fields that will be substituted in the pgAdmin template
	fields := pgAdminTemplateFields{
		Name:        pgAdminSvcName,
		ClusterName: cluster.Name,
		Port:        defPgAdminPort,
	}

	// For debugging purposes, put the template substitution in stdout
	if operator.CRUNCHY_DEBUG {
		config.PgAdminServiceTemplate.Execute(os.Stdout, fields)
	}

	// perform the actual template substitution
	doc := bytes.Buffer{}

	if err := config.PgAdminServiceTemplate.Execute(&doc, fields); err != nil {
		return err
	}

	// Set up the Kubernetes service for pgAdmin
	service := v1.Service{}

	if err := json.Unmarshal(doc.Bytes(), &service); err != nil {
		return err
	}

	if _, err := kubeapi.CreateService(clientset, &service, cluster.Namespace); err != nil {
		return err
	}

	return nil
}

// publishPgAdminEvent publishes one of the events on the event stream
func publishPgAdminEvent(eventType string, task *crv1.Pgtask) {
	var event events.EventInterface

	// prepare the topics to publish to
	topics := []string{events.EventTopicPgAdmin}
	// set up the event header
	eventHeader := events.EventHeader{
		Namespace: task.Spec.Namespace,
		Username:  task.ObjectMeta.Labels[config.LABEL_PGOUSER],
		Topic:     topics,
		Timestamp: time.Now(),
		EventType: eventType,
	}
	clusterName := task.Spec.Parameters[config.LABEL_PGADMIN_TASK_CLUSTER]

	// now determine which event format to use!
	switch eventType {
	case events.EventCreatePgAdmin:
		event = events.EventCreatePgAdminFormat{
			EventHeader: eventHeader,
			Clustername: clusterName,
		}
	case events.EventDeletePgAdmin:
		event = events.EventDeletePgAdminFormat{
			EventHeader: eventHeader,
			Clustername: clusterName,
		}
	}

	// publish the event; if there is an error, log it, but we don't care
	if err := events.Publish(event); err != nil {
		log.Error(err.Error())
	}
}
