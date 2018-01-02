package task

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	"io/ioutil"
	v1batch "k8s.io/api/batch/v1"

	"bytes"
	"encoding/json"
	"k8s.io/client-go/kubernetes"
	"text/template"
)

type rmdatajobTemplateFields struct {
	Name            string
	PvcName         string
	COImagePrefix   string
	COImageTag      string
	SecurityContext string
	DataRoot        string
}

const rmdatajobPath = "/operator-conf/rmdata-job.json"

var rmdatajobTemplate *template.Template

func init() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(rmdatajobPath)
	if err != nil {
		log.Error("error in backup.go init " + err.Error())
		panic(err.Error())
	}
	rmdatajobTemplate = template.Must(template.New("rmdata job template").Parse(string(buf)))

}

// RemoveData ...
func RemoveData(namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	//create the Job to remove the data

	jobFields := rmdatajobTemplateFields{
		Name:            task.Spec.Name,
		PvcName:         task.Spec.Parameters,
		COImagePrefix:   operator.COImagePrefix,
		COImageTag:      operator.COImageTag,
		SecurityContext: util.CreateSecContext(task.Spec.StorageSpec.Fsgroup, task.Spec.StorageSpec.SupplementalGroups),
		DataRoot:        task.Spec.Name,
	}

	var doc2 bytes.Buffer
	err := rmdatajobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}
	jobDocString := doc2.String()
	log.Debug(jobDocString)

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return
	}

	resultJob, err := clientset.Batch().Jobs(namespace).Create(&newjob)
	if err != nil {
		log.Error("error creating Job " + err.Error())
		return
	}
	log.Info("created Job " + resultJob.Name)

}
