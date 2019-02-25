package cluster

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
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
}

const PGBOUNCER_SUFFIX = "-pgbouncer"

func ReconfigurePgbouncerFromTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, task *crv1.Pgtask, namespace string) {
	log.Debugf("ReconfigurePgbouncerFromTask task cluster=[%s]", task.Spec.Parameters[util.LABEL_PGBOUNCER_TASK_CLUSTER])

	//look up the pgcluster from the task
	clusterName := task.Spec.Parameters[util.LABEL_PGBOUNCER_TASK_CLUSTER]
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
	log.Debug("AddPgbouncerFromTask task cluster=[%s]", task.Spec.Parameters[util.LABEL_PGBOUNCER_TASK_CLUSTER])

	//look up the pgcluster from the task
	clusterName := task.Spec.Parameters[util.LABEL_PGBOUNCER_TASK_CLUSTER]
	pgcluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &pgcluster, clusterName, namespace)
	if !found || err != nil {
		log.Error(err)
		return
	}
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
	pgcluster.Spec.UserLabels[util.LABEL_PGBOUNCER] = "true"
	err = kubeapi.Updatepgcluster(restclient, &pgcluster, pgcluster.Name, namespace)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debugf("added pgbouncer to cluster [%s]", clusterName)
}

func DeletePgbouncerFromTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, task *crv1.Pgtask, namespace string) {
	log.Debugf("DeletePgbouncerFromTask task cluster=[%s]", task.Spec.Parameters[util.LABEL_PGBOUNCER_TASK_CLUSTER])

	//look up the pgcluster from the task
	clusterName := task.Spec.Parameters[util.LABEL_PGBOUNCER_TASK_CLUSTER]
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
	pgcluster.Spec.UserLabels[util.LABEL_PGBOUNCER] = "false"
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

	//generate a secret for pgbouncer using the testuser credential
	secretName := cl.Spec.Name + "-" + util.LABEL_PGBOUNCER_SECRET
	primaryName := cl.Spec.Name
	replicaName := cl.Spec.Name + "-replica"
	err = CreatePgbouncerSecret(clientset, primaryName, replicaName, primaryName, secretName, namespace)
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

	err = operator.PgbouncerTemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err)
		return err
	}

	if operator.CRUNCHY_DEBUG {
		operator.PgbouncerTemplate.Execute(os.Stdout, fields)
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
func CreatePgbouncerSecret(clientset *kubernetes.Clientset, primary, replica, db, secretName, namespace string) error {

	var err error
	var username, password string
	var pgbouncerHBABytes, pgbouncerConfBytes, pgbouncerPasswdBytes []byte

	_, found, err := kubeapi.GetSecret(clientset, secretName, namespace)
	if found {
		log.Debugf("pgbouncer secret %s already present, will reuse", secretName)
		return err
	}

	pgbouncerHBABytes, err = getPgbouncerHBA()
	if err != nil {
		log.Error(err)
		return err
	}

	pgbouncerPasswdBytes, username, password, err = getPgbouncerPasswd(clientset, namespace, db)
	if err != nil {
		log.Error(err)
		return err
	}

	pgbouncerConfBytes, err = getPgbouncerConf(primary, replica, username, password)
	if err != nil {
		log.Error(err)
		return err
	}

	secret := v1.Secret{}

	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels[util.LABEL_PG_DATABASE] = db
	secret.ObjectMeta.Labels[util.LABEL_PGBOUNCER] = "true"
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
	err = operator.PgbouncerHBATemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err)
		return doc.Bytes(), err
	}
	log.Debug(doc.String())

	return doc.Bytes(), err
}

//NOTE: The config files currently uses the postgres user to admin pgouncer by default
func getPgbouncerConf(primary, replica, username, password string) ([]byte, error) {
	var err error

	fields := PgbouncerConfFields{}
	fields.PG_PRIMARY_SERVICE_NAME = primary
	fields.PG_REPLICA_SERVICE_NAME = replica
	fields.PG_USERNAME = username
	fields.PG_PASSWORD = password

	var doc bytes.Buffer
	err = operator.PgbouncerConfTemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err)
		return doc.Bytes(), err
	}
	log.Debug(doc.String())

	return doc.Bytes(), err
}

func getPgbouncerPasswd(clientset *kubernetes.Clientset, namespace, clusterName string) ([]byte, string, string, error) {
	var doc bytes.Buffer
	var pgbouncerUsername, pgbouncerPassword string

	//go get all non-pgbouncer secrets
	selector := util.LABEL_PG_DATABASE + "=" + clusterName + "," + util.LABEL_PGBOUNCER + "!=true"
	secrets, err := kubeapi.GetSecrets(clientset, selector, namespace)
	if err != nil {
		log.Error(err)
		return doc.Bytes(), pgbouncerUsername, pgbouncerPassword, err
	}

	creds := make([]PgbouncerPasswdFields, 0)
	for _, sec := range secrets.Items {
		//log.Debugf("in pgbouncer passwd with username=%s password=%s\n", sec.Data[util.LABEL_USERNAME][:], sec.Data[util.LABEL_PASSWORD][:])
		username := string(sec.Data[util.LABEL_USERNAME][:])
		password := string(sec.Data[util.LABEL_PASSWORD][:])
		c := PgbouncerPasswdFields{}
		c.Username = username
		c.Password = "md5" + util.GetMD5HashForAuthFile(password+username)
		creds = append(creds, c)

		//we will use the postgres user for pgbouncer to auth with
		if username == "postgres" {
			pgbouncerUsername = username
			pgbouncerPassword = password
		}
	}

	err = operator.PgbouncerUsersTemplate.Execute(&doc, creds)
	if err != nil {
		log.Error(err)
		return doc.Bytes(), pgbouncerUsername, pgbouncerPassword, err
	}
	log.Debug(doc.String())

	return doc.Bytes(), pgbouncerUsername, pgbouncerPassword, err
}
