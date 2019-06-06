package backup

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
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type jobTemplateFields struct {
	Name               string
	JobName            string
	PvcName            string
	CCPImagePrefix     string
	CCPImageTag        string
	SecurityContext    string
	BackupHost         string
	BackupUserSecret   string
	BackupPort         string
	BackupOpts         string
	ContainerResources string
}

// AddBackupBase creates a backup job and its pvc
func AddBackupBase(clientset *kubernetes.Clientset, client *rest.RESTClient, job *crv1.Pgbackup, namespace string) {
	var err error

	if job.Spec.BackupStatus == crv1.CompletedStatus {
		log.Warn("pgbackup " + job.Spec.Name + " already completed, not recreating it")
		return
	}

	log.Info("creating Pgbackup object" + " in namespace " + namespace)
	log.Info("created with Name=" + job.Spec.Name + " in namespace " + namespace)

	//create the PVC if necessary
	var pvcName string
	if job.Spec.BackupPVC != "" {
		pvcName = job.Spec.BackupPVC
	} else {
		pvcName, err = pvc.CreatePVC(clientset, &job.Spec.StorageSpec, job.Spec.Name+"-backup", job.Spec.BackupHost, namespace)
		if err != nil {
			log.Error(err.Error())
		} else {
			log.Info("created backup PVC =" + pvcName + " in namespace " + namespace)
		}
	}

	//update the pvc name in the CRD
	err = util.Patch(client, "/spec/storagespec/name", pvcName, "pgbackups", job.Spec.Name, namespace)

	cr := ""
	if operator.Pgo.DefaultBackupResources != "" {
		tmp, err := operator.Pgo.GetContainerResource(operator.Pgo.DefaultBackupResources)
		if err != nil {
			log.Error(err)
			return
		}
		cr = operator.GetContainerResourcesJSON(&tmp)

	}

	//generate a JobName
	jobName := "backup-" + job.Spec.Name + "-" + util.RandStringBytesRmndr(4)

	//create the job -
	jobFields := jobTemplateFields{
		JobName:            jobName,
		Name:               job.Spec.Name,
		PvcName:            util.CreatePVCSnippet(job.Spec.StorageSpec.StorageType, pvcName),
		CCPImagePrefix:     operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:        job.Spec.CCPImageTag,
		SecurityContext:    util.CreateSecContext(job.Spec.StorageSpec.Fsgroup, job.Spec.StorageSpec.SupplementalGroups),
		BackupHost:         job.Spec.BackupHost,
		BackupUserSecret:   job.Spec.BackupUserSecret,
		BackupPort:         job.Spec.BackupPort,
		BackupOpts:         job.Spec.BackupOpts,
		ContainerResources: cr,
	}

	var doc2 bytes.Buffer
	err = config.JobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		config.JobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	_, err = kubeapi.CreateJob(clientset, &newjob, namespace)
	if err != nil {
		return
	}

	//update the backup CRD status to submitted
	err = util.Patch(client, "/spec/backupstatus", crv1.SubmittedStatus, "pgbackups", job.Spec.Name, namespace)
	if err != nil {
		log.Error(err.Error())
	}

}

// DeleteBackupBase deletes a backup job
func DeleteBackupBase(clientset *kubernetes.Clientset, client *rest.RESTClient, job *crv1.Pgbackup, namespace string) {
	var jobName = "backup-" + job.Spec.Name

	err := kubeapi.DeleteJob(clientset, jobName, namespace)
	if err != nil {
		log.Error("error deleting Job " + jobName + err.Error())
		return
	}

	//make sure job is actually reporting as deleted
	for i := 0; i < 5; i++ {
		_, found := kubeapi.GetJob(clientset, jobName, namespace)
		if !found {
			break
		}
		if err != nil {
			log.Error(err)
		}
		log.Debug("waiting for backup job to report being deleted")
		time.Sleep(time.Second * time.Duration(3))
	}
}

func UpdateBackupPaths(clientset *kubernetes.Clientset, jobName, namespace string) string {
	//its pod has this label job-name=backup-yank-fjus
	selector := "job-name=" + jobName
	log.Debugf("looking for pod with selector %s", selector)
	podList, err := kubeapi.GetPods(clientset, selector, namespace)
	if err != nil {
		log.Error(err.Error())
		return err.Error()
	}

	if len(podList.Items) != 1 {
		log.Error("could not find a pod for this job")
		return "error"
	}

	podName := podList.Items[0].Name
	log.Debugf("found pod %s", podName)
	backupPath, err := getBackupPath(clientset, podName, namespace)
	if err != nil {
		log.Error("error in getting logs %s", err.Error())
		return err.Error()
	}
	log.Debugf("backupPath is %s", backupPath)

	return backupPath

}

//this func assumes the pod has completed and its a backup job pod
func getBackupPath(clientset *kubernetes.Clientset, podName, namespace string) (string, error) {
	opts := v1.PodLogOptions{
		Container: "backup",
	}
	var logs bytes.Buffer

	err := kubeapi.GetLogs(clientset, opts, &logs, podName, namespace)
	if err != nil {
		log.Error("error in getting logs %s", err.Error())
		return "", err
	}

	backPathRegEx := regexp.MustCompile(`BACKUP_PATH is set to \/pgdata\/.*\.`)
	fullBackupPathStr := backPathRegEx.FindString(logs.String())
	backupPath := strings.TrimSuffix(strings.TrimPrefix(fullBackupPathStr, "BACKUP_PATH is set to /pgdata/"), ".")

	return backupPath, nil

}
