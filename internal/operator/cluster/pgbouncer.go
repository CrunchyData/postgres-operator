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
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type pgBouncerTemplateFields struct {
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
	Replicas                  int32 `json:",string"`
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
func AddPgbouncer(clientset kubernetes.Interface, restclient *rest.RESTClient, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	log.Debugf("adding a pgbouncer")

	// get the primary pod, which is needed to update the password for the
	// pgBouncer administrative user
	pod, err := util.GetPrimaryPod(clientset, cluster)

	if err != nil {
		return err
	}

	// check to see if pgBoncer is "installed" in the PostgreSQL cluster. This
	// means checking to see if there is a pgbouncer user, effetively
	if installed, err := checkPgBouncerInstall(clientset, restconfig, pod, cluster.Spec.Port); err != nil {
		return err
	} else if !installed {
		// this can't be installed if this is a standby, so abort if that's the case
		if cluster.Spec.Standby {
			return ErrStandbyNotAllowed
		}

		if err := installPgBouncer(clientset, restconfig, pod, cluster.Spec.Port); err != nil {
			return err
		}
	}

	// only attempt to set the password if the cluster is not in standby mode
	// and the secret does not already exist. If GetPasswordFromSecret returns
	// no errors, then we can assume that the Secret does not exist
	if !cluster.Spec.Standby {
		secretName := util.GeneratePgBouncerSecretName(cluster.Name)
		pgBouncerPassword, err := util.GetPasswordFromSecret(clientset, cluster.Namespace, secretName)

		if err != nil {
			// set the password that will be used for the "pgbouncer" PostgreSQL account
			newPassword, err := generatePassword()

			if err != nil {
				return err
			}

			pgBouncerPassword = newPassword

			// create the secret that pgbouncer will include the pgBouncer
			// credentials
			if err := createPgbouncerSecret(clientset, cluster, pgBouncerPassword); err != nil {
				return err
			}
		}

		// attempt to update the password in PostgreSQL, as this is how pgBouncer
		// will properly interface with PostgreSQL
		if err := setPostgreSQLPassword(clientset, restconfig, pod, cluster.Spec.Port, pgBouncerPassword); err != nil {
			return err
		}
	}

	// next, create the pgBouncer deployment
	if err := createPgBouncerDeployment(clientset, cluster); err != nil {
		return err
	}

	// finally, try to create the pgBouncer service
	if err := createPgBouncerService(clientset, cluster); err != nil {
		return err
	}

	log.Debugf("added pgbouncer to cluster [%s]", cluster.Name)

	// publish an event
	publishPgBouncerEvent(events.EventCreatePgbouncer, cluster)

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
func DeletePgbouncer(clientset kubernetes.Interface, restclient *rest.RESTClient, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	clusterName := cluster.Name
	namespace := cluster.Namespace

	log.Debugf("delete pgbouncer from cluster [%s] in namespace [%s]", clusterName, namespace)

	// if this is a standby cluster, we cannot execute any of the SQL on the
	// PostgreSQL server, but we can still remove the Deployment and Service.
	if !cluster.Spec.Standby {
		if err := disablePgBouncer(clientset, restconfig, cluster); err != nil {
			return err
		}
	}

	// next, delete the various Kubernetes objects associated with the pgbouncer
	// these include the Service, Deployment, Secret and ConfigMap associated with
	// pgbouncer
	// If these fail, we'll just pass through
	//
	// First, delete the Service and Deployment, which share the same naem
	pgbouncerDeploymentName := fmt.Sprintf(pgBouncerDeploymentFormat, clusterName)

	if err := clientset.CoreV1().Services(namespace).Delete(pgbouncerDeploymentName, &metav1.DeleteOptions{}); err != nil {
		log.Warn(err)
	}

	deletePropagation := metav1.DeletePropagationForeground
	if err := clientset.AppsV1().Deployments(namespace).Delete(pgbouncerDeploymentName, &metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	}); err != nil {
		log.Warn(err)
	}

	// remove the secret. again, if this fails, just log the error and apss
	// through
	secretName := util.GeneratePgBouncerSecretName(clusterName)

	if err := clientset.CoreV1().Secrets(namespace).Delete(secretName, &metav1.DeleteOptions{}); err != nil {
		log.Warn(err)
	}

	// publish an event
	publishPgBouncerEvent(events.EventDeletePgbouncer, cluster)

	return nil
}

// RotatePgBouncerPassword rotates the password for a pgBouncer PostgreSQL user,
// which involves updating the password in the PostgreSQL cluster as well as
// the users secret that is available in the pgbouncer Pod
func RotatePgBouncerPassword(clientset kubernetes.Interface, restclient *rest.RESTClient, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	// determine if we are able to access the primary Pod
	primaryPod, err := util.GetPrimaryPod(clientset, cluster)

	if err != nil {
		return err
	}

	// let's also go ahead and get the secret that contains the pgBouncer
	// information. If we can't find the secret, we're basically done here
	secretName := util.GeneratePgBouncerSecretName(cluster.Name)
	secret, err := clientset.CoreV1().Secrets(cluster.Namespace).Get(secretName, metav1.GetOptions{})

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
	password, err := generatePassword()

	if err != nil {
		return err
	}

	// next, update the PostgreSQL primary with the new password. If this fails
	// we definitely return an error
	if err := setPostgreSQLPassword(clientset, restconfig, primaryPod, cluster.Spec.Port, password); err != nil {
		return err
	}

	// next, update the users.txt and password fields of the secret. the important
	// one to update is the users.txt, as that is used by pgbouncer to connect to
	// PostgreSQL to perform its authentication
	secret.Data["password"] = []byte(password)
	secret.Data["users.txt"] = util.GeneratePgBouncerUsersFileBytes(
		makePostgresPassword(pgpassword.MD5, password))

	// update the secret
	if _, err := clientset.CoreV1().Secrets(cluster.Namespace).Update(secret); err != nil {
		return err
	}

	// force the password to propagate to all of the pgbouncer pods in
	// the deployment
	selector := fmt.Sprintf("%s=%s,%s=true", config.LABEL_PG_CLUSTER, cluster.Name,
		config.LABEL_PGBOUNCER)

	// query the pods
	pods, err := clientset.CoreV1().Pods(cluster.Namespace).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		if err := clientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{}); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

// UninstallPgBouncer uninstalls the "pgbouncer" user and other management
// objects from the PostgreSQL cluster
func UninstallPgBouncer(clientset kubernetes.Interface, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	// if this is a standby cluster, exit and return an error
	if cluster.Spec.Standby {
		return ErrStandbyNotAllowed
	}

	// determine if we are able to access the primary Pod. If not, then the
	// journey ends right here
	pod, err := util.GetPrimaryPod(clientset, cluster)

	if err != nil {
		return err
	}

	// get the list of databases that we need to scan through
	databases, err := getPgBouncerDatabases(clientset, restconfig, pod, cluster.Spec.Port)

	if err != nil {
		return err
	}

	// iterate through the list of databases that are returned, and execute the
	// uninstallation script
	for databases.Scan() {
		databaseName := strings.TrimSpace(databases.Text())
		execPgBouncerScript(clientset, restconfig, pod, cluster.Spec.Port, databaseName, pgBouncerUninstallScript)
	}

	// lastly, delete the "pgbouncer" role from the PostgreSQL database
	// This is safe from SQL injection as we are using constants and a well defined
	// string
	sql := strings.NewReader(sqlUninstallPgBouncer)
	cmd := []string{"psql", "-p", cluster.Spec.Port}

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

// UpdatePgbouncer contains the various functions that are used to perform
// updates to the pgBouncer deployment for a cluster, such as rotating a
// password
//
// Any errors that are returned should be logged in the calling function, though
// some logging occurs in this function as well
func UpdatePgbouncer(clientset kubernetes.Interface, restclient *rest.RESTClient, oldCluster, newCluster *crv1.Pgcluster) error {
	clusterName := newCluster.Name
	namespace := newCluster.Namespace

	log.Debugf("update pgbouncer from cluster [%s] in namespace [%s]", clusterName, namespace)

	// we need to detect what has changed. presently, two "groups" of things could
	// have changed
	// 1. The # of replicas to maintain
	// 2. The pgBouncer container resources
	//
	// As #2 is a bit more destructive, we'll do that last

	// check if the replicas differ
	if oldCluster.Spec.PgBouncer.Replicas != newCluster.Spec.PgBouncer.Replicas {
		if err := updatePgBouncerReplicas(clientset, restclient, newCluster); err != nil {
			return err
		}
	}

	// check if the resources differ
	if !reflect.DeepEqual(oldCluster.Spec.PgBouncer.Resources, newCluster.Spec.PgBouncer.Resources) ||
		!reflect.DeepEqual(oldCluster.Spec.PgBouncer.Limits, newCluster.Spec.PgBouncer.Limits) {
		if err := updatePgBouncerResources(clientset, restclient, newCluster); err != nil {
			return err
		}
	}

	// publish an event
	publishPgBouncerEvent(events.EventUpdatePgbouncer, newCluster)

	// and that's it!
	return nil
}

// checkPgBouncerInstall checks to see if pgBouncer is installed in the
// PostgreSQL custer, which involves check to see if the pgBouncer role is
// present in the PostgreSQL cluster
func checkPgBouncerInstall(clientset kubernetes.Interface, restconfig *rest.Config, pod *v1.Pod, port string) (bool, error) {
	// set up the SQL
	sql := strings.NewReader(sqlCheckPgBouncerInstall)

	// have the command return an unaligned string of just the "t" or "f"
	cmd := []string{"psql", "-A", "-t", "-p", port}

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
func createPgBouncerDeployment(clientset kubernetes.Interface, cluster *crv1.Pgcluster) error {
	log.Debugf("creating pgbouncer deployment: %s", cluster.Name)

	// derive the name of the Deployment...which is also used as the name of the
	// service
	pgbouncerDeploymentName := fmt.Sprintf(pgBouncerDeploymentFormat, cluster.Name)

	// get the fields that will be substituted in the pgBouncer template
	fields := pgBouncerTemplateFields{
		Name:            pgbouncerDeploymentName,
		ClusterName:     cluster.Name,
		CCPImagePrefix:  util.GetValueOrDefault(cluster.Spec.CCPImagePrefix, operator.Pgo.Cluster.CCPImagePrefix),
		CCPImageTag:     util.GetStandardImageTag(cluster.Spec.CCPImage, cluster.Spec.CCPImageTag),
		Port:            cluster.Spec.Port,
		PGBouncerSecret: util.GeneratePgBouncerSecretName(cluster.Name),
		ContainerResources: operator.GetResourcesJSON(cluster.Spec.PgBouncer.Resources,
			cluster.Spec.PgBouncer.Limits),
		PodAntiAffinity: operator.GetPodAntiAffinity(cluster,
			crv1.PodAntiAffinityDeploymentPgBouncer, cluster.Spec.PodAntiAffinity.PgBouncer),
		PodAntiAffinityLabelName: config.LABEL_POD_ANTI_AFFINITY,
		PodAntiAffinityLabelValue: string(operator.GetPodAntiAffinityType(cluster,
			crv1.PodAntiAffinityDeploymentPgBouncer, cluster.Spec.PodAntiAffinity.PgBouncer)),
		Replicas: cluster.Spec.PgBouncer.Replicas,
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

	if _, err := clientset.AppsV1().Deployments(cluster.Namespace).Create(&deployment); err != nil {
		return err
	}

	return nil
}

// createPgbouncerSecret create a secret used by pgbouncer. Returns the
// plaintext password and/or an error
func createPgbouncerSecret(clientset kubernetes.Interface, cluster *crv1.Pgcluster, password string) error {
	secretName := util.GeneratePgBouncerSecretName(cluster.Name)

	// see if this secret already exists...if it does, then take an early exit
	if _, err := util.GetPasswordFromSecret(clientset, cluster.Namespace, secretName); err == nil {
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
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
			Labels: map[string]string{
				config.LABEL_PG_CLUSTER: cluster.Name,
				config.LABEL_PGBOUNCER:  "true",
				config.LABEL_VENDOR:     config.LABEL_CRUNCHY,
			},
		},
		Data: map[string][]byte{
			"password":      []byte(password),
			"pgbouncer.ini": pgBouncerConf,
			"pg_hba.conf":   pgbouncerHBA,
			"users.txt": util.GeneratePgBouncerUsersFileBytes(
				makePostgresPassword(pgpassword.MD5, password)),
		},
	}

	if _, err := clientset.CoreV1().Secrets(cluster.Namespace).Create(&secret); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// createPgBouncerService creates the Kubernetes Service for pgBouncer
func createPgBouncerService(clientset kubernetes.Interface, cluster *crv1.Pgcluster) error {
	// pgBouncerServiceName is the name of the Service of the pgBouncer, which
	// matches that for the Deploymnt
	pgBouncerServiceName := fmt.Sprintf(pgBouncerDeploymentFormat, cluster.Name)

	// set up the service template fields
	fields := ServiceTemplateFields{
		Name:        pgBouncerServiceName,
		ServiceName: pgBouncerServiceName,
		ClusterName: cluster.Name,
		// TODO: I think "port" needs to be evaluated, but I think for now using
		// the standard PostgreSQL port works
		Port: operator.Pgo.Cluster.Port,
	}

	if err := CreateService(clientset, &fields, cluster.Namespace); err != nil {
		return err
	}

	return nil
}

// disablePgBouncer executes codes on the primary PostgreSQL pod in order to
// disable the "pgbouncer" role from being able to log in. It keeps the
// artificats that were created during normal pgBouncer operation
func disablePgBouncer(clientset kubernetes.Interface, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	log.Debugf("disable pgbouncer user on cluster [%s]", cluster.Name)
	// disable the pgbouncer user in the PostgreSQL cluster.
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

	return nil
}

// execPgBouncerScript runs a script pertaining to the management of pgBouncer
// on the PostgreSQL pod
func execPgBouncerScript(clientset kubernetes.Interface, restconfig *rest.Config, pod *v1.Pod, port, databaseName, script string) {
	cmd := []string{"psql", "-p", port, databaseName, "-f", script}

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
func generatePassword() (string, error) {
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
		PG_PRIMARY_SERVICE_NAME: cluster.Name,
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
		Name:       fmt.Sprintf("%s-%s", taskLabel, cluster.Name),
		TaskType:   taskType,
		Parameters: parameters,
	}

	// create the pgtask object
	task := &crv1.Pgtask{
		ObjectMeta: metav1.ObjectMeta{
			Name: spec.Name,
			Labels: map[string]string{
				config.LABEL_PG_CLUSTER: cluster.Name,
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
func getPgBouncerDatabases(clientset kubernetes.Interface, restconfig *rest.Config, pod *v1.Pod, port string) (*bufio.Scanner, error) {
	// so the way this works is that there needs to be a special SQL installation
	// script that is executed on every database EXCEPT for postgres and template0
	// but is executed on template1
	sql := strings.NewReader(sqlGetDatabasesForPgBouncer)

	// have the command return an unaligned string of just the "t" or "f"
	cmd := []string{"psql", "-A", "-t", "-p", port}

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

// getPgBouncerDeployment finds the pgBouncer deployment for a PostgreSQL
// cluster
func getPgBouncerDeployment(clientset kubernetes.Interface, cluster *crv1.Pgcluster) (*appsv1.Deployment, error) {
	log.Debugf("find pgbouncer for: %s", cluster.Name)

	// derive the name of the Deployment...which is also used as the name of the
	// service
	pgbouncerDeploymentName := fmt.Sprintf(pgBouncerDeploymentFormat, cluster.Name)

	deployment, err := clientset.AppsV1().Deployments(cluster.Namespace).Get(pgbouncerDeploymentName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return deployment, nil
}

// installPgBouncer installs the "pgbouncer" user and other management objects
// into the PostgreSQL pod
func installPgBouncer(clientset kubernetes.Interface, restconfig *rest.Config, pod *v1.Pod, port string) error {
	// get the list of databases that we need to scan through
	databases, err := getPgBouncerDatabases(clientset, restconfig, pod, port)

	if err != nil {
		return err
	}

	// iterate through the list of databases that are returned, and execute the
	// installation script
	for databases.Scan() {
		databaseName := strings.TrimSpace(databases.Text())

		execPgBouncerScript(clientset, restconfig, pod, port, databaseName, pgBouncerInstallScript)
	}

	return nil
}

// makePostgresPassword creates the expected hash for a password type for a
// PostgreSQL password
func makePostgresPassword(passwordType pgpassword.PasswordType, password string) string {
	// get the PostgreSQL password generate based on the password type
	// as all of these values are valid, this not not error
	postgresPassword, _ := pgpassword.NewPostgresPassword(passwordType, crv1.PGUserPgBouncer, password)

	// create the PostgreSQL style hashed password and return
	hashedPassword, _ := postgresPassword.Build()

	return hashedPassword
}

// publishPgBouncerEvent publishes one of the events on the event stream
func publishPgBouncerEvent(eventType string, cluster *crv1.Pgcluster) {
	var event events.EventInterface

	// prepare the topics to publish to
	topics := []string{events.EventTopicPgbouncer}
	// set up the event header
	eventHeader := events.EventHeader{
		Namespace: cluster.Namespace,
		Topic:     topics,
		Timestamp: time.Now(),
		EventType: eventType,
	}
	clusterName := cluster.Name

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

// setPostgreSQLPassword updates the pgBouncer password in the PostgreSQL
// cluster by executing into the primary Pod and changing it
func setPostgreSQLPassword(clientset kubernetes.Interface, restconfig *rest.Config, pod *v1.Pod, port, password string) error {
	log.Debug("set pgbouncer password in PostgreSQL")

	// we use the PostgreSQL "md5" hashing mechanism here to pre-hash the
	// password. This is semi-hard coded but is now prepped for SCRAM as a
	// password type can be passed in. Almost to SCRAM!
	sqlpgBouncerPassword := makePostgresPassword(pgpassword.MD5, password)

	if err := util.SetPostgreSQLPassword(clientset, restconfig, pod,
		port, crv1.PGUserPgBouncer, sqlpgBouncerPassword, sqlEnableLogin); err != nil {
		log.Error(err)
		return err
	}

	// and that's all!
	return nil
}

// updatePgBouncerReplicas updates the pgBouncer Deployment with the number
// of replicas (Pods) that it should run. Presently, this is fairly naive, but
// as pgBouncer is "semi-stateful" we may want to improve upon this in the
// future
func updatePgBouncerReplicas(clientset kubernetes.Interface, restclient *rest.RESTClient, cluster *crv1.Pgcluster) error {
	log.Debugf("scale pgbouncer replicas to [%d]", cluster.Spec.PgBouncer.Replicas)

	// get the pgBouncer deployment so the resources can be updated
	deployment, err := getPgBouncerDeployment(clientset, cluster)

	if err != nil {
		return err
	}

	// update the number of replicas
	deployment.Spec.Replicas = &cluster.Spec.PgBouncer.Replicas

	// and update the deployment
	// update the deployment with the new values
	if _, err := clientset.AppsV1().Deployments(deployment.Namespace).Update(deployment); err != nil {
		return err
	}

	return nil
}

// updatePgBouncerResources updates the pgBouncer Deployment with the container
// resource request values that are desired
func updatePgBouncerResources(clientset kubernetes.Interface, restclient *rest.RESTClient, cluster *crv1.Pgcluster) error {
	log.Debugf("update pgbouncer resources to [%+v]", cluster.Spec.PgBouncer.Resources)

	// get the pgBouncer deployment so the resources can be updated
	deployment, err := getPgBouncerDeployment(clientset, cluster)

	if err != nil {
		return err
	}

	// the pgBouncer container is the first one, the resources can be updated
	// from it
	deployment.Spec.Template.Spec.Containers[0].Resources.Requests = cluster.Spec.PgBouncer.Resources.DeepCopy()
	deployment.Spec.Template.Spec.Containers[0].Resources.Limits = cluster.Spec.PgBouncer.Limits.DeepCopy()

	// and update the deployment
	// update the deployment with the new values
	if _, err := clientset.AppsV1().Deployments(deployment.Namespace).Update(deployment); err != nil {
		return err
	}

	return nil
}
