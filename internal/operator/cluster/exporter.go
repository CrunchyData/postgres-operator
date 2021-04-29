package cluster

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
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// exporterInstallScript references the embedded script that installs all of
	// the pgMonitor functions
	exporterInstallScript = "/opt/crunchy/bin/exporter/install.sh"

	// exporterServicePortName is the name used to identify the exporter port in
	// the service
	exporterServicePortName = "postgres-exporter"
)

// AddExporter ensures that a PostgreSQL cluster is able to undertake the
// actions required by the "crunchy-postgres-exporter", i.e.
//
//   - enable a service port so scrapers can access the metrics
//   - it can authenticate as the "ccp_monitoring" user; manages the Secret as
//     well
//   - all of the monitoring views and functions are available
func AddExporter(clientset kubernetes.Interface, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	ctx := context.TODO()

	// even if this is a standby, we can still set up a Secret (though the password
	// value of the Secret is of limited use when the standby is promoted, it can
	// be rotated, similar to the pgBouncer password)

	// only create a password Secret if one does not already exist, which is
	// handled in the delegated function
	password, err := CreateExporterSecret(clientset, cluster)
	if err != nil {
		return err
	}

	// set up the Services, which are still needed on a standby
	services, err := getClusterInstanceServices(clientset, cluster)
	if err != nil {
		return err
	}

	// loop over each service to perform the necessary modifications
svcLoop:
	for i := range services.Items {
		svc := &services.Items[i]

		// loop over the service ports to see if exporter port is already set up. if
		// it is, we can continue and skip the outerloop
		for _, svcPort := range svc.Spec.Ports {
			if svcPort.Name == exporterServicePortName {
				continue svcLoop
			}
		}

		// otherwise, we need to append a service port to the list
		port, err := strconv.ParseInt(
			util.GetValueOrDefault(cluster.Spec.ExporterPort, operator.Pgo.Cluster.ExporterPort), 10, 32)
		// if we can't parse this for whatever reason, issue a warning and continue on
		if err != nil {
			log.Warn(err)
		}

		svcPort := v1.ServicePort{
			Name:     exporterServicePortName,
			Protocol: v1.ProtocolTCP,
			Port:     int32(port),
		}

		svc.Spec.Ports = append(svc.Spec.Ports, svcPort)

		// if we fail to update the service, warn, but continue on
		if _, err := clientset.CoreV1().Services(svc.Namespace).Update(ctx, svc, metav1.UpdateOptions{}); err != nil {
			log.Warn(err)
		}
	}

	// this can't be installed if this is a standby, so abort if that's the case
	if cluster.Spec.Standby {
		return ErrStandbyNotAllowed
	}

	// get the primary pod, which is needed to update the password for the
	// exporter user
	pod, err := util.GetPrimaryPod(clientset, cluster)
	if err != nil {
		return err
	}

	// add the monitoring user and all the views associated with this
	// user. this can be done by executing a script on the container itself
	cmd := []string{"/bin/bash", exporterInstallScript}

	if _, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		cmd, "database", pod.Name, pod.ObjectMeta.Namespace, nil); err != nil {
		return fmt.Errorf(stderr)
	}

	// attempt to update the password in PostgreSQL, as this is how the exporter
	// will properly interface with PostgreSQL
	return setPostgreSQLPassword(clientset, restconfig, pod, cluster.Spec.Port, crv1.PGUserMonitor, password)
}

// CreateExporterSecret create a secret used by the exporter containing the
// user credientials. Sees if a Secret already exists and if it does, uses that.
// Otherwise, it will generate the password. Returns an error if it fails.
func CreateExporterSecret(clientset kubernetes.Interface, cluster *crv1.Pgcluster) (string, error) {
	ctx := context.TODO()
	secretName := util.GenerateExporterSecretName(cluster.Name)

	// see if this secret already exists...if it does, then take an early exit
	if password, err := util.GetPasswordFromSecret(clientset, cluster.Namespace, secretName); err == nil {
		log.Infof("exporter secret %s already present, will reuse", secretName)
		return password, nil
	}

	// well, we have to generate the password
	password, err := generatePassword()
	if err != nil {
		return "", err
	}

	// the remainder of this is generating the various entries in the pgbouncer
	// secret, i.e. substituting values into templates files that contain:
	// - the pgbouncer user password
	// - the pgbouncer "users.txt" file that contains the credentials for the
	// "pgbouncer" user

	// now, we can do what we came here to do, which is create the secret
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
			Labels: map[string]string{
				config.LABEL_EXPORTER:   config.LABEL_TRUE,
				config.LABEL_PG_CLUSTER: cluster.Name,
				config.LABEL_VENDOR:     config.LABEL_CRUNCHY,
			},
		},
		Data: map[string][]byte{
			"username": []byte(crv1.PGUserMonitor),
			"password": []byte(password),
		},
	}

	for k, v := range util.GetCustomLabels(cluster) {
		secret.ObjectMeta.Labels[k] = v
	}

	if _, err := clientset.CoreV1().Secrets(cluster.Namespace).
		Create(ctx, &secret, metav1.CreateOptions{}); err != nil {
		log.Error(err)
		return "", err
	}

	return password, nil
}

// RemoveExporter disables the ability for a PostgreSQL cluster to use the
// exporter functionality. In particular this function:
//
//   - disallows the login of the monitoring user (ccp_monitoring)
//   - removes the Secret that contains the ccp_monitoring user credentials
//   - removes the port on the cluster Service
//
// This does not modify the Deployment that has the exporter sidecar. That is
// handled by the "UpdateExporter" function, so it can be handled as part of a
// rolling update
func RemoveExporter(clientset kubernetes.Interface, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	ctx := context.TODO()

	// close the exporter port on each service
	services, err := getClusterInstanceServices(clientset, cluster)
	if err != nil {
		return err
	}

	for i := range services.Items {
		svc := &services.Items[i]
		svcPorts := []v1.ServicePort{}

		for _, svcPort := range svc.Spec.Ports {
			// if we find the service port for the exporter, skip it in the loop, but
			// as we will not be including it in the update
			if svcPort.Name == exporterServicePortName {
				continue
			}

			svcPorts = append(svcPorts, svcPort)
		}

		svc.Spec.Ports = svcPorts

		// if we fail to update the service, warn but continue
		if _, err := clientset.CoreV1().Services(svc.Namespace).Update(ctx, svc, metav1.UpdateOptions{}); err != nil {
			log.Warn(err)
		}
	}

	// disable the user before clearing the Secret, so there does not end up being
	// a race condition between the existence of the Secret and the Pod definition
	// if this is a standby cluster, return as we cannot execute any SQL
	if !cluster.Spec.Standby {
		// if this fails, warn and continue
		if err := disablePostgreSQLLogin(clientset, restconfig, cluster, crv1.PGUserMonitor); err != nil {
			log.Warn(err)
		}
	}

	// delete the Secret. If there is an error deleting the Secret, log as info
	// and continue on
	if err := clientset.CoreV1().Secrets(cluster.Namespace).Delete(ctx,
		util.GenerateExporterSecretName(cluster.Name), metav1.DeleteOptions{}); err != nil {
		log.Warnf("could not remove exporter secret: %q", err.Error())
	}

	return nil
}

// RotateExporterPassword rotates the password for the monitoring PostgreSQL
// user
func RotateExporterPassword(clientset kubernetes.Interface, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	ctx := context.TODO()

	// let's also go ahead and get the secret that contains the pgBouncer
	// information. If we can't find the secret, we're basically done here
	secretName := util.GenerateExporterSecretName(cluster.Name)
	secret, err := clientset.CoreV1().Secrets(cluster.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// update the password on the PostgreSQL instance
	password, err := rotatePostgreSQLPassword(clientset, restconfig, cluster, crv1.PGUserMonitor)
	if err != nil {
		return err
	}

	// next, update the password field of the secret.
	secret.Data["password"] = []byte(password)

	// update the secret
	if _, err := clientset.CoreV1().Secrets(cluster.Namespace).
		Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return err
	}

	// and that's it - the changes will be propagated to the exporter sidecars
	return nil
}

// UpdateExporterSidecar either adds or emoves the metrics sidcar from the
// cluster. This is meant to be used as a rolling update callback function
func UpdateExporterSidecar(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *appsv1.Deployment) error {
	// need to determine if we are adding or removing
	if cluster.Spec.Exporter {
		return addExporterSidecar(cluster, deployment)
	}

	removeExporterSidecar(deployment)

	return nil
}

// addExporterSidecar adds the metrics collection exporter to a Deployment
// This does two things:
// - adds the exporter container to the manifest. If the exporter manifest
//   already exists, this supersedes it.
// - adds the exporter label to the label template, so it can be discovered that
//   this container has an exporter
func addExporterSidecar(cluster *crv1.Pgcluster, deployment *appsv1.Deployment) error {
	// use the legacy template generation to make the appropriate substitutions,
	// and then get said generation to be placed into an actual Container object
	template := operator.GetExporterAddon(cluster.Spec)

	container := v1.Container{}

	if err := json.Unmarshal([]byte(template), &container); err != nil {
		return fmt.Errorf("error unmarshalling exporter json into Container: %w ", err)
	}

	// append the container to the deployment container list. However, we are
	// going to do this carefully, in case the exporter container already exists.
	// this definition will supersede any exporter container already in the
	// containers list
	containers := []v1.Container{}
	for _, c := range deployment.Spec.Template.Spec.Containers {
		// skip if this is the exporter container
		if c.Name == exporterContainerName {
			continue
		}

		containers = append(containers, c)
	}

	// add the exporter container and override the containers list definition
	containers = append(containers, container)
	deployment.Spec.Template.Spec.Containers = containers

	// add the label to the deployment template
	deployment.Spec.Template.ObjectMeta.Labels[config.LABEL_EXPORTER] = config.LABEL_TRUE

	return nil
}

// removeExporterSidecar removes the metrics collection exporter to a
// Deployment.
//
// This involves:
//  - Removing the container entry for the exporter
//  - Removing the label from the deployment template
func removeExporterSidecar(deployment *appsv1.Deployment) {
	// first, find the container entry in the list of containers and remove it
	containers := []v1.Container{}
	for _, c := range deployment.Spec.Template.Spec.Containers {
		// skip if this is the exporter container
		if c.Name == exporterContainerName {
			continue
		}

		containers = append(containers, c)
	}

	deployment.Spec.Template.Spec.Containers = containers

	// alright, so this moves towards the mix of modern/legacy behavior, but we
	// need to scan the environmental variables on the "database" container and
	// remove one with the name "PGMONITOR_PASSWORD"
	for i, c := range deployment.Spec.Template.Spec.Containers {
		if c.Name == "database" {
			env := []v1.EnvVar{}

			for _, e := range c.Env {
				if e.Name == "PGMONITOR_PASSWORD" {
					continue
				}

				env = append(env, e)
			}

			deployment.Spec.Template.Spec.Containers[i].Env = env
			break
		}
	}

	// finally, remove the label
	delete(deployment.Spec.Template.ObjectMeta.Labels, config.LABEL_EXPORTER)
}
