package backrestservice

/*
Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/apiserver/backupoptions"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/util"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

const containername = "database"

// pgBackRestInfoCommand is the baseline command used for getting the
// pgBackRest info
var pgBackRestInfoCommand = []string{"pgbackrest", "info", "--output", "json"}

// repoTypeFlagS3 is used for getting the pgBackRest info for a repository that
// is stored in S3
var repoTypeFlagS3 = []string{"--repo1-type", "s3"}

// noRepoS3VerifyTLS is used to disable SSL certificate verification when getting
// the pgBackRest info for a repository that is stored in S3
var noRepoS3VerifyTLS = "--no-repo1-s3-verify-tls"

//  CreateBackup ...
// pgo backup mycluster
// pgo backup --selector=name=mycluster
func CreateBackup(request *msgs.CreateBackrestBackupRequest, ns, pgouser string) msgs.CreateBackrestBackupResponse {
	resp := msgs.CreateBackrestBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	if request.BackupOpts != "" {
		err := backupoptions.ValidateBackupOpts(request.BackupOpts, request)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	clusterList := crv1.PgclusterList{}
	var err error
	if request.Selector != "" {
		//use the selector instead of an argument list to filter on
		err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, request.Selector, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		if len(clusterList.Items) == 0 {
			log.Debug("no clusters found")
			resp.Results = append(resp.Results, "no clusters found with that selector")
			return resp
		} else {
			newargs := make([]string, 0)
			for _, cluster := range clusterList.Items {
				newargs = append(newargs, cluster.Spec.Name)
			}
			request.Args = newargs
		}

	}

	// Convert the names of all pgclusters specified for the request to a pgclusterList
	if clusterList.Items == nil {
		clusterList, err = clusterNamesToPGClusterList(ns, request.Args)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	// Return an error if any clusters identified for the backup are in standby mode.  Backups
	// from standby servers are not allowed since the cluster is following a remote primary,
	// which itself is responsible for performing any backups for the cluster as required.
	if hasStandby, standbyClusters := apiserver.PGClusterListHasStandby(clusterList); hasStandby {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("Request rejected, unable to create backups for clusters "+
			"%s: %s.", strings.Join(standbyClusters, ","), apiserver.ErrStandbyNotAllowed.Error())
		return resp
	}

	for _, clusterName := range request.Args {
		log.Debugf("create backrestbackup called for %s", clusterName)
		taskName := "backrest-backup-" + clusterName

		cluster := crv1.Pgcluster{}
		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, clusterName, ns)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = clusterName + " was not found, verify cluster name"
			return resp
		} else if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		// check if the current cluster is not upgraded to the deployed
		// Operator version. If not, do not allow the command to complete
		if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = fmt.Sprintf("%s %s", cluster.Name, msgs.UpgradeError)
			return resp
		}

		if cluster.Labels[config.LABEL_BACKREST] != "true" {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = clusterName + " does not have pgbackrest enabled"
			return resp
		}

		err = util.ValidateBackrestStorageTypeOnBackupRestore(request.BackrestStorageType,
			cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], false)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		result := crv1.Pgtask{}

		// error if it already exists
		found, err = kubeapi.Getpgtask(apiserver.RESTClient, &result, taskName, ns)
		if !found {
			log.Debugf("backrest backup pgtask %s was not found so we will create it", taskName)
		} else if err != nil {

			resp.Results = append(resp.Results, "error getting pgtask for "+taskName)
			break
		} else {

			log.Debugf("pgtask %s was found so we will recreate it", taskName)
			//remove the existing pgtask
			err := kubeapi.Deletepgtask(apiserver.RESTClient, taskName, ns)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}

			//remove any previous backup job

			//selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_BACKREST + "=true"
			selector := config.LABEL_BACKREST_COMMAND + "=" + crv1.PgtaskBackrestBackup + "," + config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_BACKREST + "=true"
			deletePropagation := metav1.DeletePropagationForeground
			err = apiserver.Clientset.
				BatchV1().Jobs(ns).
				DeleteCollection(
					&metav1.DeleteOptions{PropagationPolicy: &deletePropagation},
					metav1.ListOptions{LabelSelector: selector})
			if err != nil {
				log.Error(err)
			}

			//a hack sort of due to slow propagation
			for i := 0; i < 3; i++ {
				jobList, err := apiserver.Clientset.BatchV1().Jobs(ns).List(metav1.ListOptions{LabelSelector: selector})
				if err != nil {
					log.Error(err)
				}
				if len(jobList.Items) > 0 {
					log.Debug("sleeping a bit for delete job propagation")
					time.Sleep(time.Second * 2)
				}
			}

		}

		// get pod name from cluster
		var podname string
		podname, err = getBackrestRepoPodName(&cluster, ns)

		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		// check if primary is ready
		if err := isPrimaryReady(&cluster, ns); err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		jobName := "backrest-" + crv1.PgtaskBackrestBackup + "-" + clusterName
		log.Debugf("setting jobName to %s", jobName)

		err = kubeapi.Createpgtask(apiserver.RESTClient,
			getBackupParams(cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], clusterName, taskName, crv1.PgtaskBackrestBackup, podname, "database",
				util.GetValueOrDefault(cluster.Spec.PGOImagePrefix, apiserver.Pgo.Pgo.PGOImagePrefix), request.BackupOpts, request.BackrestStorageType, operator.GetS3VerifyTLSSetting(&cluster), jobName, ns, pgouser),
			ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "created Pgtask "+taskName)

	}

	return resp
}

func getBackupParams(identifier, clusterName, taskName, action, podName, containerName, imagePrefix, backupOpts, backrestStorageType, s3VerifyTLS, jobName, ns, pgouser string) *crv1.Pgtask {
	var newInstance *crv1.Pgtask
	spec := crv1.PgtaskSpec{}
	spec.Name = taskName
	spec.Namespace = ns

	spec.TaskType = crv1.PgtaskBackrest
	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_JOB_NAME] = jobName
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName
	spec.Parameters[config.LABEL_POD_NAME] = podName
	spec.Parameters[config.LABEL_CONTAINER_NAME] = containerName
	// pass along the appropriate image prefix for the backup task
	// this will be used by the associated backrest job
	spec.Parameters[config.LABEL_IMAGE_PREFIX] = imagePrefix
	spec.Parameters[config.LABEL_BACKREST_COMMAND] = action
	spec.Parameters[config.LABEL_BACKREST_OPTS] = backupOpts
	spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE] = backrestStorageType
	spec.Parameters[config.LABEL_BACKREST_S3_VERIFY_TLS] = s3VerifyTLS

	newInstance = &crv1.Pgtask{
		ObjectMeta: metav1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = identifier
	newInstance.ObjectMeta.Labels[config.LABEL_PGOUSER] = pgouser
	return newInstance
}

// getBackrestRepoPodName goes through the pod list to identify the
// pgBackRest repo pod and then returns the pod name.
func getBackrestRepoPodName(cluster *crv1.Pgcluster, ns string) (string, error) {
	//look up the backrest-repo pod name
	selector := "pg-cluster=" + cluster.Spec.Name + ",pgo-backrest-repo=true"

	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: selector,
	}

	repopods, err := apiserver.Clientset.CoreV1().Pods(ns).List(options)
	if len(repopods.Items) != 1 {
		log.Errorf("pods len != 1 for cluster %s", cluster.Spec.Name)
		return "", errors.New("backrestrepo pod not found for cluster " + cluster.Spec.Name)
	}
	if err != nil {
		log.Error(err)
		return "", err
	}

	repopodName := repopods.Items[0].Name

	return repopodName, err
}

func isPrimary(pod *v1.Pod, clusterName string) bool {
	if pod.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] == clusterName {
		return true
	}
	return false

}

func isReady(pod *v1.Pod) bool {
	readyCount := 0
	containerCount := 0
	for _, stat := range pod.Status.ContainerStatuses {
		containerCount++
		if stat.Ready {
			readyCount++
		}
	}
	if readyCount != containerCount {
		return false
	}
	return true

}

// isPrimaryReady goes through the pod list to first identify the
// Primary pod and, once identified, determine if it is in a
// ready state. If not, it returns an error, otherwise it returns
// a nil value
func isPrimaryReady(cluster *crv1.Pgcluster, ns string) error {
	primaryReady := false

	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, cluster.Name,
		config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_PRIMARY)

	pods, err := apiserver.Clientset.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, p := range pods.Items {
		if isPrimary(&p, cluster.Spec.Name) && isReady(&p) {
			primaryReady = true
		}
	}

	if primaryReady == false {
		return errors.New("primary pod is not in Ready state")
	}
	return nil
}

// ShowBackrest ...
func ShowBackrest(name, selector, ns string) msgs.ShowBackrestResponse {
	var err error

	response := msgs.ShowBackrestResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Items = make([]msgs.ShowBackrestDetail, 0)

	if selector == "" && name == "all" {
	} else {
		if selector == "" {
			selector = "name=" + name
		}
	}

	clusterList := crv1.PgclusterList{}

	//get a list of all clusters
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
		&clusterList, selector, ns)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debugf("clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {
		podname, err := getBackrestRepoPodName(&c, ns)

		if err != nil {
			log.Error(err)
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		// so we potentially add two "pieces of detail" based on whether or not we
		// have a local repository, a s3 repository, or both
		storageTypes := c.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]

		for _, storageType := range apiserver.GetBackrestStorageTypes() {

			// so the way we currently store the different repos is not ideal, and
			// this is not being fixed right now, so we'll follow this logic:
			//
			// 1. If storage type is "local" and the string either contains "local" or
			// is empty, we can add the pgBackRest info
			// 2. if the storage type is "s3" and the string contains "s3", we can
			// add the pgBackRest info
			// 3. Otherwise, continue
			if (storageTypes == "" && storageType != "local") || (storageTypes != "" && !strings.Contains(storageTypes, storageType)) {
				continue
			}

			// begin preparing the detailed response
			detail := msgs.ShowBackrestDetail{
				Name:        c.Name,
				StorageType: storageType,
			}

			verifyTLS, _ := strconv.ParseBool(operator.GetS3VerifyTLSSetting(&c))

			// get the pgBackRest info using this legacy function
			info, err := getInfo(c.Name, storageType, podname, ns, verifyTLS)

			// see if the function returned successfully, and if so, unmarshal the JSON
			if err != nil {
				log.Error(err)
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()

				return response
			}

			if err := json.Unmarshal([]byte(info), &detail.Info); err != nil {
				log.Error(err)
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()

				return response
			}

			// append the details to the list of items
			response.Items = append(response.Items, detail)
		}

	}

	return response
}

func getInfo(clusterName, storageType, podname, ns string, verifyTLS bool) (string, error) {
	log.Debug("backrest info command requested")

	cmd := pgBackRestInfoCommand

	if storageType == "s3" {
		cmd = append(cmd, repoTypeFlagS3...)

		if !verifyTLS {
			cmd = append(cmd, noRepoS3VerifyTLS)
		}
	}

	output, stderr, err := kubeapi.ExecToPodThroughAPI(apiserver.RESTConfig, apiserver.Clientset, cmd, containername, podname, ns, nil)

	if err != nil {
		log.Error(err, stderr)
		return "", err
	}

	log.Debug("output=[" + output + "]")

	log.Debug("backrest info ends")

	return output, err
}

//  Restore ...
// pgo restore mycluster --to-cluster=restored
func Restore(request *msgs.RestoreRequest, ns, pgouser string) msgs.RestoreResponse {
	resp := msgs.RestoreResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("Restore %v\n", request)

	if request.RestoreOpts != "" {
		err := backupoptions.ValidateBackupOpts(request.RestoreOpts, request)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, request.FromCluster, ns)
	if !found {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = request.FromCluster + " was not found, verify cluster name"
		return resp
	} else if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	// check if the current cluster is not upgraded to the deployed
	// Operator version. If not, do not allow the command to complete
	if cluster.Annotations[config.ANNOTATION_IS_UPGRADED] == config.ANNOTATIONS_FALSE {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("%s %s", cluster.Name, msgs.UpgradeError)
		return resp
	}

	// verify that the cluster we are restoring from has backrest enabled
	if cluster.Labels[config.LABEL_BACKREST] != "true" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "can't restore, cluster restoring from does not have backrest enabled"
		return resp
	}

	// Return an error if any clusters identified for the restore are in standby mode.  Restoring
	// from a standby cluster is not allowed since the cluster is following a remote primary,
	// which itself is responsible for performing any restores as required for the cluster.
	if cluster.Spec.Standby {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("Request rejected, unable to restore cluster "+
			"%s: %s.", cluster.Name, apiserver.ErrStandbyNotAllowed.Error())
		return resp
	}

	// ensure the backrest storage type specified for the backup is valid and enabled in the
	// cluster
	err = util.ValidateBackrestStorageTypeOnBackupRestore(request.BackrestStorageType,
		cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], true)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	var id string
	id, err = createRestoreWorkflowTask(cluster.Name, ns)
	if err != nil {
		resp.Results = append(resp.Results, err.Error())
		return resp
	}

	pgtask, err := getRestoreParams(request, ns, cluster)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	pgtask.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER] = cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER]
	pgtask.ObjectMeta.Labels[config.LABEL_PGOUSER] = pgouser
	pgtask.Spec.Parameters[crv1.PgtaskWorkflowID] = id

	//create a pgtask for the restore workflow
	err = kubeapi.Createpgtask(apiserver.RESTClient, pgtask, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	resp.Results = append(resp.Results, "restore performed on "+request.FromCluster+" to "+request.ToPVC+" opts="+request.RestoreOpts+" pitr-target="+request.PITRTarget)

	resp.Results = append(resp.Results, "workflow id "+id)

	return resp
}

func getRestoreParams(request *msgs.RestoreRequest, ns string, cluster crv1.Pgcluster) (*crv1.Pgtask, error) {
	var newInstance *crv1.Pgtask

	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = "backrest-restore-" + request.FromCluster + "-to-" + request.ToPVC
	spec.TaskType = crv1.PgtaskBackrestRestore
	spec.Parameters = make(map[string]string)
	spec.Parameters[config.LABEL_BACKREST_RESTORE_FROM_CLUSTER] = request.FromCluster
	spec.Parameters[config.LABEL_BACKREST_RESTORE_TO_PVC] = request.ToPVC
	spec.Parameters[config.LABEL_BACKREST_RESTORE_OPTS] = request.RestoreOpts
	spec.Parameters[config.LABEL_BACKREST_PITR_TARGET] = request.PITRTarget
	spec.Parameters[config.LABEL_PGBACKREST_STANZA] = "db"
	spec.Parameters[config.LABEL_PGBACKREST_DB_PATH] = "/pgdata/" + request.ToPVC
	spec.Parameters[config.LABEL_PGBACKREST_REPO_PATH] = "/backrestrepo/" + request.FromCluster + "-backrest-shared-repo"
	spec.Parameters[config.LABEL_PGBACKREST_REPO_HOST] = request.FromCluster + "-backrest-shared-repo"
	spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE] = request.BackrestStorageType
	spec.Parameters[config.LABEL_BACKREST_S3_VERIFY_TLS] = operator.GetS3VerifyTLSSetting(&cluster)

	// validate & parse nodeLabel if exists
	if request.NodeLabel != "" {
		if err := apiserver.ValidateNodeLabel(request.NodeLabel); err != nil {
			return nil, err
		}

		parts := strings.Split(request.NodeLabel, "=")
		spec.Parameters[config.LABEL_NODE_LABEL_KEY] = parts[0]
		spec.Parameters[config.LABEL_NODE_LABEL_VALUE] = parts[1]

		log.Debug("Restore node labels used from user entered flag")
	}

	labels := make(map[string]string)
	newInstance = &crv1.Pgtask{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Name:   spec.Name,
		},
		Spec: spec,
	}
	return newInstance, nil
}

func createRestoreWorkflowTask(clusterName, ns string) (string, error) {

	existingTask := crv1.Pgtask{}

	taskName := clusterName + "-" + crv1.PgtaskWorkflowBackrestRestoreType

	//delete any existing pgtask with the same name
	found, _ := kubeapi.Getpgtask(apiserver.RESTClient,
		&existingTask, taskName, ns)
	if found {
		log.Debugf("deleting prior pgtask %s", taskName)

		if err := kubeapi.Deletepgtask(apiserver.RESTClient, taskName, ns); err != nil {
			return "", err
		}
	}

	//create pgtask CRD
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = clusterName + "-" + crv1.PgtaskWorkflowBackrestRestoreType
	spec.TaskType = crv1.PgtaskWorkflow

	spec.Parameters = make(map[string]string)
	spec.Parameters[crv1.PgtaskWorkflowSubmittedStatus] = time.Now().Format(time.RFC3339)
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName

	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error(err)
		return "", err
	}
	spec.Parameters[crv1.PgtaskWorkflowID] = string(u[:len(u)-1])

	newInstance := &crv1.Pgtask{
		ObjectMeta: metav1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	newInstance.ObjectMeta.Labels[crv1.PgtaskWorkflowID] = spec.Parameters[crv1.PgtaskWorkflowID]

	err = kubeapi.Createpgtask(apiserver.RESTClient, newInstance, ns)
	if err != nil {
		log.Error(err)
		return "", err
	}
	return spec.Parameters[crv1.PgtaskWorkflowID], err
}

// clusterNamesToPGClusterList takes a list of cluster names as specified by a slice of
// strings containing cluster names and then returns a PgclusterList containing Pgcluster's
// corresponding to those names.
func clusterNamesToPGClusterList(namespace string, clusterNames []string) (crv1.PgclusterList,
	error) {
	selector := fmt.Sprintf("%s in(%s)", config.LABEL_NAME, strings.Join(clusterNames, ","))
	clusterList := crv1.PgclusterList{}
	if err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, selector,
		namespace); err != nil {
		return clusterList, err
	}
	return clusterList, nil
}
