package dfservice

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
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
)

// pvcContainerName contains the name of the container that the PVCs are mounted
// to, which, curiously, is "database" for all of them
const pvcContainerName = "database"

var (
	pvcSizePattern = regexp.MustCompile("^([0-9]+)")
)

func DfCluster(request msgs.DfRequest) msgs.DfResponse {
	response := msgs.DfResponse{}
	// set the namespace
	namespace := request.Namespace
	// set up the selector
	selector := ""
	// if the selector is not set to "*", then set it to the value that is in the
	// Selector paramater
	if request.Selector != msgs.DfShowAllSelector {
		selector = request.Selector
	}

	log.Debugf("df selector is [%s]", selector)

	// get all of the clusters that match the selector
	clusterList := crv1.PgclusterList{}

	//get a list of matching clusters
	if err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, selector, namespace); err != nil {
		return CreateErrorResponse(err.Error())
	}

	log.Debugf("df clusters found len is %d", len(clusterList.Items))

	// iterate through each cluster and get the information about the disk
	// utilization. As there could be a lot of clusters doing this, we opt for
	// concurrency, but have a way to escape if one of the clusters has an error
	// response
	clusterResultsChannel := make(chan msgs.DfDetail)
	errorChannel := make(chan error)
	clusterProgressChannel := make(chan bool)

	for _, c := range clusterList.Items {
		// first, to properly handle the goroutine, declare a new variable here
		cluster := c
		// now, go get the disk capacity information about the cluster
		go getClusterDf(&cluster, clusterResultsChannel, clusterProgressChannel, errorChannel)
	}

	// track the progress / completion, so we know when to exit
	processed := 0
	totalClusters := len(clusterList.Items)

loop:
	for {
		select {
		// if a result comes through, append it to the list
		case result := <-clusterResultsChannel:
			response.Results = append(response.Results, result)
			// if an error comes through, immeidately abort
		case err := <-errorChannel:
			return CreateErrorResponse(err.Error())
			// and if we have finished, then break the loop
		case <-clusterProgressChannel:
			processed++

			log.Debugf("df [%s] progress: [%d/%d]", selector, processed, totalClusters)

			if processed == totalClusters {
				break loop
			}
		}
	}

	// lastly, set the response as being OK
	response.Status = msgs.Status{Code: msgs.Ok}

	return response
}

// getClaimCapacity makes a call to the PVC API to get the total capacity
// available on the PVC
func getClaimCapacity(clientset *kubernetes.Clientset, pvcName, ns string) (string, error) {
	log.Debugf("in df pvc name found to be %s", pvcName)

	pvc, _, err := kubeapi.GetPVC(clientset, pvcName, ns)

	if err != nil {
		log.Error(err)
		return "", err
	}

	qty := pvc.Status.Capacity[v1.ResourceStorage]

	log.Debugf("storage cap string value %s", qty.String())

	return qty.String(), err
}

// getClusterDf breaks out the tasks for getting all the capacity information
// about a PostgreSQL cluster so it can be performed on each relevant instance
// (Pod)
//
// we use pointers to keep the argument size down and because we are not
// modifying any of the content
func getClusterDf(cluster *crv1.Pgcluster, clusterResultsChannel chan msgs.DfDetail, clusterProgressChannel chan bool, errorChannel chan error) {
	log.Debugf("pod df: %s", cluster.Spec.Name)

	selector := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, cluster.Spec.Name)

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, cluster.Spec.Namespace)

	// if there is an error attempting to get the pods, just return
	if err != nil {
		errorChannel <- err
		return
	}

	// set up channels for collecting the results that will be sent to the user
	podResultsChannel := make(chan msgs.DfDetail)
	podProgressChannel := make(chan bool)

	// figure out how many pods will need to be checked, as this will be the
	// "completed" number
	totalPods := 0

	for _, p := range pods.Items {
		// to properly handle the goroutine that is coming up, we first declare a
		// new variable
		pod := p

		// get the map of labels for convenience
		podLabels := pod.ObjectMeta.GetLabels()

		// if this is neither a PostgreSQL or pgBackRest pod, skip
		// we can cheat a little bit and check that the HA label is present, or
		// the pgBackRest repo pod label
		if podLabels[config.LABEL_PGHA_ROLE] == "" && podLabels[config.LABEL_PGO_BACKREST_REPO] == "" {
			continue
		}

		// at this point, we can include this pod in the total pods
		totalPods++

		// now, we can spin up goroutines to get the information and results from
		// the rest of the pods
		go getPodDf(cluster, &pod, podResultsChannel, podProgressChannel, errorChannel)
	}

	// track how many pods have been processed
	processed := 0

loop:
	for {
		select {
		// if a result is found, immediately put onto the cluster results channel
		case result := <-podResultsChannel:
			log.Debug(result)
			clusterResultsChannel <- result
			// if a pod is fully processed, increment the processed counter and
			// determine if we have finished and can break the loop
		case <-podProgressChannel:
			processed++
			log.Debugf("df cluster [%s] pod progress: [%d/%d]", cluster.Spec.Name, processed, totalPods)
			if processed == totalPods {
				break loop
			}
		}
	}

	// if we are finished with this cluster, indicate we are done
	clusterProgressChannel <- true
}

// getPodDf performs the heavy lifting of getting the total capacity values for
// the PostgreSQL cluster by introspecting each Pod, which requires a few API
// calls. This function is optimized to return concurrenetly, though has an
// escape if an error is reached by reusing the error channel from the main Df
// function
//
// we use pointers to keep the argument size down and because we are not
// modifying any of the content
func getPodDf(cluster *crv1.Pgcluster, pod *v1.Pod, podResultsChannel chan msgs.DfDetail, podProgressChannel chan bool, errorChannel chan error) {
	podLabels := pod.ObjectMeta.GetLabels()
	// at this point, we can get the instance name, which is conveniently
	// available from the deployment label
	//
	/// ...this is a bit hacky to get the pgBackRest repo name, but it works
	instanceName := podLabels[config.LABEL_DEPLOYMENT_NAME]

	if instanceName == "" {
		log.Debug(podLabels)
		instanceName = podLabels[config.LABEL_NAME]
	}

	log.Debugf("df processing pod [%s]", instanceName)

	// now, iterate through each volume, and only continue one if this is a
	// "volume of interest"
	for _, volume := range pod.Spec.Volumes {
		// as a first check, ensure there is a PVC associated with this volume
		// if not, this is a nonstarter
		if volume.VolumeSource.PersistentVolumeClaim == nil {
			continue
		}

		// start setting up the result...there's a chance we may not need it
		// based on the next check, but it's more convenient
		result := msgs.DfDetail{
			InstanceName: instanceName, // OK to set this here, even if we continue
			PodName:      pod.ObjectMeta.Name,
		}

		// we want three types of volumes:
		// PostgreSQL data directories (pgdata)
		// PostgreSQL tablespaces (tablespace-)
		// pgBackRest repositories (backrestrepo)
		// classify by the type of volume that we want...if we don't find any of
		// them, continue one
		switch {
		case volume.Name == config.VOLUME_POSTGRESQL_DATA:
			result.PVCType = msgs.PVCTypePostgreSQL
		case volume.Name == config.VOLUME_PGBACKREST_REPO_NAME:
			result.PVCType = msgs.PVCTypepgBackRest
		case strings.HasPrefix(volume.Name, config.VOLUME_TABLESPACE_NAME_PREFIX):
			result.PVCType = msgs.PVCTypeTablespace
		default:
			continue
		}

		// get the name of the PVC
		result.PVCName = volume.VolumeSource.PersistentVolumeClaim.ClaimName

		log.Debugf("pvc found [%s]", result.PVCName)

		// next, get the size of the PVC. First have to get the correct PVC
		// mount point
		var pvcMountPoint string

		switch result.PVCType {
		case msgs.PVCTypePostgreSQL:
			pvcMountPoint = fmt.Sprintf("%s/%s", config.VOLUME_POSTGRESQL_DATA_MOUNT_PATH, result.PVCName)
		case msgs.PVCTypepgBackRest:
			pvcMountPoint = fmt.Sprintf("%s/%s", config.VOLUME_PGBACKREST_REPO_MOUNT_PATH, podLabels["Name"])
		case msgs.PVCTypeTablespace:
			// first, extract the name of the tablespace by removing the
			// VOLUME_TABLESPACE_NAME_PREFIX prefix from the volume name
			tablespaceName := strings.Replace(volume.Name, config.VOLUME_TABLESPACE_NAME_PREFIX, "", 1)
			// use that to populate the path structure for the tablespaces
			pvcMountPoint = fmt.Sprintf("%s%s/%s", config.VOLUME_TABLESPACE_PATH_PREFIX, tablespaceName, tablespaceName)
		}

		cmd := []string{"du", "-s", "--block-size", "1", pvcMountPoint}

		stdout, stderr, err := kubeapi.ExecToPodThroughAPI(apiserver.RESTConfig,
			apiserver.Clientset, cmd, pvcContainerName, pod.Name, cluster.Spec.Namespace, nil)

		// if the command fails, exit here
		if err != nil {
			err := fmt.Errorf(stderr)
			log.Error(err)
			errorChannel <- err
			return
		}

		// have to parse the size out from the statement. Size is in bytes
		pvcSizeMatches := pvcSizePattern.FindStringSubmatch(stdout)

		log.Debugf("pvc size [%s]", pvcSizeMatches)

		// ensure that the substring is matched
		if len(pvcSizeMatches) < 2 {
			msg := fmt.Sprintf("could not find the size of pvc %s", result.PVCName)
			err := errors.New(msg)
			log.Error(err)
			errorChannel <- err
			return
		}

		// get the size of the PVC...will need to be converted to an integer
		pvcSize, err := strconv.Atoi(pvcSizeMatches[1])

		if err != nil {
			log.Error(err)
			errorChannel <- err
			return
		}

		result.PVCUsed = int64(pvcSize)

		if claimSize, err := getClaimCapacity(apiserver.Clientset, result.PVCName, cluster.Spec.Namespace); err != nil {
			errorChannel <- err
			return
		} else {
			resourceClaimSize := resource.MustParse(claimSize)
			result.PVCCapacity, _ = resourceClaimSize.AsInt64()
		}

		log.Debugf("pvc info [%+v]", result)

		// put the result on the result channel
		podResultsChannel <- result
	}

	podProgressChannel <- true
}
