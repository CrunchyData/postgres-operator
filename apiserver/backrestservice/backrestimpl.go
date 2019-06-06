package backrestservice

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
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/apiserver/backupoptions"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const backrestCommand = "pgbackrest"
const backrestInfoCommand = "info"
const containername = "database"

//  CreateBackup ...
// pgo backup mycluster
// pgo backup --selector=name=mycluster
func CreateBackup(request *msgs.CreateBackrestBackupRequest, ns string) msgs.CreateBackrestBackupResponse {
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

	if request.Selector != "" {
		//use the selector instead of an argument list to filter on

		clusterList := crv1.PgclusterList{}

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

		if cluster.Labels[config.LABEL_BACKREST] != "true" {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = clusterName + " does not have pgbackrest enabled"
			return resp
		}

		err = validateBackrestStorageType(request.BackrestStorageType, cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], false)
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
			err = kubeapi.DeleteJobs(apiserver.Clientset, selector, ns)
			if err != nil {
				log.Error(err)
			}

			//a hack sort of due to slow propagation
			for i := 0; i < 3; i++ {
				jobList, err := kubeapi.GetJobs(apiserver.Clientset, selector, ns)
				if err != nil {
					log.Error(err)
				}
				if len(jobList.Items) > 0 {
					log.Debug("sleeping a bit for delete job propagation")
					time.Sleep(time.Second * 2)
				}
			}

		}

		//get pod name from cluster
		var podname string
		podname, err = getPrimaryPodName(&cluster, ns)

		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		jobName := "backrest-" + crv1.PgtaskBackrestBackup + "-" + clusterName
		log.Debugf("setting jobName to %s", jobName)

		err = kubeapi.Createpgtask(apiserver.RESTClient,
			getBackupParams(clusterName, taskName, crv1.PgtaskBackrestBackup, podname, "database", request.BackupOpts, request.BackrestStorageType, jobName, ns),
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

func getBackupParams(clusterName, taskName, action, podName, containerName, backupOpts, backrestStorageType, jobName, ns string) *crv1.Pgtask {
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
	spec.Parameters[config.LABEL_BACKREST_COMMAND] = action
	spec.Parameters[config.LABEL_BACKREST_OPTS] = backupOpts
	spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE] = backrestStorageType

	newInstance = &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	newInstance.ObjectMeta.Labels = make(map[string]string)
	newInstance.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] = clusterName
	return newInstance
}

func removeBackupJob(clusterName string) {

}

func getDeployName(cluster *crv1.Pgcluster, ns string) (string, error) {
	var depName string

	selector := config.LABEL_PGPOOL + "!=true," + config.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name + "," + config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name

	deps, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
	if err != nil {
		return depName, err
	}

	if len(deps.Items) != 1 {
		return depName, errors.New("error:  deployment count is wrong for backrest backup " + cluster.Spec.Name)
	}
	for _, d := range deps.Items {
		return d.Name, err
	}

	return depName, errors.New("unknown error in backrest backup")
}

func getPrimaryPodName(cluster *crv1.Pgcluster, ns string) (string, error) {

	//look up the backrest-repo pod name
	selector := "pg-cluster=" + cluster.Spec.Name + ",pgo-backrest-repo=true"
	repopods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
	if len(repopods.Items) != 1 {
		log.Errorf("pods len != 1 for cluster %s", cluster.Spec.Name)
		return "", errors.New("backrestrepo pod not found for cluster " + cluster.Spec.Name)
	}
	if err != nil {
		log.Error(err)
		return "", err
	}

	repopodName := repopods.Items[0].Name

	primaryReady := false

	//make sure the primary pod is in the ready state
	selector = config.LABEL_SERVICE_NAME + "=" + cluster.Spec.Name

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
	if err != nil {
		return "", err
	}
	for _, p := range pods.Items {
		if isPrimary(&p, cluster.Spec.Name) && isReady(&p) {
			primaryReady = true
		}
	}

	if primaryReady == false {
		return "", errors.New("primary pod is not in Ready state")
	}

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
		detail := msgs.ShowBackrestDetail{}
		detail.Name = c.Name

		podname, err := getPrimaryPodName(&c, ns)

		if err != nil {
			log.Error(err)
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		//here is where we would exec to get the backrest info
		info, err := getInfo(c.Name, c.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], podname, ns)
		if err != nil {
			detail.Info = err.Error()
		} else {
			detail.Info = info
		}

		response.Items = append(response.Items, detail)
	}

	return response

}

func getInfo(clusterName, storageType, podname, ns string) (string, error) {

	var err error
	const repoTypeFlagS3 = "--repo-type=s3"

	cmd := make([]string, 0)

	log.Debug("backrest info command requested")
	//pgbackrest --stanza=db info
	cmd = append(cmd, backrestCommand)
	cmd = append(cmd, backrestInfoCommand)

	log.Debugf("command is %v ", cmd)

	var output string
	if storageType != "s3" {
		outputLocal, stderr, err := kubeapi.ExecToPodThroughAPI(apiserver.RESTConfig, apiserver.Clientset, cmd, containername, podname, ns, nil)
		if err != nil {
			log.Error(err, stderr)
			return "", err
		}
		output = "\nStorage Type: local\n" + outputLocal
	}

	if strings.Contains(storageType, "s3") {
		cmd = append(cmd, repoTypeFlagS3)
		outputS3, stderr, err := kubeapi.ExecToPodThroughAPI(apiserver.RESTConfig, apiserver.Clientset, cmd, containername, podname, ns, nil)
		if err != nil {
			log.Error(err, stderr)
			return "", err
		}
		output = output + "\nStorage Type: s3\n" + outputS3
	}

	log.Debug("output=[" + output + "]")

	log.Debug("backrest info ends")
	return output, err

}

//  Restore ...
// pgo restore mycluster --to-cluster=restored
func Restore(request *msgs.RestoreRequest, ns string) msgs.RestoreResponse {
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

	//verify that the cluster we are restoring from has backrest enabled
	if cluster.Labels[config.LABEL_BACKREST] != "true" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "can't restore, cluster restoring from does not have backrest enabled"
		return resp
	}

	err = validateBackrestStorageType(request.BackrestStorageType, cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], true)
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

	newInstance = &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: spec.Name,
		},
		Spec: spec,
	}
	return newInstance, nil
}

func createRestoreWorkflowTask(clusterName, ns string) (string, error) {

	existingTask := crv1.Pgtask{}

	taskName := clusterName + "-" + crv1.PgtaskWorkflowBackrestRestoreType

	//delete any existing pgtask with the same name
	found, err := kubeapi.Getpgtask(apiserver.RESTClient,
		&existingTask, taskName, ns)
	if found {
		log.Debugf("deleting prior pgtask %s", taskName)
		err = kubeapi.Deletepgtask(apiserver.RESTClient, taskName, ns)
		if err != nil {
			return "", err
		}
	}

	//create pgtask CRD
	spec := crv1.PgtaskSpec{}
	spec.Namespace = ns
	spec.Name = clusterName + "-" + crv1.PgtaskWorkflowBackrestRestoreType
	spec.TaskType = crv1.PgtaskWorkflow

	spec.Parameters = make(map[string]string)
	spec.Parameters[crv1.PgtaskWorkflowSubmittedStatus] = time.Now().Format("2006-01-02.15.04.05")
	spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName

	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error(err)
		return "", err
	}
	spec.Parameters[crv1.PgtaskWorkflowID] = string(u[:len(u)-1])

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
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

func validateBackrestStorageType(requestBackRestStorageType, clusterBackRestStorageType string, restore bool) error {

	if requestBackRestStorageType != "" && !apiserver.IsValidBackrestStorageType(requestBackRestStorageType) {
		return fmt.Errorf("Invalid value provided for pgBackRest storage type. The following values are allowed: %s",
			"\""+strings.Join(apiserver.GetBackrestStorageTypes(), "\", \"")+"\"")
	} else if requestBackRestStorageType != "" && strings.Contains(requestBackRestStorageType, "s3") &&
		!strings.Contains(clusterBackRestStorageType, "s3") {
		return errors.New("Storage type 's3' not allowed. S3 storage is not enabled for pgBackRest in this cluster")
	} else if (requestBackRestStorageType == "" || strings.Contains(requestBackRestStorageType, "local")) &&
		(clusterBackRestStorageType != "" && !strings.Contains(clusterBackRestStorageType, "local")) {
		return errors.New("Storage type 'local' not allowed. Local storage is not enabled for pgBackRest in this cluster. " +
			"If this cluster uses S3 storage only, specify 's3' for the pgBackRest storage type.")
	}

	// storage type validation that is only applicable for restores
	if restore && requestBackRestStorageType != "" && len(strings.Split(requestBackRestStorageType, ",")) > 1 {
		return fmt.Errorf("Multiple storage types cannot be selected cannot be select when performing a restore. Please "+
			"select one of the following: %s", "\""+strings.Join(apiserver.GetBackrestStorageTypes(), "\", \"")+"\"")
	}

	return nil
}
