package backrestservice

/*
Copyright 2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const backrestCommand = "pgbackrest"
const backrestStanza = "--stanza=db"
const backrestInfoCommand = "info"
const containername = "database"

//  CreateBackup ...
// pgo backrest mycluster
// pgo backrest --selector=name=mycluster
func CreateBackup(request *msgs.CreateBackrestBackupRequest) msgs.CreateBackrestBackupResponse {
	resp := msgs.CreateBackrestBackupResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	if request.Selector != "" {
		//use the selector instead of an argument list to filter on

		clusterList := crv1.PgclusterList{}

		err := kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, request.Selector, apiserver.Namespace)
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
		log.Debug("create backrestbackup called for " + clusterName)
		taskName := clusterName + "-backrest-backup"

		cluster := crv1.Pgcluster{}
		found, err := kubeapi.Getpgcluster(apiserver.RESTClient, &cluster, clusterName, apiserver.Namespace)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = clusterName + " was not found, verify cluster name"
			return resp
		} else if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		result := crv1.Pgtask{}

		// error if it already exists
		found, err = kubeapi.Getpgtask(apiserver.RESTClient, &result, taskName, apiserver.Namespace)
		if !found {
			log.Debug("backrest backup pgtask " + taskName + " not found so we create it")
		} else if err != nil {
			resp.Results = append(resp.Results, "error getting pgtask for "+taskName)
			break
		} else {
			log.Debug("pgtask " + taskName + " was found so we recreate it")
			if result.Spec.Status == "" {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "pgtask " + taskName + " was not marked completed so we can not do another backup request."
				return resp
			}
			//remove the existing pgtask
			err := kubeapi.Deletepgtask(apiserver.RESTClient, taskName, apiserver.Namespace)
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}

			//remove any previous backup job
			removeBackupJob(taskName)
		}

		//get pod name from cluster
		var podname string
		podname, err = getPrimaryPodName(&cluster)

		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		err = kubeapi.Createpgtask(apiserver.RESTClient, getBackupParams(clusterName, taskName, crv1.PgtaskBackrestBackup, podname, "database"), apiserver.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "created Pgtask "+taskName)

	}

	return resp
}

func getBackupParams(clusterName, taskName, action, podName, containerName string) *crv1.Pgtask {
	var newInstance *crv1.Pgtask

	spec := crv1.PgtaskSpec{}
	spec.Name = taskName
	spec.TaskType = crv1.PgtaskBackrest
	spec.Parameters = make(map[string]string)
	spec.Parameters[util.LABEL_PG_CLUSTER] = clusterName
	spec.Parameters[util.LABEL_POD_NAME] = podName
	spec.Parameters[util.LABEL_CONTAINER_NAME] = containerName
	spec.Parameters[util.LABEL_ACTION] = action

	newInstance = &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
		},
		Spec: spec,
	}
	return newInstance
}

func removeBackupJob(name string) {

	_, found := kubeapi.GetJob(apiserver.Clientset, name, apiserver.Namespace)
	if !found {
		return
	}

	log.Debugf("found backrest backup job %s will remove\n", name)

	kubeapi.DeleteJob(apiserver.Clientset, name, apiserver.Namespace)
}

func getPrimaryPodName(cluster *crv1.Pgcluster) (string, error) {
	var podname string

	selector := util.LABEL_PGPOOL + "!=true," + util.LABEL_PG_CLUSTER + "=" + cluster.Spec.Name + "," + util.LABEL_PRIMARY + "=true"

	pods, err := kubeapi.GetPods(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		return podname, err
	}

	for _, p := range pods.Items {
		if isPrimary(&p) && isReady(&p) {
			return p.Name, err
		}
	}

	return podname, errors.New("primary pod is not in Ready state")
}

func isPrimary(pod *v1.Pod) bool {
	if pod.ObjectMeta.Labels[util.LABEL_PRIMARY] == "true" {
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
func ShowBackrest(name, selector string) msgs.ShowBackrestResponse {
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
		&clusterList, selector, apiserver.Namespace)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	log.Debug("clusters found len is %d\n", len(clusterList.Items))

	for _, c := range clusterList.Items {
		detail := msgs.ShowBackrestDetail{}
		detail.Name = c.Name

		//here is where we would exec to get the backrest info
		info, err := getInfo(c.Name)
		if err != nil {
			detail.Info = err.Error()
		} else {
			detail.Info = info
		}

		response.Items = append(response.Items, detail)
	}

	return response

}

func getInfo(clusterName string) (string, error) {

	var err error
	var PODNAME string

	//lookup podname
	PODNAME = "foo"

	cmd := make([]string, 0)

	log.Info("backrest info command requested")
	//pgbackrest --stanza=db info
	cmd = append(cmd, backrestCommand)
	cmd = append(cmd, backrestStanza)
	cmd = append(cmd, backrestInfoCommand)

	err = util.Exec(apiserver.RESTConfig, apiserver.Namespace, PODNAME, containername, cmd)
	if err != nil {
		log.Error(err)
		return "", err
	}
	log.Debug("backrest info ends")
	return "something", err

}
