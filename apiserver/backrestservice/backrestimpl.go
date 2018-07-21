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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

		err = kubeapi.Createpgtask(apiserver.RESTClient, getBackupParams(clusterName, taskName), apiserver.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "created Pgtask "+taskName)

	}

	return resp
}

func getBackupParams(clusterName, taskName string) *crv1.Pgtask {
	var newInstance *crv1.Pgtask

	spec := crv1.PgtaskSpec{}
	spec.Name = taskName
	spec.TaskType = crv1.PgtaskBackrestBackup
	spec.Parameters = make(map[string]string)
	spec.Parameters[util.LABEL_PG_CLUSTER] = clusterName

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
