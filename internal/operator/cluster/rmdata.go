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
	"context"
	"encoding/json"
	"os"
	"strconv"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RmdataJob struct {
	JobName        string
	ClusterName    string
	PGOImagePrefix string
	PGOImageTag    string
	//		SecurityContext string
	RemoveData   string
	RemoveBackup string
	IsBackup     string
	IsReplica    string
}

func CreateRmdataJob(clientset kubernetes.Interface, cl *crv1.Pgcluster, namespace string, removeData, removeBackup, isReplica, isBackup bool) error {
	ctx := context.TODO()
	var err error

	jobName := cl.Spec.Name + "-rmdata-" + util.RandStringBytesRmndr(4)

	jobFields := RmdataJob{
		JobName:        jobName,
		ClusterName:    cl.Spec.Name,
		PGOImagePrefix: util.GetValueOrDefault(cl.Spec.PGOImagePrefix, operator.Pgo.Pgo.PGOImagePrefix),
		PGOImageTag:    operator.Pgo.Pgo.PGOImageTag,
		RemoveData:     strconv.FormatBool(removeData),
		RemoveBackup:   strconv.FormatBool(removeBackup),
		IsBackup:       strconv.FormatBool(isReplica),
		IsReplica:      strconv.FormatBool(isBackup),
	}

	doc := bytes.Buffer{}

	if err := config.RmdatajobTemplate.Execute(&doc, jobFields); err != nil {
		log.Error(err.Error())
		return err
	}

	if operator.CRUNCHY_DEBUG {
		_ = config.RmdatajobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}

	if err := json.Unmarshal(doc.Bytes(), &newjob); err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return err
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_RMDATA,
		&newjob.Spec.Template.Spec.Containers[0])

	_, err = clientset.BatchV1().Jobs(namespace).
		Create(ctx, &newjob, metav1.CreateOptions{})
	return err
}
