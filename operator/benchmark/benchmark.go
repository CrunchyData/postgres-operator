package benchmark

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
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type benchmarkJobTemplateFields struct {
	JobName             string
	TaskName            string
	Created             string
	ClusterName         string
	CCPImagePrefix      string
	CCPImageTag         string
	PGDatabase          string
	PGHost              string
	PGPort              string
	PGUserSecret        string
	PGBenchOpts         string
	PGBenchInitOpts     string
	PGBenchClients      string
	PGBenchJobs         string
	PGBenchScale        string
	PGBenchTransactions string
	PGBenchConfigMap    string
	WorkflowName        string
}

// Create ...
func Create(namespace string, clientset *kubernetes.Clientset, restclient *rest.RESTClient, task *crv1.Pgtask) {
	log.Debug("Create benchmark called in operator")

	jobFields := benchmarkJobTemplateFields{
		CCPImagePrefix:      task.Spec.Parameters["ccpImagePrefix"],
		CCPImageTag:         task.Spec.Parameters["ccpImageTag"],
		ClusterName:         task.Spec.Parameters["clusterName"],
		Created:             task.Spec.Parameters["created"],
		JobName:             task.Spec.Parameters["taskName"],
		PGBenchClients:      task.Spec.Parameters["clients"],
		PGBenchConfigMap:    task.Spec.Parameters["configmapName"],
		PGBenchInitOpts:     task.Spec.Parameters["initOpts"],
		PGBenchJobs:         task.Spec.Parameters["jobs"],
		PGBenchOpts:         task.Spec.Parameters["benchmarkOpts"],
		PGBenchScale:        task.Spec.Parameters["scale"],
		PGBenchTransactions: task.Spec.Parameters["transactions"],
		PGDatabase:          task.Spec.Parameters["database"],
		PGHost:              task.Spec.Parameters["host"],
		PGPort:              task.Spec.Parameters["port"],
		PGUserSecret:        task.Spec.Parameters["secret"],
		TaskName:            task.Spec.Parameters["taskName"],
		WorkflowName:        task.Spec.Parameters["workflowName"],
	}

	var doc2 bytes.Buffer
	err := config.BenchmarkJobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debug("Unmarshal benchmark template")
	newJob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newJob)
	if err != nil {
		log.Errorf("error unmarshalling json into Job %s", err)
		return
	}

	log.Debug("Creating benchmark job")
	_, err = kubeapi.CreateJob(clientset, &newJob, namespace)
	if err != nil {
		return
	}

	log.Debug("Getting benchmark job")
	_, found := kubeapi.GetJob(clientset, newJob.Name, namespace)
	if !found {
		log.Errorf("benchmark job not found: %s", newJob.Name)
		return
	}

	log.Debug("Updating benchmark workflow")
	workflowName := task.Spec.Parameters["workflowName"]
	err = UpdateWorkflow(restclient, workflowName, namespace, crv1.PgtaskWorkflowSubmittedStatus)
	if err != nil {
		log.Errorf("could not update benchmark workflow: %s", err)
		return
	}
}

func UpdateWorkflow(client *rest.RESTClient, name, namespace, status string) error {
	log.Debugf("benchmark workflow: update workflow %s", name)

	task := crv1.Pgtask{}
	found, err := kubeapi.Getpgtask(client, &task, name, namespace)
	if !found {
		log.Errorf("pgtask %s not found", name)
		return err
	} else if err != nil {
		return err
	}

	log.Debug("Updating workflow task")
	task.Spec.Parameters[status] = time.Now().Format("2006-01-02.15.04.05")
	err = kubeapi.Updatepgtask(client, &task, task.Name, namespace)
	if err != nil {
		return err
	}

	return err
}
