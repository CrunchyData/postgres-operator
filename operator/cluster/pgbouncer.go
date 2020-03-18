package cluster

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PgbouncerPasswdFields struct {
	Username string
	Password string
}

type PgbouncerConfFields struct {
	PG_PRIMARY_SERVICE_NAME string
	PG_PORT                 string
}

type PgbouncerTemplateFields struct {
	Name                      string
	ClusterName               string
	PGBouncerSecret           string
	CCPImagePrefix            string
	CCPImageTag               string
	Port                      string
	PrimaryServiceName        string
	ContainerResources        string
	PodAntiAffinity           string
	PodAntiAffinityLabelName  string
	PodAntiAffinityLabelValue string
}

// pgBouncerDeploymentFormat is the name of the Kubernetes Deployment that
// manages pgBouncer, and follows the format "<clusterName>-pgbouncer"
const pgBouncerDeploymentFormat = "%s-pgbouncer"

// ...the default PostgreSQL port
const pgPort = "5432"

const (
	// the path to the pgbouncer uninstallation script script
	pgBouncerUninstallScript = "/opt/cpm/bin/sql/pgbouncer/pgbouncer-uninstall.sql"

	// the path to the pgbouncer installation script
	pgBouncerInstallScript = "/opt/cpm/bin/sql/pgbouncer/pgbouncer-install.sql"
)

const (
	// pgBouncerSecretPropagationPeriod is the number of seconds between each
	// check of when the secret is propogated
	pgBouncerSecretPropagationPeriod = 5
	// pgBouncerSecretPropagationTimeout is the maximum amount of time in seconds
	// to wait for the secret to propagate
	pgBouncerSecretPropagationTimeout = 60
)

const (
	// a string to check to see if the pgbouncer machinery is installed in the
	// PostgreSQL cluster
	sqlCheckPgBouncerInstall = `SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'pgbouncer' LIMIT 1);`

	// disable the pgbouncer user from logging in. This is safe from SQL injection
	// as the string that is being interpolated is the util.PgBouncerUser constant
	//
	// This had the "PASSWORD NULL" feature, but this is only found in
	// PostgreSQL 11+, and given we don't want to check for the PG version before
	// running the command, we will not use it
	sqlDisableLogin = `ALTER ROLE "%s" NOLOGIN;`

	// sqlEnableLogin is the SQL to update the password
	// NOTE: this is safe from SQL injection as we explicitly add the inerpolated
	// string as a MD5 hash and we are using the crv1.PGUserPgBouncer constant
	// However, the escaping is handled in the util.SetPostgreSQLPassword function
	sqlEnableLogin = `ALTER ROLE %s PASSWORD %s LOGIN;`

	// sqlGetDatabasesForPgBouncer gets all the databases where pgBouncer can be
	// installed or uninstalled
	sqlGetDatabasesForPgBouncer = `SELECT datname FROM pg_catalog.pg_database WHERE datname NOT IN ('postgres', 'template0') AND datallowconn;`
)

var (
	// this command allows one to view the users.txt file secret to determine if
	// it has propagated
	cmdViewPgBouncerUsersSecret = []string{"cat", "/pgconf/users.txt"}
	// sqlUninstallPgBouncer provides the final piece of SQL to uninstall
	// pgbouncer, which is to remove the user
	sqlUninstallPgBouncer = fmt.Sprintf(`DROP ROLE "%s";`, crv1.PGUserPgBouncer)
)

// AddPgbouncer contains the various functions that are used to add a pgBouncer
// Deployment to a PostgreSQL cluster
//
// Any returned error is logged in the calling function
func AddPgbouncer(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	log.Debugf("adding a pgbouncer")

	// first, ensure that the Cluster CR is updated to know that there is now
	// a pgBouncer associated with it
	// if we cannot update this we abort
	cluster.Labels[config.LABEL_PGBOUNCER] = "true"

	if err := kubeapi.Updatepgcluster(restclient, cluster, cluster.Spec.ClusterName, cluster.Namespace); err != nil {
		return err
	}

	// get the primary pod, which is needed to update the password for the
	// pgBouncer administrative user
	pod, err := util.GetPrimaryPod(clientset, cluster)

	if err != nil {
		return err
	}

	// check to see if pgBoncer is "installed" in the PostgreSQL cluster. This
	// means checking to see if there is a pgbouncer user, effetively
	if installed, err := checkPgBouncerInstall(clientset, restconfig, pod); err != nil {
		return err
	} else if !installed {
		if err := installPgBouncer(clientset, restconfig, pod); err != nil {
			return err
		}
	}

	// set the password that will be used for the "pgbouncer" PostgreSQL account
	pgBouncerPassword := generatePassword()

	// only attempt to set the password if the cluster is not in standby mode
	if !cluster.Spec.Standby {
		// attempt to update the password in PostgreSQL, as this is how pgBouncer
		// will properly interface with PostgreSQL
		if err := setPostgreSQLPassword(clientset, restconfig, pod, pgBouncerPassword); err != nil {
			return err
		}
	}

	// next, create the secret that pgbouncer will use to be properly configure
	if err := createPgbouncerSecret(clientset, cluster, pgBouncerPassword); err != nil {
		return err
	}

	// next, create the pgBouncer deployment
	if err := createPgBouncerDeployment(clientset, cluster); err != nil {
		return err
	}

	// finally, try to create the pgBouncer service
	if err := createPgBouncerService(clientset, cluster); err != nil {
		return err
	}

	log.Debugf("added pgbouncer to cluster [%s]", cluster.Spec.Name)

	return nil
}

// AddPgbouncerFromPgTask is effectively a legacy method that helps to bring up
// the pgBouncer deployment that sits alongside a PostgreSQL cluster
func AddPgbouncerFromPgTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, task *crv1.Pgtask) {
	clusterName := task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER]
	namespace := task.Spec.Namespace

	log.Debugf("add pgbouncer from task called for cluster [%s] in namespace [%s]",
		clusterName, namespace)

	// first, check to ensure that the cluster still exosts
	cluster := crv1.Pgcluster{}

	if found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace); !found || err != nil {
		// even if it's not found, this is pretty bad and we cannot continue
		log.Error(err)
		return
	}

	// bring up the pgbouncer deployment and all of its trappings!
	if err := AddPgbouncer(clientset, restclient, restconfig, &cluster); err != nil {
		log.Error(err)
		return
	}

	// publish an event
	publishPgBouncerEvent(events.EventCreatePgbouncer, task)

	// at this point, the pgback is successful, so we can safely remove it
	// we can fallthrough in the event of an error, because we're returning anyway
	if err := kubeapi.Deletepgtask(restclient, task.Spec.Name, namespace); err != nil {
		log.Error(err)
	}

	return
}

// CreatePgTaskforAddpgBouncer creates a pgtask to process adding a pgBouncer
func CreatePgTaskforAddpgBouncer(restclient *rest.RESTClient, cluster *crv1.Pgcluster, pgouser string) error {
	log.Debugf("create pgtask for adding pgbouncer to cluster [%s]", cluster.Spec.ClusterName)

	// generate the pgtask, first setting up the parameters it needs
	parameters := map[string]string{
		config.LABEL_PGBOUNCER_TASK_CLUSTER: cluster.Spec.ClusterName,
	}
	task := generatePgtaskForPgBouncer(cluster, pgouser,
		crv1.PgtaskAddPgbouncer, config.LABEL_PGBOUNCER_TASK_ADD, parameters)

	// try to create the pgtask!
	if err := kubeapi.Createpgtask(restclient, task, task.Spec.Namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// CreatePgTaskforUpdatepgBouncer creates a pgtask to process for updating
// pgBouncer
//
// The "parameters" attribute contains a list of parameters that can guide what
// will take place during an update, e.g.
//
// - RotoatePassword="true" will rotate the pgBouncer PostgreSQL user password
func CreatePgTaskforUpdatepgBouncer(restclient *rest.RESTClient, cluster *crv1.Pgcluster, pgouser string, parameters map[string]string) error {
	log.Debugf("create pgtask for updating pgbouncer for cluster [%s]", cluster.Spec.ClusterName)

	// generate the pgtask, first adding in some boilerplate parameters
	parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER] = cluster.Spec.ClusterName
	log.Debugf("ANDY bouncer pgouser: %+v", pgouser)
	task := generatePgtaskForPgBouncer(cluster, pgouser,
		crv1.PgtaskUpdatePgbouncer, config.LABEL_PGBOUNCER_TASK_UPDATE, parameters)
	log.Debugf("ANDY bouncer task: %+v", task)
	// try to create the pgtask!
	if err := kubeapi.Createpgtask(restclient, task, task.Spec.Namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// DeletePgbouncer contains the various functions that are used to delete a
// pgBouncer Deployment for a PostgreSQL cluster
//
// Note that "uninstall" deletes all of the objects that are added to the
// PostgreSQL database, such as the "pgbouncer" user. This is not normally
// neded to be done as pgbouncer user is disabled, but if the user wishes to be
// thorough they can do this
//
// Any errors that are returned should be logged in the calling function, though
// some logging occurs in this function as well
func DeletePgbouncer(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, cluster *crv1.Pgcluster, uninstall bool) error {
	clusterName := cluster.Spec.ClusterName
	namespace := cluster.Spec.Namespace

	log.Debugf("delete pgbouncer from cluster [%s] in namespace [%s]", clusterName, namespace)

	// first, ensure that the Cluster CR is updated to know that there is no
	// longer a pgBouncer associated with it
	// if we cannot update this we abort
	cluster.Labels[config.LABEL_PGBOUNCER] = "false"

	if err := kubeapi.Updatepgcluster(restclient, cluster, cluster.Spec.ClusterName, namespace); err != nil {
		return err
	}

	// next, disable the pgbouncer user in the PostgreSQL cluster.
	// first, get the primary pod. If we cannot do this, let's consider it an
	// error and abort
	pod, err := util.GetPrimaryPod(clientset, cluster)

	if err != nil {
		return err
	}

	// This is safe from SQL injection as we are using constants and a well defined
	// string
	sql := strings.NewReader(fmt.Sprintf(sqlDisableLogin, crv1.PGUserPgBouncer))
	cmd := []string{"psql"}

	// exec into the pod to run the query
	_, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		cmd, "database", pod.Name, pod.ObjectMeta.Namespace, sql)

	// if there is an error, log the error from the stderr and return the error
	if err != nil {
		log.Error(stderr)
		return err
	}

	// next, delete the various Kubernetes objects associated with the pgbouncer
	// these include the Service, Deployment, and the pgBouncer secret
	// If these fail, we'll just pass through
	//
	// First, delete the Service and Deployment, which share the same naem
	pgbouncerDeploymentName := fmt.Sprintf(pgBouncerDeploymentFormat, clusterName)

	if err := kubeapi.DeleteService(clientset, pgbouncerDeploymentName, namespace); err != nil {
		log.Warn(err)
	}

	if err := kubeapi.DeleteDeployment(clientset, pgbouncerDeploymentName, namespace); err != nil {
		log.Warn(err)
	}

	// remove the secret. again, if this fails, just log the error and apss
	// through
	secretName := util.GeneratePgBouncerSecretName(clusterName)

	if err := kubeapi.DeleteSecret(clientset, secretName, namespace); err != nil {
		log.Warn(err)
	}

	// lastly, if uninstall is set, remove the pgbouncer owned objects from the
	// PostgreSQL cluster, and the pgbouncer as well
	if uninstall {
		// if the uninstall fails, only warn that it fails
		if err := uninstallPgBouncer(clientset, restconfig, pod); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

// DeletePgbouncerFromPgTask is effectively a legacy method that helps to delete
// the pgBouncer deployment that sits alongside a PostgreSQL cluster
func DeletePgbouncerFromPgTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, task *crv1.Pgtask) {
	clusterName := task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER]
	namespace := task.Spec.Namespace
	uninstall, _ := strconv.ParseBool(task.Spec.Parameters[config.LABEL_PGBOUNCER_UNINSTALL])

	log.Debugf("delete pgbouncer from task called for cluster [%s] in namespace [%s]",
		clusterName, namespace)

	// find the pgcluster that is associated with this task
	cluster := crv1.Pgcluster{}

	if found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace); !found || err != nil {
		// if even if it's found and there is an error, it's pretty bad so abort
		log.Error(err)
		return
	}

	// attempt to delete the pgbouncer!
	if err := DeletePgbouncer(clientset, restclient, restconfig, &cluster, uninstall); err != nil {
		log.Error(err)
		return
	}

	// publish an event
	publishPgBouncerEvent(events.EventDeletePgbouncer, task)

	// lastly, remove the task
	if err := kubeapi.Deletepgtask(restclient, task.Spec.Name, namespace); err != nil {
		log.Warn(err)
	}
}

// UpdatePgbouncer contains the various functions that are used to perform
// updates to the pgBouncer deployment for a cluster, such as rotating a
// password
//
// Any errors that are returned should be logged in the calling function, though
// some logging occurs in this function as well
func UpdatePgbouncer(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, cluster *crv1.Pgcluster, parameters map[string]string) error {
	clusterName := cluster.Spec.ClusterName
	namespace := cluster.Spec.Namespace

	log.Debugf("update pgbouncer from cluster [%s] in namespace [%s] with parameters [%v]", clusterName, namespace)

	// Alright, so we need to figure out which parameters are set, so we can take
	// action on them
	for param, _ := range parameters {
		switch param {
		// determine if we need to rotate the password. if there is an error, return
		// early as we cannot guarantee anything else can occur
		case config.LABEL_PGBOUNCER_ROTATE_PASSWORD:
			if err := rotatePgBouncerPassword(clientset, restclient, restconfig, cluster); err != nil {
				return err
			}
		}
	}

	// and that's it!
	return nil
}

// UpdatePgbouncerFromPgTask is effectively a legacy method (though modernized to fit this patterh)
// that checks basic information about the Pgtask that is being present, and delegates the actual
// work to UpdatePgBouncer
func UpdatePgbouncerFromPgTask(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, task *crv1.Pgtask) {
	clusterName := task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER]
	namespace := task.Spec.Namespace
	parameters := task.Spec.Parameters

	log.Debugf("update pgbouncer from task called for cluster [%s] in namespace [%s] with parameters [%+v]",
		clusterName, namespace, parameters)

	// find the pgcluster that is associated with this task
	cluster := crv1.Pgcluster{}

	if found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace); !found || err != nil {
		// if even if it's found and there is an error, it's pretty bad so abort
		log.Error(err)
		return
	}

	// attempt to delete the pgbouncer!
	if err := UpdatePgbouncer(clientset, restclient, restconfig, &cluster, parameters); err != nil {
		log.Error(err)
		return
	}

	// publish an event
	publishPgBouncerEvent(events.EventUpdatePgbouncer, task)

	// lastly, remove the task
	if err := kubeapi.Deletepgtask(restclient, task.Spec.Name, namespace); err != nil {
		log.Warn(err)
	}
}

// checkPgBouncerInstall checks to see if pgBouncer is installed in the
// PostgreSQL custer, which involves check to see if the pgBouncer role is
// present in the PostgreSQL cluster
func checkPgBouncerInstall(clientset *kubernetes.Clientset, restconfig *rest.Config, pod *v1.Pod) (bool, error) {
	// set up the SQL
	sql := strings.NewReader(sqlCheckPgBouncerInstall)

	// have the command return an unaligned string of just the "t" or "f"
	cmd := []string{"psql", "-A", "-t"}

	// exec into the pod to run the query
	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		cmd, "database", pod.Name, pod.ObjectMeta.Namespace, sql)

	// if there is an error executing the command, log the error message from
	// stderr and return the error
	if err != nil {
		log.Error(stderr)
		return false, err
	}

	// next, parse the boolean value and determine if the pgbouncer user is
	// present
	if installed, err := strconv.ParseBool(strings.TrimSpace(stdout)); err != nil {
		return false, err
	} else {
		return installed, nil
	}
}

// createPgBouncerDeployment creates the Kubernetes Deployment for pgBouncer
func createPgBouncerDeployment(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster) error {
	log.Debugf("creating pgbouncer deployment: %s", cluster.Spec.Name)

	// derive the name of the Deployment...which is also used as the name of the
	// service
	pgbouncerDeploymentName := fmt.Sprintf(pgBouncerDeploymentFormat, cluster.Spec.Name)

	// get the fields that will be substituted in the pgBouncer template
	fields := PgbouncerTemplateFields{
		Name:               pgbouncerDeploymentName,
		ClusterName:        cluster.Spec.Name,
		CCPImagePrefix:     operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:        cluster.Spec.CCPImageTag,
		Port:               operator.Pgo.Cluster.Port,
		PGBouncerSecret:    util.GeneratePgBouncerSecretName(cluster.Spec.Name),
		ContainerResources: "",
		PodAntiAffinity: operator.GetPodAntiAffinity(cluster,
			crv1.PodAntiAffinityDeploymentPgBouncer, cluster.Spec.PodAntiAffinity.PgBouncer),
		PodAntiAffinityLabelName: config.LABEL_POD_ANTI_AFFINITY,
		PodAntiAffinityLabelValue: string(operator.GetPodAntiAffinityType(cluster,
			crv1.PodAntiAffinityDeploymentPgBouncer, cluster.Spec.PodAntiAffinity.PgBouncer)),
	}

	// Determine if a custom resource profile should be used for the pgBouncer
	// deployment
	if operator.Pgo.DefaultPgbouncerResources != "" {
		pgBouncerResources, err := operator.Pgo.GetContainerResource(operator.Pgo.DefaultPgbouncerResources)

		// if there is an error getting this, log it as an error, but continue on
		if err != nil {
			log.Warn(err)
		}

		fields.ContainerResources = operator.GetContainerResourcesJSON(&pgBouncerResources)
	}

	// For debugging purposes, put the template substitution in stdout
	if operator.CRUNCHY_DEBUG {
		config.PgbouncerTemplate.Execute(os.Stdout, fields)
	}

	// perform the actual template substitution
	doc := bytes.Buffer{}

	if err := config.PgbouncerTemplate.Execute(&doc, fields); err != nil {
		return err
	}

	// Set up the Kubernetes deployment for pgBouncer
	deployment := appsv1.Deployment{}

	if err := json.Unmarshal(doc.Bytes(), &deployment); err != nil {
		return err
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_CRUNCHY_PGBOUNCER,
		&deployment.Spec.Template.Spec.Containers[0])

	if err := kubeapi.CreateDeployment(clientset, &deployment, cluster.Spec.Namespace); err != nil {
		return err
	}

	return nil
}

// createPgbouncerSecret create a secret used by pgbouncer. Returns the
// plaintext password and/or an error
func createPgbouncerSecret(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster, password string) error {
	secretName := util.GeneratePgBouncerSecretName(cluster.Spec.Name)

	// see if this secret already exists...if it does, then take an early exit
	if _, err := util.GetPasswordFromSecret(clientset, cluster.Spec.Namespace, secretName); err == nil {
		log.Debugf("pgbouncer secret %s already present, will reuse", secretName)
		return nil
	}

	// the remainder of this is generating the various entries in the pgbouncer
	// secret, i.e. substituting values into templates files that contain:
	// - the pgbouncer.ini file
	// - the pgbouncer HBA file
	// - the pgbouncer "users.txt" file that contains the credentials for the
	// "pgbouncer" user

	// first, generate the pgbouncer.ini information
	pgBouncerConf, err := generatePgBouncerConf(cluster)

	if err != nil {
		log.Error(err)
		return err
	}

	// finally, generate the pgbouncer HBA file
	pgbouncerHBA, err := generatePgBouncerHBA()

	if err != nil {
		log.Error(err)
		return err
	}

	// now, we can do what we came here to do, which is create the secret
	secret := v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: secretName,
			Labels: map[string]string{
				config.LABEL_PG_CLUSTER: cluster.Spec.Name,
				config.LABEL_PGBOUNCER:  "true",
				config.LABEL_VENDOR:     config.LABEL_CRUNCHY,
			},
		},
		Data: map[string][]byte{
			"password":      []byte(password),
			"pgbouncer.ini": pgBouncerConf,
			"pg_hba.conf":   pgbouncerHBA,
			"users.txt": util.GeneratePgBouncerUsersFileBytes(
				util.GeneratePostgreSQLMD5Password(crv1.PGUserPgBouncer, password)),
		},
	}

	if err := kubeapi.CreateSecret(clientset, &secret, cluster.Spec.Namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// createPgBouncerService creates the Kubernetes Service for pgBouncer
func createPgBouncerService(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster) error {
	// pgBouncerServiceName is the name of the Service of the pgBouncer, which
	// matches that for the Deploymnt
	pgBouncerServiceName := fmt.Sprintf(pgBouncerDeploymentFormat, cluster.Spec.Name)

	// set up the service template fields
	fields := ServiceTemplateFields{
		Name:        pgBouncerServiceName,
		ServiceName: pgBouncerServiceName,
		ClusterName: cluster.Spec.Name,
		// TODO: I think "port" needs to be evaluated, but I think for now using
		// the standard PostgreSQL port works
		Port: operator.Pgo.Cluster.Port,
	}

	if err := CreateService(clientset, &fields, cluster.Spec.Namespace); err != nil {
		return err
	}

	return nil
}

// execPgBouncerScript runs a script pertaining to the management of pgBouncer
// on the PostgreSQL pod
func execPgBouncerScript(clientset *kubernetes.Clientset, restconfig *rest.Config, pod *v1.Pod, databaseName, script string) {
	cmd := []string{"psql", databaseName, "-f", script}

	// exec into the pod to run the query
	_, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		cmd, "database", pod.Name, pod.ObjectMeta.Namespace, nil)

	// if there is an error executing the command, log the error as a warning
	// that it failed, and continue. It's hard to rollback from this one :\
	if err != nil {
		log.Warn(stderr)
		log.Warnf("You can attempt to rerun the script [%s] on [%s]",
			script, databaseName)
	}
}

// generatePassword generates a password that is used for the "pgbouncer"
// PostgreSQL user that provides the associated pgBouncer functionality
func generatePassword() string {
	// first, get the length of what the password should be
	generatedPasswordLength := util.GeneratedPasswordLength(operator.Pgo.Cluster.PasswordLength)
	// from there, the password can be generated!
	return util.GeneratePassword(generatedPasswordLength)
}

// generatePgBouncerConf generates the content that is stored in the secret
// for the "pgbouncer.ini" file
func generatePgBouncerConf(cluster *crv1.Pgcluster) ([]byte, error) {
	// first, get the port
	port := cluster.Spec.Port
	// if the "port" value is not set, default to the PostgreSQL port.
	if port == "" {
		port = pgPort
	}

	// set up the substitution fields for the pgbouncer.ini file
	fields := PgbouncerConfFields{
		PG_PRIMARY_SERVICE_NAME: cluster.Spec.Name,
		PG_PORT:                 port,
	}

	// perform the substitution
	doc := bytes.Buffer{}

	// if there is an error, return an empty byte slice
	if err := config.PgbouncerConfTemplate.Execute(&doc, fields); err != nil {
		log.Error(err)

		return []byte{}, err
	}

	log.Debug(doc.String())

	// and if not, return the full byte slice
	return doc.Bytes(), nil
}

// generatePgBouncerConf generates the pgBouncer host-based authentication file
// using the template that is vailable
func generatePgBouncerHBA() ([]byte, error) {
	// ...apparently this is overkill, but this is here from the legacy method
	// and it seems like it's "ok" to leave it like this for now...
	doc := bytes.Buffer{}

	if err := config.PgbouncerHBATemplate.Execute(&doc, struct{}{}); err != nil {
		log.Error(err)

		return []byte{}, err
	}

	log.Debug(doc.String())

	return doc.Bytes(), nil
}

// generatePgtaskForPgBouncer generates a pgtask specific to a pgbouncer
// deployment
func generatePgtaskForPgBouncer(cluster *crv1.Pgcluster, pgouser, taskType, taskLabel string, parameters map[string]string) *crv1.Pgtask {
	// create the specfile with the required parameters for creating a pgtask
	spec := crv1.PgtaskSpec{
		Namespace:  cluster.Namespace,
		Name:       fmt.Sprintf("%s-%s", taskLabel, cluster.Spec.ClusterName),
		TaskType:   taskType,
		Parameters: parameters,
	}

	// create the pgtask object
	task := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: spec.Name,
			Labels: map[string]string{
				config.LABEL_PG_CLUSTER: cluster.Spec.ClusterName,
				taskLabel:               "true",
				config.LABEL_PGOUSER:    pgouser,
			},
		},
		Spec: spec,
	}

	return task
}

// getPgBouncerDatabases gets the databases in a PostgreSQL cluster that have
// the pgBouncer objects, etc.
func getPgBouncerDatabases(clientset *kubernetes.Clientset, restconfig *rest.Config, pod *v1.Pod) (*bufio.Scanner, error) {
	// so the way this works is that there needs to be a special SQL installation
	// script that is executed on every database EXCEPT for postgres and template0
	// but is executed on template1
	sql := strings.NewReader(sqlGetDatabasesForPgBouncer)

	// have the command return an unaligned string of just the "t" or "f"
	cmd := []string{"psql", "-A", "-t"}

	// exec into the pod to run the query
	stdout, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		cmd, "database", pod.Name, pod.ObjectMeta.Namespace, sql)

	// if there is an error executing the command, log the error message from
	// stderr and return the error
	if err != nil {
		log.Error(stderr)
		return nil, err
	}

	// return the list of databases, that will be in a multi-line string
	return bufio.NewScanner(strings.NewReader(stdout)), nil
}

// installPgBouncer installs the "pgbouncer" user and other management objects
// into the PostgreSQL pod
func installPgBouncer(clientset *kubernetes.Clientset, restconfig *rest.Config, pod *v1.Pod) error {
	// get the list of databases that we need to scan through
	databases, err := getPgBouncerDatabases(clientset, restconfig, pod)

	if err != nil {
		return err
	}

	// iterate through the list of databases that are returned, and execute the
	// installation script
	for databases.Scan() {
		databaseName := strings.TrimSpace(databases.Text())

		execPgBouncerScript(clientset, restconfig, pod, databaseName, pgBouncerInstallScript)
	}

	return nil
}

// publishPgBouncerEvent publishes one of the events on the event stream
func publishPgBouncerEvent(eventType string, task *crv1.Pgtask) {
	var event events.EventInterface

	// prepare the topics to publish to
	topics := []string{events.EventTopicPgbouncer}
	// set up the event header
	eventHeader := events.EventHeader{
		Namespace: task.Spec.Namespace,
		Username:  task.ObjectMeta.Labels[config.LABEL_PGOUSER],
		Topic:     topics,
		Timestamp: time.Now(),
		EventType: eventType,
	}
	clusterName := task.Spec.Parameters[config.LABEL_PGBOUNCER_TASK_CLUSTER]

	// now determine which event format to use!
	switch eventType {
	case events.EventCreatePgbouncer:
		event = events.EventCreatePgbouncerFormat{
			EventHeader: eventHeader,
			Clustername: clusterName,
		}
	case events.EventUpdatePgbouncer:
		event = events.EventUpdatePgbouncerFormat{
			EventHeader: eventHeader,
			Clustername: clusterName,
		}
	case events.EventDeletePgbouncer:
		event = events.EventDeletePgbouncerFormat{
			EventHeader: eventHeader,
			Clustername: clusterName,
		}
	}

	// publish the event; if there is an error, log it, but we don't care
	if err := events.Publish(event); err != nil {
		log.Error(err.Error())
	}
}

// rotatePgBouncerPassword rotates the password for a pgBouncer PostgreSQL user,
// which involves updating the password in the PostgreSQL cluster as well as
// the users secret that is available in the pgbouncer Pod
func rotatePgBouncerPassword(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	namspace := cluster.Spec.Namespace
	// determine if we are able to access the primary Pod
	primaryPod, err := util.GetPrimaryPod(clientset, cluster)

	if err != nil {
		return err
	}

	// let's also go ahead and get the secret that contains the pgBouncer
	// information. If we can't find the secret, we're basically done here
	secretName := util.GeneratePgBouncerSecretName(cluster.Spec.Name)
	secret, _, err := kubeapi.GetSecret(clientset, secretName, namspace)

	if err != nil {
		return err
	}

	// there are a few steps that must occur in order for the password to be
	// successfully rotated:
	//
	// 1. The PostgreSQL cluster must have the pgbouncer user's password updated
	// 2. The secret that containers the values of "users.txt" must be updated
	// 3. The pgBouncer pods must be bounced and have the new password loaded, but
	// 		we must first ensure the password propagates to them
	//
	// ...wouldn't it be nice if we could run this in a transaction? rolling back
	// is hard :(

	// first, generate a new password
	password := generatePassword()

	// next, update the PostgreSQL primary with the new password. If this fails
	// we definitely return an error
	if err := setPostgreSQLPassword(clientset, restconfig, primaryPod, password); err != nil {
		return err
	}

	// next, update the users.txt and password fields of the secret. the important
	// one to update is the users.txt, as that is used by pgbouncer to connect to
	// PostgreSQL to perform its authentication
	secret.Data["password"] = []byte(password)
	secret.Data["users.txt"] = util.GeneratePgBouncerUsersFileBytes(
		util.GeneratePostgreSQLMD5Password(crv1.PGUserPgBouncer, password))

	// update the secret
	if err := kubeapi.UpdateSecret(clientset, secret, namspace); err != nil {
		return err
	}

	// now we wait for the password to propagate to all of the pgbouncer pods in
	// the deployment
	// set up the selector for the primary pod
	selector := fmt.Sprintf("%s=true", config.LABEL_PGBOUNCER)

	// query the pods
	pods, err := kubeapi.GetPods(clientset, selector, namspace)

	if err != nil {
		return err
	}

	// iterate through each pod and see if the secret has propagated. once it
	// returns, restart the pod (i.e. deleted it)
	for _, pod := range pods.Items {
		waitForSecretPropagation(clientset, restconfig, pod, string(secret.Data["users.txt"]),
			pgBouncerSecretPropagationTimeout, pgBouncerSecretPropagationPeriod)

		// after this waiting period has passed, delete Pod. If the pod fails to
		// delete, warn but continue on
		if err := kubeapi.DeletePod(clientset, pod.Name, pod.Namespace); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

// setPostgreSQLPassword updates the pgBouncer password in the PostgreSQL
// cluster by executing into the primary Pod and changing it
func setPostgreSQLPassword(clientset *kubernetes.Clientset, restconfig *rest.Config, pod *v1.Pod, password string) error {
	log.Debug("set pgbouncer password in PostgreSQL")

	// we use the PostgreSQL "md5" hashing mechanism here to pre-hash the
	// password. This is hard coded, which will make moving up to SCRAM ever more
	// so lovely, but at least it's better than sending it around as plaintext
	sqlpgBouncerPassword := util.GeneratePostgreSQLMD5Password(crv1.PGUserPgBouncer, password)

	if err := util.SetPostgreSQLPassword(clientset, restconfig, pod, crv1.PGUserPgBouncer, sqlpgBouncerPassword, sqlEnableLogin); err != nil {
		log.Error(err)
		return err
	}

	// and that's all!
	return nil
}

// uninstallPgBouncer uninstalls the "pgbouncer" user and other management
// objects from the PostgreSQL pod
func uninstallPgBouncer(clientset *kubernetes.Clientset, restconfig *rest.Config, pod *v1.Pod) error {
	// get the list of databases that we need to scan through
	databases, err := getPgBouncerDatabases(clientset, restconfig, pod)

	if err != nil {
		return err
	}

	// iterate through the list of databases that are returned, and execute the
	// uninstallation script
	for databases.Scan() {
		databaseName := strings.TrimSpace(databases.Text())
		execPgBouncerScript(clientset, restconfig, pod, databaseName, pgBouncerUninstallScript)
	}

	// lastly, delete the "pgbouncer" role from the PostgreSQL database
	// This is safe from SQL injection as we are using constants and a well defined
	// string
	sql := strings.NewReader(sqlUninstallPgBouncer)
	cmd := []string{"psql"}

	// exec into the pod to run the query
	_, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		cmd, "database", pod.Name, pod.ObjectMeta.Namespace, sql)

	// if there is an error executing the command, log the error message from
	// stderr and return the error
	if err != nil {
		log.Error(stderr)
		return err
	}

	return nil
}

// waitForSecretPropagation waits until the update to the pgbouncer secret has
// propogated
func waitForSecretPropagation(clientset *kubernetes.Clientset, restconfig *rest.Config, pod v1.Pod, expected string, timeoutSecs, periodSecs time.Duration) {
	timeout := time.After(timeoutSecs * time.Second)
	tick := time.Tick(periodSecs * time.Second)

loop:
	for {
		select {
		// in the case of hitting the timeout, warn that the timeout was hit, but
		// continue onward
		case <-timeout:
			log.Warnf("timed out after [%d]s waiting for secret to propogate to pod [%s]", timeout, pod.Name)
			return
		case <-tick:
			// exec into the pod to run the query
			stdout, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
				cmdViewPgBouncerUsersSecret, "pgbouncer", pod.Name, pod.ObjectMeta.Namespace, nil)

			// if there is an error, warn about it, but try again
			if err != nil {
				log.Warnf(stderr)
				continue loop
			}

			// trim any space that may be there for an accurate read
			secret := strings.TrimSpace(stdout)

			// if we have match, break the loop
			if secret == expected {
				log.Debug("secret propogated to pod [%s]", pod.Name)
				return
			}
		}
	}
}
