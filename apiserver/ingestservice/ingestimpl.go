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
	"k8s.io/client-go/rest"
	//	"github.com/crunchydata/postgres-operator/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateIngest ...
func CreateIngest(RESTClient *rest.RESTClient, request *msgs.CreateIngestRequest) msgs.CreateIngestResponse {
	var err error

	resp := msgs.CreateIngestResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	log.Debug("create ingest called for " + request.Name)

	// error if it already exists
	result := crv1.Pgingest{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgingestResourcePlural).
		Namespace(apiserver.Namespace).
		Name(request.Name).
		Do().
		Into(&result)
	if err == nil {
		log.Debug("pgingest " + request.Name + " was found so we will not create it")
		resp.Status.Msg = "pingest " + request.Name + " was found so we will not create it"
		return resp
	} else if kerrors.IsNotFound(err) {
		log.Debug("pgingest " + request.Name + " not found so we will create it")
	} else {
		log.Error("error getting pgingest " + request.Name + err.Error())
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "error getting pgingest " + request.Name + err.Error()
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

	err = apiserver.RESTClient.Post().
		Resource(crv1.PgingestResourcePlural).
		Namespace(apiserver.Namespace).
		Body(newInstance).
		Do().Into(&result)
	if err != nil {
		log.Error(" in creating Pgingest instance" + err.Error())
	}
	resp.Results = append(resp.Results, "created Pgingest "+request.Name)

	return resp

}
