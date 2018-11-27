package ingestservice

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
	"k8s.io/client-go/rest"
)

// CreateIngest ...
func CreateIngest(RESTClient *rest.RESTClient, request *msgs.CreateIngestRequest) msgs.CreateIngestResponse {

	resp := msgs.CreateIngestResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debugf("create ingest called for %s", request.Name)

	// error if it already exists
	result := crv1.Pgingest{}
	found, err := kubeapi.Getpgingest(apiserver.RESTClient,
		&result, request.Name, apiserver.Namespace)
	if !found {
		log.Debugf("pgingest %s not found so we will create it", request.Name)
	} else if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "error getting pgingest " + request.Name + err.Error()
		return resp
	} else {
		log.Debugf("pgingest %s was found so we will not create it", request.Name)
		resp.Status.Msg = "pingest " + request.Name + " was found so we will not create it"
		return resp
	}

	spec := crv1.PgingestSpec{}
	spec.Name = request.Name
	spec.WatchDir = request.WatchDir
	spec.DBHost = request.DBHost
	spec.DBPort = request.DBPort
	spec.DBName = request.DBName
	spec.DBSecret = request.DBSecret
	spec.DBTable = request.DBTable
	spec.DBColumn = request.DBColumn
	spec.MaxJobs = request.MaxJobs
	spec.PVCName = request.PVCName
	spec.SecurityContext = request.SecurityContext
	spec.Status = "just a start"

	newInstance := &crv1.Pgingest{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: request.Name,
		},
		Spec: spec,
		Status: crv1.PgingestStatus{
			State:   crv1.PgingestStateCreated,
			Message: "Created, not processed yet",
		},
	}

	err = kubeapi.Createpgingest(apiserver.RESTClient,
		newInstance, apiserver.Namespace)
	if err != nil {
		resp.Results = append(resp.Results, "error creating Pgingest "+err.Error())
	} else {
		resp.Results = append(resp.Results, "created Pgingest "+request.Name)
	}

	return resp

}

// ShowIngest ...
func ShowIngest(name string) msgs.ShowIngestResponse {
	response := msgs.ShowIngestResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if name == "all" {
		//get a list of all ingests
		ingestList := crv1.PgingestList{}
		err := kubeapi.Getpgingests(apiserver.RESTClient,
			&ingestList, apiserver.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		log.Debugf("ingests found len is %d", len(ingestList.Items))

		for _, i := range ingestList.Items {
			detail := msgs.ShowIngestResponseDetail{}
			detail.Ingest = i
			detail.JobCountRunning, detail.JobCountCompleted = getJobCounts(i.Name)
			response.Details = append(response.Details, detail)
		}
		return response
	} else {
		ingest := crv1.Pgingest{}
		_, err := kubeapi.Getpgingest(apiserver.RESTClient,
			&ingest, name, apiserver.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		detail := msgs.ShowIngestResponseDetail{}
		detail.Ingest = ingest
		detail.JobCountRunning, detail.JobCountCompleted = getJobCounts(name)
		response.Details = make([]msgs.ShowIngestResponseDetail, 1)
		response.Details[0] = detail
	}

	return response

}

// DeleteIngest ...
func DeleteIngest(name string) msgs.DeleteIngestResponse {
	response := msgs.DeleteIngestResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 1)

	if name == "all" {
		err := kubeapi.DeleteAllpgingest(apiserver.RESTClient,
			apiserver.Namespace)
		if err != nil {
			log.Error("error deleting all ingests" + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.Results[0] = "all"
	} else {
		err := kubeapi.Deletepgingest(apiserver.RESTClient,
			name, apiserver.Namespace)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.Results[0] = name
	}

	return response

}

func getJobCounts(ingestName string) (int, int) {
	var running, completed int

	selector := util.LABEL_INGEST + "=" + ingestName
	fieldselector := "status.phase=Succeeded"
	pods, err := kubeapi.GetPodsWithBothSelectors(apiserver.Clientset, selector, fieldselector, apiserver.Namespace)
	if err != nil {
		return 0, 0
	}
	log.Debugf("There are %d ingest load pods completed", len(pods.Items))
	completed = len(pods.Items)

	fieldselector = "status.phase!=Succeeded"
	pods, err = kubeapi.GetPodsWithBothSelectors(apiserver.Clientset, selector, fieldselector, apiserver.Namespace)
	if err != nil {
		return 0, 0
	}
	log.Debugf("There are %d ingest load pods running", len(pods.Items))
	running = len(pods.Items)

	return running, completed
}
