package cluster

/*
 Copyright 2020 - 2022 Crunchy Data Solutions, Inc.
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
	// pgBadgerServicePortName is the name used to identify the pgBadger port in
	// the service
	pgBadgerServicePortName = "pgbadger"
)

// AddPGBadger ensures that a PostgreSQL cluster is able to undertake the
// actions required by the "crunchy-badger", i.e. updating the Service.
// This executes regardless if this is a standby cluster.
//
// This does not modify the Deployment that has the pgBadger sidecar. That is
// handled by the "UpdatePGBadgerSidecar" function, so it can be handled as part
// of a rolling update.
func AddPGBadger(clientset kubernetes.Interface, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
	ctx := context.TODO()
	// set up the Services, which are still needed on a standby
	services, err := getClusterInstanceServices(clientset, cluster)
	if err != nil {
		return err
	}

	// loop over each service to perform the necessary modifications
svcLoop:
	for i := range services.Items {
		svc := &services.Items[i]

		// loop over the service ports to see if pgBadger port is already set up. if
		// it is, we can continue and skip the outerloop
		for _, svcPort := range svc.Spec.Ports {
			if svcPort.Name == pgBadgerServicePortName {
				continue svcLoop
			}
		}

		// otherwise, we need to append a service port to the list
		port, err := strconv.ParseInt(
			util.GetValueOrDefault(cluster.Spec.PGBadgerPort, operator.Pgo.Cluster.PGBadgerPort), 10, 32)
		// if we can't parse this for whatever reason, issue a warning and continue on
		if err != nil {
			log.Warn(err)
		}

		svcPort := v1.ServicePort{
			Name:     pgBadgerServicePortName,
			Protocol: v1.ProtocolTCP,
			Port:     int32(port),
		}

		svc.Spec.Ports = append(svc.Spec.Ports, svcPort)

		// if we fail to update the service, warn, but continue on
		if _, err := clientset.CoreV1().Services(svc.Namespace).Update(ctx, svc, metav1.UpdateOptions{}); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

// RemovePGBadger disables the ability for a PostgreSQL cluster to run a
// pgBadger cluster.
//
// This does not modify the Deployment that has the pgBadger sidecar. That is
// handled by the "UpdatePGBadgerSidecar" function, so it can be handled as part
// of a rolling update.
func RemovePGBadger(clientset kubernetes.Interface, restconfig *rest.Config, cluster *crv1.Pgcluster) error {
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
			// if we find the service port for the pgBadger, skip it in the loop, but
			// as we will not be including it in the update
			if svcPort.Name == pgBadgerServicePortName {
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
	return nil
}

// UpdatePGBadgerSidecar either adds or emoves the pgBadger sidcar from the
// cluster. This is meant to be used as a rolling update callback function
func UpdatePGBadgerSidecar(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *appsv1.Deployment) error {
	// need to determine if we are adding or removing
	if cluster.Spec.PGBadger {
		return addPGBadgerSidecar(cluster, deployment)
	}

	removePGBadgerSidecar(deployment)

	return nil
}

// addPGBadgerSidecar adds the pgBadger sidecar to a Deployment. If pgBadger is
// already present, this call supersedes it and adds the "new version" of the
// pgBadger container.
func addPGBadgerSidecar(cluster *crv1.Pgcluster, deployment *appsv1.Deployment) error {
	// use the legacy template generation to make the appropriate substitutions,
	// and then get said generation to be placed into an actual Container object
	template := operator.GetBadgerAddon(cluster, deployment.Name)
	container := v1.Container{}

	if err := json.Unmarshal([]byte(template), &container); err != nil {
		return fmt.Errorf("error unmarshalling exporter json into Container: %w ", err)
	}

	// append the container to the deployment container list. However, we are
	// going to do this carefully, in case the pgBadger container already exists.
	// this definition will supersede any exporter container already in the
	// containers list
	containers := []v1.Container{}
	for _, c := range deployment.Spec.Template.Spec.Containers {
		// skip if this is the pgBadger container. pgBadger is added after the loop
		if c.Name == pgBadgerContainerName {
			continue
		}

		containers = append(containers, c)
	}

	// add the pgBadger container and override the containers list definition
	containers = append(containers, container)
	deployment.Spec.Template.Spec.Containers = containers

	return nil
}

// removePGBadgerSidecar removes the pgBadger sidecar from a Deployment.
//
// This involves:
//  - Removing the container entry for pgBadger
func removePGBadgerSidecar(deployment *appsv1.Deployment) {
	// first, find the container entry in the list of containers and remove it
	containers := []v1.Container{}
	for _, c := range deployment.Spec.Template.Spec.Containers {
		// skip if this is the pgBadger container
		if c.Name == pgBadgerContainerName {
			continue
		}

		containers = append(containers, c)
	}

	deployment.Spec.Template.Spec.Containers = containers
}
