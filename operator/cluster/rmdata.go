// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"strconv"
)

type RmdataJob struct {
	JobName        string
	ClusterName    string
	PGOImagePrefix string
	PGOImageTag    string
	//		SecurityContext string
	RemoveData         string
	RemoveBackup       string
	IsBackup           string
	IsReplica          string
	ContainerResources string
}

// CreateService ...
func CreateRmdataJob(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, namespace string, removeData, removeBackup, isReplica, isBackup bool) error {
	var err error

	cr := ""
	if operator.Pgo.DefaultBackupResources != "" {
		tmp, err := operator.Pgo.GetContainerResource(operator.Pgo.DefaultBackupResources)
		if err != nil {
			log.Error(err)
			return err
		}
		cr = operator.GetContainerResourcesJSON(&tmp)

	}

	jobName := cl.Spec.Name + "-rmdata-" + util.RandStringBytesRmndr(4)

	jobFields := RmdataJob{
		JobName:        jobName,
		ClusterName:    cl.Spec.Name,
		PGOImagePrefix: operator.Pgo.Pgo.PGOImagePrefix,
		PGOImageTag:    operator.Pgo.Pgo.PGOImageTag,
		//		SecurityContext:    util.CreateSecContext(job.Spec.StorageSpec.Fsgroup, job.Spec.StorageSpec.SupplementalGroups),
		RemoveData:         strconv.FormatBool(removeData),
		RemoveBackup:       strconv.FormatBool(removeBackup),
		IsBackup:           strconv.FormatBool(isReplica),
		IsReplica:          strconv.FormatBool(isBackup),
		ContainerResources: cr,
	}

	var doc2 bytes.Buffer
	err = config.JobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if operator.CRUNCHY_DEBUG {
		config.JobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return err
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_RMDATA,
		&newjob.Spec.Template.Spec.Containers[0])

	_, err = kubeapi.CreateJob(clientset, &newjob, namespace)
	if err != nil {
		return err
	}

	return err
}
