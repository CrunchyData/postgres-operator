package task

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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

type rmdatajobTemplateFields struct {
	JobName            string
	Name               string
	PvcName            string
	ClusterName        string
	PGOImagePrefix     string
	PGOImageTag        string
	SecurityContext    string
	DataRoot           string
	ContainerResources string
}

// RemoveData ...
func RemoveData(namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	//create the Job to remove the data
	pvcName := task.Spec.Parameters[config.LABEL_PVC_NAME]
	clusterName := task.Spec.Parameters[config.LABEL_PG_CLUSTER]

	cr := ""
	if operator.Pgo.DefaultRmdataResources != "" {
		tmp, err := operator.Pgo.GetContainerResource(operator.Pgo.DefaultRmdataResources)
		if err != nil {
			log.Error(err)
			return
		}
		cr = operator.GetContainerResourcesJSON(&tmp)

	}

	jobName := clusterName + "-rmdata-" + util.RandStringBytesRmndr(4)

	jobFields := rmdatajobTemplateFields{
		JobName:            jobName,
		Name:               task.Spec.Name + "-" + pvcName,
		ClusterName:        clusterName,
		PvcName:            pvcName,
		PGOImagePrefix:     operator.Pgo.Pgo.PGOImagePrefix,
		PGOImageTag:        operator.Pgo.Pgo.PGOImageTag,
		SecurityContext:    util.CreateSecContext(task.Spec.StorageSpec.Fsgroup, task.Spec.StorageSpec.SupplementalGroups),
		DataRoot:           task.Spec.Parameters[config.LABEL_DATA_ROOT],
		ContainerResources: cr,
	}
	log.Debugf("creating rmdata job %s for cluster %s pvc %s", jobName, task.Spec.Name, pvcName)

	var doc2 bytes.Buffer
	err := config.RmdatajobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		config.RmdatajobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	var jobname string
	jobname, err = kubeapi.CreateJob(clientset, &newjob, namespace)
	if err != nil {
		log.Errorf("got error when creating rmdata job %s", jobname)
		return
	}
	log.Debugf("successfully created rmdata job %s", jobname)

}
