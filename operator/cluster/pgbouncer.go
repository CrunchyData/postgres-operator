package cluster

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
	"bytes"
	"encoding/json"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"time"
)

type PgbouncerPasswdFields struct {
	Username string
	Password string
}

type PgbouncerHBAFields struct {
}

type PgbouncerConfFields struct {
	PG_PRIMARY_SERVICE_NAME string
	PG_REPLICA_SERVICE_NAME string
	PG_USERNAME             string
	PG_PASSWORD             string
	PG_PORT                 string
	PG_DATABASE             string
}

type PgbouncerTemplateFields struct {
	Name               string
	ClusterName        string
	SecretsName        string
	CCPImagePrefix     string
	CCPImageTag        string
	Port               string
	PrimaryServiceName string
	ReplicaServiceName string
	ContainerResources string
	PgBouncerUser      string
	PgBouncerPass      string
}

const PGBOUNCER_SUFFIX = "-pgbouncer"

func ReconfigurePgbouncerFromTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, task *crv1.Pgtask, namespace string) {
	log.Debugf("ReconfigurePgbouncerFromTask task cluster=[%s]", task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER])

	//look up the pgcluster from the task
	clusterName := task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER]
	pgcluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &pgcluster, clusterName, namespace)
	if !found || err != nil {
		log.Error(err)
		return
	}

	depName := clusterName + "-pgbouncer"
	//remove the pgbouncer deployment (deployment name is the same as svcname)
	err = kubeapi.DeleteDeployment(clientset, depName, namespace)
	if err != nil {
		log.Error(err)
	}

	//remove the pgbouncer secret
	secretName := clusterName + "-pgbouncer-secret"
	err = kubeapi.DeleteSecret(clientset, secretName, namespace)
	if err != nil {
		log.Error(err)
	}

	//check for the deployment to be fully deleted
	for i := 0; i < 10; i++ {
		_, found, err := kubeapi.GetDeployment(clientset, depName, namespace)
		if !found {
			break
		}
		if err != nil {
			log.Error(err)
		}
		log.Debugf("pgbouncer reconfigure sleeping till deployment [%s] is removed", depName)
		time.Sleep(time.Second * time.Duration(4))
	}

	//create the pgbouncer but leave the existing service in place
	err = AddPgbouncer(clientset, &pgcluster, namespace, false)

	//remove task to cleanup
	err = kubeapi.Deletepgtask(restclient, task.Spec.Name, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("reconfigure pgbouncer to cluster [%s]", clusterName)
}

func AddPgbouncerFromTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, task *crv1.Pgtask, namespace string) {
	log.Debug("AddPgbouncerFromTask task cluster=[%s]", task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER])

	//look up the pgcluster from the task
	clusterName := task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER]
	pgcluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &pgcluster, clusterName, namespace)
	if !found || err != nil {
		log.Error(err)
		return
	}

	// add necessary fields from task to the cluster for pgbouncer specifics
	pgcluster.Spec.UserLabels[config.LABEL_PGBOUNCER_USER] = task.Spec.Parameters[config.LABEL_PGBOUNCER_USER]
	pgcluster.Spec.UserLabels[config.LABEL_PGBOUNCER_PASS] = task.Spec.Parameters[config.LABEL_PGBOUNCER_PASS]

	err = AddPgbouncer(clientset, &pgcluster, namespace, true)
	if err != nil {
		log.Error(err)
		return
	}

	//remove task
	err = kubeapi.Deletepgtask(restclient, task.Spec.Name, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	//update the pgcluster CRD
	pgcluster.Spec.UserLabels[config.LABEL_PGBOUNCER] = "true"
	err = kubeapi.Updatepgcluster(restclient, &pgcluster, pgcluster.Name, namespace)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debugf("added pgbouncer to cluster [%s]", clusterName)
}

func DeletePgbouncerFromTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, task *crv1.Pgtask, namespace string) {
	log.Debugf("DeletePgbouncerFromTask task cluster=[%s]", task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER])

	//look up the pgcluster from the task
	clusterName := task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER]
	pgcluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &pgcluster, clusterName, namespace)
	if !found || err != nil {
		log.Error(err)
		return
	}

	//remove the pgbouncer service
	serviceName := clusterName + "-pgbouncer"
	err = kubeapi.DeleteService(clientset, serviceName, namespace)
	if err != nil {
		log.Error(err)
	}

	//remove the pgbouncer deployment (deployment name is the same as svcname)
	err = kubeapi.DeleteDeployment(clientset, serviceName, namespace)
	if err != nil {
		log.Error(err)
	}

	//remove the pgbouncer secret
	secretName := clusterName + "-pgbouncer-secret"
	err = kubeapi.DeleteSecret(clientset, secretName, namespace)
	if err != nil {
		log.Error(err)
	}

	//remove task
	err = kubeapi.Deletepgtask(restclient, task.Spec.Name, namespace)
	if err != nil {
		log.Error(err)
	}

	//update the pgcluster CRD
	pgcluster.Spec.UserLabels[config.LABEL_PGBOUNCER] = "false"
	err = kubeapi.Updatepgcluster(restclient, &pgcluster, pgcluster.Name, namespace)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("delete pgbouncer from cluster [%s]", clusterName)
}

// ProcessPgbouncer ...
func AddPgbouncer(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, namespace string, createService bool) error {
	var doc bytes.Buffer
	var err error

	//generate a secret for pgbouncer using passed in user or default pgbouncer user
	secretName := cl.Spec.Name + "-" + config.LABEL_PGBOUNCER_SECRET
	primaryName := cl.Spec.Name
	replicaName := cl.Spec.Name + "-replica"
	err = createPgbouncerSecret(clientset, cl, primaryName, replicaName, primaryName, secretName, namespace)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debug("pgbouncer secret created")

	clusterName := cl.Spec.Name
	pgbouncerName := clusterName + PGBOUNCER_SUFFIX
	log.Debugf("adding a pgbouncer %s", pgbouncerName)

	//create the pgbouncer deployment
	fields := PgbouncerTemplateFields{
		Name:               pgbouncerName,
		ClusterName:        clusterName,
		CCPImagePrefix:     operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:        cl.Spec.CCPImageTag,
		Port:               operator.Pgo.Cluster.Port,
		PgBouncerUser:      cl.Spec.UserLabels[config.LABEL_PGBOUNCER_USER],
		PgBouncerPass:      cl.Spec.UserLabels[config.LABEL_PGBOUNCER_PASS],
		SecretsName:        secretName,
		ContainerResources: "",
	}

	if operator.Pgo.DefaultPgbouncerResources != "" {
		tmp, err := operator.Pgo.GetContainerResource(operator.Pgo.DefaultPgbouncerResources)
		if err != nil {
			log.Error(err)
			return err
		}
		fields.ContainerResources = operator.GetContainerResourcesJSON(&tmp)

	}

	err = config.PgbouncerTemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err)
		return err
	}

	if operator.CRUNCHY_DEBUG {
		config.PgbouncerTemplate.Execute(os.Stdout, fields)
	}

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(doc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling pgbouncer json into Deployment " + err.Error())
		return err
	}

	err = kubeapi.CreateDeployment(clientset, &deployment, namespace)
	if err != nil {
		log.Error("error creating pgbouncer Deployment " + err.Error())
		return err
	}

	if createService {
		//create a service for the pgbouncer
		svcFields := ServiceTemplateFields{}
		svcFields.Name = pgbouncerName
		svcFields.ServiceName = pgbouncerName
		svcFields.ClusterName = clusterName
		svcFields.Port = operator.Pgo.Cluster.Port

		err = CreateService(clientset, &svcFields, namespace)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	return err
}

// DeletePgbouncer
func DeletePgbouncer(clientset *kubernetes.Clientset, clusterName, namespace string) {

	pgbouncerDepName := clusterName + "-pgbouncer"

	kubeapi.DeleteDeployment(clientset, pgbouncerDepName, namespace)

	//delete the service name=<clustename>-pgbouncer

	kubeapi.DeleteService(clientset, pgbouncerDepName, namespace)

}

// CreatePgbouncerSecret create a secret used by pgbouncer
func createPgbouncerSecret(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, primary, replica, db, secretName, namespace string) error {

	var err error
	var username, password string
	var pgbouncerHBABytes, pgbouncerConfBytes, pgbouncerPasswdBytes []byte

	_, found, err := kubeapi.GetSecret(clientset, secretName, namespace)
	if found {
		log.Debugf("pgbouncer secret %s already present, will reuse", secretName)
		return err
	}

	// pgbouncer port for ini file
	port := cl.Spec.Port

	if !(len(port) > 0) {
		port = "5432" // default if unset in pgo.yaml
	}

	pgbouncerDb := cl.Spec.Database
	if !(len(pgbouncerDb) > 0) {
		pgbouncerDb = db // default to passed in database name
	}

	pgbouncerHBABytes, err = getPgbouncerHBA()
	if err != nil {
		log.Error(err)
		return err
	}

	pgbouncerPasswdBytes, username, password, err = getPgbouncerPasswd(clientset, cl, namespace, db)
	if err != nil {
		log.Error(err)
		return err
	}

	pgbouncerConfBytes, err = getPgbouncerConf(primary, replica, username, password, port, pgbouncerDb)
	if err != nil {
		log.Error(err)
		return err
	}

	secret := v1.Secret{}

	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels[config.LABEL_PG_DATABASE] = db
	secret.ObjectMeta.Labels[config.LABEL_PGBOUNCER] = "true"
	secret.Data = make(map[string][]byte)
	secret.Data["pgbouncer.ini"] = pgbouncerConfBytes
	secret.Data["pg_hba.conf"] = pgbouncerHBABytes
	secret.Data["users.txt"] = pgbouncerPasswdBytes

	err = kubeapi.CreateSecret(clientset, &secret, namespace)

	return err

}

func getPgbouncerHBA() ([]byte, error) {
	var err error

	fields := PgbouncerHBAFields{}

	var doc bytes.Buffer
	err = config.PgbouncerHBATemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err)
		return doc.Bytes(), err
	}
	log.Debug(doc.String())

	return doc.Bytes(), err
}

//NOTE: The config files currently uses the postgres user to admin pgouncer by default
func getPgbouncerConf(primary, replica, username, password, port, database string) ([]byte, error) {
	var err error

	fields := PgbouncerConfFields{}
	fields.PG_PRIMARY_SERVICE_NAME = primary
	fields.PG_REPLICA_SERVICE_NAME = replica
	fields.PG_USERNAME = username
	fields.PG_PASSWORD = password
	fields.PG_PORT = port
	fields.PG_DATABASE = database

	var doc bytes.Buffer
	err = config.PgbouncerConfTemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err)
		return doc.Bytes(), err
	}
	log.Debug(doc.String())

	return doc.Bytes(), err
}

func getPgbouncerPasswd(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, namespace, clusterName string) ([]byte, string, string, error) {
	var doc bytes.Buffer
	var pgbouncerUsername, pgbouncerPassword string

	// command line specified username or default to "pgbouncer" if not set.
	var username = cl.Spec.UserLabels[config.LABEL_PGBOUNCER_USER]
	if !(len(username) > 0) {
		pgbouncerUsername = "pgbouncer"
	} else {
		pgbouncerUsername = username
	}

	var password = cl.Spec.UserLabels[config.LABEL_PGBOUNCER_PASS]
	if !(len(password) > 0) {
		log.Debugf("Using generated password, none provided by user")
		pgbouncerPassword = util.GeneratePassword(10) // default password case when not specified by user.
	} else {
		log.Debugf("using provided pgbouncer password")
		pgbouncerPassword = password
	}

	creds := make([]PgbouncerPasswdFields, 0)

	c := PgbouncerPasswdFields{}
	c.Username = pgbouncerUsername
	c.Password = "md5" + util.GetMD5HashForAuthFile(pgbouncerPassword+pgbouncerUsername)
	creds = append(creds, c)

	err := config.PgbouncerUsersTemplate.Execute(&doc, creds)
	if err != nil {
		log.Error(err)
		return doc.Bytes(), pgbouncerUsername, pgbouncerPassword, err
	}
	log.Debug(doc.String())

	return doc.Bytes(), pgbouncerUsername, pgbouncerPassword, err
}
