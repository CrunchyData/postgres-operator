package benchmarkservice

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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ShowBenchmark ...
func ShowBenchmark(request *msgs.ShowBenchmarkRequest) msgs.ShowBenchmarkResponse {
	log.Debug("Show Benchmark called")

	br := &msgs.ShowBenchmarkResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
		Results: make([]string, 0),
	}

	if err := request.Validate(); err != nil {
		br.Status.Code = msgs.Error
		br.Status.Msg = fmt.Sprintf("invalid show benchmark request: %s", err)
		return *br
	}

	selector := fmt.Sprintf("%s=true,", config.LABEL_PGO_BENCHMARK)
	if request.ClusterName != "" {
		selector += fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, request.ClusterName)
	} else if request.Selector != "" {
		selector += request.Selector
	}

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, request.Namespace)
	if err != nil {
		br.Status.Code = msgs.Error
		br.Status.Msg = fmt.Sprintf("error retrieving benchmark pods: %s", err)
		return *br
	}

	if len(pods.Items) == 0 {
		br.Status.Code = msgs.Error
		br.Status.Msg = errors.New("no benchmark jobs found").Error()
		return *br
	}

	for _, pod := range pods.Items {
		opts := v1.PodLogOptions{
			Container: "pgbench",
		}

		var logs bytes.Buffer
		err := kubeapi.GetLogs(apiserver.Clientset, opts, &logs, pod.Name, request.Namespace)
		if err != nil {
			br.Status.Code = msgs.Error
			br.Status.Msg = errors.New("no benchmarks jobs found").Error()
			return *br
		}

		results := fmt.Sprintf("Results for %s:\n", pod.Labels["job-name"])
		scanner := bufio.NewScanner(strings.NewReader(logs.String()))
		for scanner.Scan() {
			results += fmt.Sprintf("\t%s\n", scanner.Text())
		}

		br.Results = append(br.Results, results)
	}
	return *br
}

// DeleteBenchmark ...
func DeleteBenchmark(request *msgs.DeleteBenchmarkRequest) msgs.DeleteBenchmarkResponse {
	log.Debug("Delete Benchmark called")

	br := &msgs.DeleteBenchmarkResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
		Results: make([]string, 0),
	}

	if err := request.Validate(); err != nil {
		br.Status.Code = msgs.Error
		br.Status.Msg = fmt.Sprintf("Invalid delete benchmark request: %s", err)
		return *br
	}

	selector := fmt.Sprintf("%s=true,", config.LABEL_PGO_BENCHMARK)
	if request.ClusterName != "" {
		selector += fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, request.ClusterName)
	} else if request.Selector != "" {
		selector += request.Selector
	}

	jobs, err := kubeapi.GetJobs(apiserver.Clientset, selector, request.Namespace)
	if err != nil {
		br.Status.Code = msgs.Error
		br.Status.Msg = fmt.Sprintf("Could not get benchmark jobs: %s", err)
		return *br
	}

	for _, job := range jobs.Items {
		err := kubeapi.DeleteJob(apiserver.Clientset, job.Name, request.Namespace)
		if err != nil {
			br.Status.Code = msgs.Error
			br.Status.Msg = fmt.Sprintf("Could not delete benchmark job %s: %s", job.Name, err)
			return *br
		}

		msg := fmt.Sprintf("deleted benchmark %s", job.Name)
		br.Results = append(br.Results, msg)
	}

	err = kubeapi.Deletepgtasks(apiserver.RESTClient, selector, request.Namespace)
	if err != nil {
		br.Status.Code = msgs.Error
		br.Status.Msg = fmt.Sprintf("Could not delete benchmark pgtasks: %s", err)
	}

	return *br
}

//  CreateBenchmark ...
func CreateBenchmark(request *msgs.CreateBenchmarkRequest, ns string) msgs.CreateBenchmarkResponse {
	br := &msgs.CreateBenchmarkResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
		Results: make([]string, 0),
	}

	if err := request.Validate(); err != nil {
		br.Status.Code = msgs.Error
		br.Status.Msg = fmt.Sprintf("Invalid create benchmark request: %s", err)
		return *br
	}

	log.Debug("Getting cluster")
	var selector string
	if request.ClusterName != "" {
		selector = fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, request.ClusterName)
	} else if request.Selector != "" {
		selector = request.Selector
	}

	clusterList := crv1.PgclusterList{}
	err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, selector, ns)
	if err != nil {
		br.Status.Code = msgs.Error
		br.Status.Msg = fmt.Sprintf("Could not get cluster via selector: %s", err)
		return *br
	}

	timeNow := time.Now().Format("2006-01-02-01-04-05")
	for _, cluster := range clusterList.Items {
		uid := util.RandStringBytesRmndr(4)

		taskName := fmt.Sprintf("benchmark-%s-%s", cluster.Name, uid)
		configMapName := fmt.Sprintf("%s-tx", taskName)

		configMapName, err := createConfigMapFromPolicy(request.Policy, configMapName, ns)
		if err != nil {
			br.Status.Code = msgs.Error
			br.Status.Msg = err.Error()
			return *br
		}

		workflowID, err := createWorkflowTask(cluster.Name, uid, ns)
		if err != nil {
			br.Status.Code = msgs.Error
			br.Status.Msg = fmt.Errorf("could not create benchmark workflow task: %s", err).Error()
			return *br
		}

		benchmark := benchmarkTask{
			benchmarkOpts:  request.BenchmarkOpts,
			ccpImagePrefix: apiserver.Pgo.Cluster.CCPImagePrefix,
			ccpImageTag:    apiserver.Pgo.Cluster.CCPImageTag,
			clients:        defaultInt(request.Clients, 1),
			clusterName:    cluster.Name,
			configmapName:  configMapName,
			database:       defaultString(request.Database, cluster.Spec.Database),
			host:           cluster.Spec.PrimaryHost,
			initOpts:       request.InitOpts,
			jobs:           defaultInt(request.Jobs, 1),
			port:           cluster.Spec.Port,
			scale:          defaultInt(request.Scale, 1),
			secret:         cluster.Spec.PrimarySecretName,
			taskName:       taskName,
			timestamp:      timeNow,
			transactions:   defaultInt(request.Transactions, 1),
			workflowID:     workflowID,
			workflowName:   fmt.Sprintf("%s-%s-%s", cluster.Name, uid, crv1.PgtaskWorkflowCreateBenchmarkType),
		}

		task := benchmark.newBenchmarkTask()
		err = kubeapi.Createpgtask(apiserver.RESTClient, task, ns)
		if err != nil {
			br.Status.Code = msgs.Error
			br.Status.Msg = fmt.Sprintf("Could not create benchmark task: %s", err)
			return *br
		}

		created := fmt.Sprintf("Created benchmark task for %s", cluster.Name)
		br.Results = append(br.Results, created)

		workflow := fmt.Sprintf("workflow id is %s", workflowID)
		br.Results = append(br.Results, workflow)
	}
	return *br
}

type benchmarkTask struct {
	benchmarkOpts  string
	ccpImagePrefix string
	ccpImageTag    string
	clients        int
	clusterName    string
	configmapName  string
	database       string
	host           string
	initOpts       string
	jobs           int
	port           string
	scale          int
	secret         string
	taskName       string
	timestamp      string
	transactions   int
	workflowID     string
	workflowName   string
}

func (b benchmarkTask) newBenchmarkTask() *crv1.Pgtask {
	return &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: b.taskName,
			Labels: map[string]string{
				config.LABEL_PG_CLUSTER:    b.clusterName,
				config.LABEL_PGO_BENCHMARK: "true",
			},
		},
		Spec: crv1.PgtaskSpec{
			Name:     b.taskName,
			TaskType: crv1.PgtaskBenchmark,
			Parameters: map[string]string{
				"benchmarkOpts":  b.benchmarkOpts,
				"ccpImagePrefix": b.ccpImagePrefix,
				"ccpImageTag":    b.ccpImageTag,
				"clients":        strconv.Itoa(b.clients),
				"clusterName":    b.clusterName,
				"configmapName":  b.configmapName,
				"database":       b.database,
				"host":           b.host,
				"initOpts":       b.initOpts,
				"jobs":           strconv.Itoa(b.jobs),
				"port":           b.port,
				"scale":          strconv.Itoa(b.scale),
				"secret":         b.secret,
				"taskName":       b.taskName,
				"timestamp":      b.timestamp,
				"transactions":   strconv.Itoa(b.transactions),
				"workflowID":     b.workflowID,
				"workflowName":   b.workflowName,
			},
		},
	}
}

func defaultString(actual, defaultValue string) string {
	if actual == "" {
		return defaultValue
	}
	return actual
}

func defaultInt(actual, defaultValue int) int {
	if actual == 0 {
		return defaultValue
	}
	return actual
}

func createConfigMapFromPolicy(policyName, configMapName, ns string) (string, error) {
	configMapValue := "\"medium\": \"Memory\""

	if policyName != "" {
		policy := &crv1.Pgpolicy{}
		found, err := kubeapi.Getpgpolicy(apiserver.RESTClient, policy, policyName, ns)
		if !found {
			return "", fmt.Errorf("Policy not found: %s", policyName)
		}

		if policy.Spec.SQL == "" {
			return "", fmt.Errorf("Policy contained no SQL: %s", policy.Spec.Name)
		}

		configmap := &v1.ConfigMap{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: configMapName,
				Labels: map[string]string{
					config.LABEL_PGO_BENCHMARK: "true",
				},
			},
			Data: map[string]string{
				"transactions.sql": policy.Spec.SQL,
			},
		}

		err = kubeapi.CreateConfigMap(apiserver.Clientset, configmap, ns)
		if err != nil {
			return "", fmt.Errorf("Could not create transactions config map: %s", err)
		}
		configMapValue = fmt.Sprintf("\"configMap\": { \"name\": \"%s\" }", configMapName)
	}

	return configMapValue, nil
}

func createWorkflowTask(clusterName, uid, ns string) (string, error) {
	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		return "", err
	}
	id := string(u[:len(u)-1])

	taskName := fmt.Sprintf("%s-%s-%s", clusterName, uid, crv1.PgtaskWorkflowCreateBenchmarkType)
	task := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
			Labels: map[string]string{
				config.LABEL_PG_CLUSTER: clusterName,
				crv1.PgtaskWorkflowID:   id,
			},
		},
		Spec: crv1.PgtaskSpec{
			Namespace: ns,
			Name:      taskName,
			TaskType:  crv1.PgtaskWorkflow,
			Parameters: map[string]string{
				crv1.PgtaskWorkflowSubmittedStatus: time.Now().Format("2006-01-02.15.04.05"),
				config.LABEL_PG_CLUSTER:            clusterName,
				crv1.PgtaskWorkflowID:              id,
			},
		},
	}

	err = kubeapi.Createpgtask(apiserver.RESTClient, task, ns)
	if err != nil {
		return "", err
	}

	return id, err
}
