package cluster

import (
	"encoding/json"
	"fmt"

	"github.com/percona/percona-postgresql-operator/internal/kubeapi"
	"github.com/percona/percona-postgresql-operator/internal/operator"
	crv1 "github.com/percona/percona-postgresql-operator/pkg/apis/crunchydata.com/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

func UpdatePMMSidecar(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *appsv1.Deployment) error {
	// need to determine if we are adding or removing
	if cluster.Spec.PMM.Enabled {
		return AddPMMSidecar(cluster, cluster.Name, deployment)
	}

	RemovePMMSidecar(deployment)

	return nil
}

func AddPMMSidecar(cluster *crv1.Pgcluster, name string, deployment *appsv1.Deployment) error {
	// use the legacy template generation to make the appropriate substitutions,
	// and then get said generation to be placed into an actual Container object
	template := operator.GetPMMContainer(cluster, name)
	container := v1.Container{}

	if err := json.Unmarshal([]byte(template), &container); err != nil {
		return fmt.Errorf("error unmarshalling exporter json into Container: %w ", err)
	}

	RemovePMMSidecar(deployment)

	deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, container)

	return nil
}

func RemovePMMSidecar(deployment *appsv1.Deployment) {
	// first, find the container entry in the list of containers and remove it
	containers := []v1.Container{}
	for _, c := range deployment.Spec.Template.Spec.Containers {
		// skip if this is the PMM container
		if c.Name == pmmContainerName {
			continue
		}

		containers = append(containers, c)
	}

	deployment.Spec.Template.Spec.Containers = containers
}
