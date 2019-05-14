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
	"database/sql"
	"encoding/json"
	"fmt"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// connInfo ....
type connectionInfo struct {
	Username string
	Hostip   string
	Port     string
	Database string
	Password string
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
	err = AddPgbouncer(clientset, restclient, &pgcluster, namespace, false, true)

	//remove task to cleanup
	err = kubeapi.Deletepgtask(restclient, task.Spec.Name, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("reconfigure pgbouncer to cluster [%s]", clusterName)
}

func AddPgbouncerFromTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, task *crv1.Pgtask, namespace string) {
	log.Debugf("AddPgbouncerFromTask task cluster=[ %s ], NS=[ %s ]", task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER], namespace)

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

	err = AddPgbouncer(clientset, restclient, &pgcluster, namespace, true, true)
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
	//	pgcluster.Spec.UserLabels[config.LABEL_PGBOUNCER] = "true"
	pgcluster.Labels[config.LABEL_PGBOUNCER] = "true"
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
	//	pgcluster.Spec.UserLabels[config.LABEL_PGBOUNCER] = "false"
	pgcluster.Labels[config.LABEL_PGBOUNCER] = "false"
	err = kubeapi.Updatepgcluster(restclient, &pgcluster, pgcluster.Name, namespace)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("delete pgbouncer from cluster [%s]", clusterName)
}

// ProcessPgbouncer ...
func AddPgbouncer(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, namespace string, createService bool, updateCreds bool) error {
	var doc bytes.Buffer
	var err error

	//generate a secret for pgbouncer using passed in user or default pgbouncer user
	secretName := cl.Spec.Name + "-" + config.LABEL_PGBOUNCER_SECRET
	primaryName := cl.Spec.Name
	replicaName := cl.Spec.Name + "-replica"
	clusterName := cl.Spec.Name

	// attempt creation of pgbouncer secret, obtains user and pass from passed in vars, existing secret.
	err, secretUser, secretPass := createPgbouncerSecret(clientset, cl, primaryName, replicaName, primaryName, secretName, namespace)

	log.Debugf("secretUser: %s", secretUser)
	log.Debugf("secretPass: %s", secretPass)

	if err != nil {
		log.Error(err)
		return err
	}

	if updateCreds {

		log.Debug("Updating pgbouncer password in secret and database")

		err := UpdatePgBouncerAuthorizations(clientset, namespace, secretUser, secretPass, secretName, clusterName, "")

		if err != nil {
			log.Debug("Failed to update existing pgbouncer credentials")
			log.Debug(err.Error())
			return err
		}
	} else {

		log.Debug("Creating task to update pgbouncer creds in postgres when it becomes ready")

		createPgbouncerDBUpdateTask(restclient, clusterName, secretName, namespace)

	}

	pgbouncerName := clusterName + PGBOUNCER_SUFFIX
	log.Debugf("adding a pgbouncer %s", pgbouncerName)
	//	log.Debugf("secretUser: %s, secretPass: %s", secretUser, secretPass)

	//create the pgbouncer deployment
	fields := PgbouncerTemplateFields{
		Name:               pgbouncerName,
		ClusterName:        clusterName,
		CCPImagePrefix:     operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:        cl.Spec.CCPImageTag,
		Port:               operator.Pgo.Cluster.Port,
		PgBouncerUser:      secretUser,
		PgBouncerPass:      secretPass,
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

	deployment := appsv1.Deployment{}
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

func UpdatePgBouncerAuthorizations(clientset *kubernetes.Clientset, namespace, username, password, secretName, clusterName, podIP string) error {

	err, databaseNames := getDatabaseListForCredentials(namespace, clusterName, clientset, podIP)

	if err != nil {
		log.Debug(err)
		return err
	}

	for _, dbName := range databaseNames {

		connectionInfo := getDBUserInfo(namespace, clusterName, dbName, clientset)

		log.Debugf("Pod ip %s", podIP)

		if len(podIP) > 0 {
			log.Debugf("Updating IP address to use pods IP %s ", podIP)
			connectionInfo.Hostip = podIP
		}

		log.Debugf("Creating pgbouncer authorization in %s database", dbName)

		err = createPgBouncerAuthInDB(clusterName, connectionInfo, username, namespace)

		if err != nil {
			log.Debugf("Unable to create pgbouncer user in %s database", dbName)
			log.Debug(err.Error())
			return err
		}
	}

	// update the password for the pgbouncer user in postgres database
	connectionInfo := getDBUserInfo(namespace, clusterName, "postgres", clientset)

	if len(podIP) > 0 {
		connectionInfo.Hostip = podIP
	}

	err = updatePgBouncerDBPassword(clusterName, connectionInfo, username, password, namespace)

	if err != nil {
		log.Debug("Unable to update pgbouncer password in database.")
		log.Debug(err.Error())
	}

	return err
}

func updatePgBouncerDBPassword(clusterName string, p connectionInfo, username, newPassword, namespace string) error {

	var err error
	var conn *sql.DB

	log.Debugf("Updating credentials for %s in %s with %s ", username, p.Database, newPassword)

	conn, err = sql.Open("postgres", "sslmode=disable user="+p.Username+" host="+p.Hostip+" port="+p.Port+" dbname="+p.Database+" password="+p.Password)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	var rows *sql.Rows
	querystr := "ALTER user " + username + " PASSWORD '" + newPassword + "'"
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()

	return err

}

func createPgBouncerAuthInDB(clusterName string, p connectionInfo, username string, namespace string) error {

	var err error
	var conn *sql.DB

	log.Debugf("Creating %s user for pgbouncer in %s ", username, p.Database)

	conn, err = sql.Open("postgres", "sslmode=disable user="+p.Username+" host="+p.Hostip+" port="+p.Port+" dbname="+p.Database+" password="+p.Password)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	var rows *sql.Rows

	// create pgbouncer role and setup authorization.
	querystr := `
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pgbouncer') THEN
        CREATE ROLE pgbouncer;
    END IF;
END
$$;

ALTER ROLE pgbouncer LOGIN;

CREATE SCHEMA IF NOT EXISTS pgbouncer AUTHORIZATION pgbouncer;

CREATE OR REPLACE FUNCTION pgbouncer.get_auth(p_username TEXT)
RETURNS TABLE(username TEXT, password TEXT) AS
$$
BEGIN
    RAISE WARNING 'PgBouncer auth request: %', p_username;

    RETURN QUERY
    SELECT rolname::TEXT, rolpassword::TEXT
      FROM pg_authid
      WHERE NOT rolsuper
        AND rolname = p_username;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

REVOKE ALL ON FUNCTION pgbouncer.get_auth(p_username TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION pgbouncer.get_auth(p_username TEXT) TO pgbouncer; `

	rows, err = conn.Query(querystr)

	if err != nil {
		log.Debug(err.Error())
		return err
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()

	return err

}

func getDBUserInfo(namespace, clusterName string, targetDB string, clientset *kubernetes.Clientset) connectionInfo {
	info := connectionInfo{}

	//get the service for the cluster
	service, found, err := kubeapi.GetService(clientset, clusterName, namespace)
	if !found || err != nil {
		return info
	}

	//get the pgbouncer secret for this cluster
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName
	secrets, err := kubeapi.GetSecrets(clientset, selector, namespace)
	if err != nil {
		return info
	}

	//get the postgres user secret info
	var username, password, database, hostip string
	for _, s := range secrets.Items {
		username = string(s.Data[config.LABEL_USERNAME][:])
		password = string(s.Data[config.LABEL_PASSWORD][:])
		database = targetDB
		hostip = service.Spec.ClusterIP
		if username == "postgres" {
			log.Debug("got postgres user secrets")
			break
		}
	}

	strPort := fmt.Sprint(service.Spec.Ports[0].Port)
	info.Username = username
	info.Password = password
	info.Database = database
	info.Hostip = hostip
	info.Port = strPort

	return info
}

func getDatabaseListForCredentials(namespace, clusterName string, clientSet *kubernetes.Clientset, podIP string) (error, []string) {

	info := getDBUserInfo(namespace, clusterName, "postgres", clientSet)

	if len(podIP) > 0 {
		log.Debugf("Updating IP address to use pods IP %s for database list request ", podIP)
		info.Hostip = podIP
	}

	log.Debug("Getting list of database names to update for pgbouncer")

	var databases []string

	var err error
	var conn *sql.DB

	conn, err = sql.Open("postgres", "sslmode=disable user="+info.Username+" host="+info.Hostip+" port="+info.Port+" dbname="+info.Database+" password="+info.Password)
	if err != nil {
		log.Debug(err.Error())
		return err, databases
	}

	// get a list of database names from postgres
	var rows *sql.Rows
	querystr := "SELECT datname FROM pg_database WHERE datname NOT IN ('template0', 'template1')"
	rows, err = conn.Query(querystr)
	if err != nil {
		log.Debug(err.Error())
		return err, databases
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()

	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			log.Debug(err)
		}
		databases = append(databases, dbName)
	}

	return err, databases

}

// CreatePgbouncerSecret create a secret used by pgbouncer, return username and password, unencoded.
func createPgbouncerSecret(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, primary, replica, db, secretName, namespace string) (error, string, string) {

	var err error
	var username, password string
	var pgbouncerHBABytes, pgbouncerConfBytes, pgbouncerPasswdBytes []byte

	username, password, err = util.GetPasswordFromSecret(clientset, namespace, secretName)
	if err == nil {
		log.Debugf("pgbouncer secret %s already present, will reuse", secretName)
		return err, username, password
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
		return err, "", ""
	}

	pgbouncerPasswdBytes, username, password, err = getPgbouncerPasswd(clientset, cl, namespace, db)
	if err != nil {
		log.Error(err)
		return err, "", ""
	}

	pgbouncerConfBytes, err = getPgbouncerConf(primary, replica, username, password, port, pgbouncerDb)
	if err != nil {
		log.Error(err)
		return err, "", ""
	}

	secret := v1.Secret{}

	secret.Name = secretName
	secret.ObjectMeta.Labels = make(map[string]string)
	secret.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = db
	secret.ObjectMeta.Labels[config.LABEL_PGBOUNCER] = "true"
	secret.Data = make(map[string][]byte)

	secret.Data["username"] = []byte(username)
	secret.Data["password"] = []byte(password)

	secret.Data["pgbouncer.ini"] = pgbouncerConfBytes
	secret.Data["pg_hba.conf"] = pgbouncerHBABytes
	secret.Data["users.txt"] = pgbouncerPasswdBytes

	err = kubeapi.CreateSecret(clientset, &secret, namespace)

	return err, username, password

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

// provides encoded pw for pgbouncer in the byte array as well as user and pw in unencoded strings.
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
		log.Debugf("Pgbouncer: creating password, none provided by user")
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

func createPgbouncerDBUpdateTask(restclient *rest.RESTClient, clusterName, secretName, ns string) error {

	//create pgtask CRD
	taskName := clusterName + crv1.PgtaskUpdatePgbouncerAuths

	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = taskName
	spec.TaskType = crv1.PgtaskUpdatePgbouncerAuths

	spec.Parameters = make(map[string]string)
	//	spec.Parameters[config.LABEL_PVC_NAME] = pvcName
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}

	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[config.LABEL_PGBOUNCER_SECRET] = secretName
	err := kubeapi.Createpgtask(restclient, newInstance, ns)
	if err != nil {
		log.Error(err)
		//	return err
	}

	return err
}
