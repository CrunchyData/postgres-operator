package task

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

type rmdatajobTemplateFields struct {
	Name            string
	PvcName         string
	ClusterName     string
	COImagePrefix   string
	COImageTag      string
	SecurityContext string
	DataRoot        string
}

// RemoveData ...
func RemoveData(namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	//create the Job to remove the data
	//in this case the pvcname is the key value in the map
	//map holds [volumename] = [pvcname]
	//in the case of multiple volumes (pgdata and pgwal) we iterate

	var pvcName string
	for k, v := range task.Spec.Parameters {
		pvcName = v

		jobFields := rmdatajobTemplateFields{
			Name:            task.Spec.Name + "-" + k,
			ClusterName:     task.Spec.Name,
			PvcName:         pvcName,
			COImagePrefix:   operator.Pgo.Pgo.COImagePrefix,
			COImageTag:      operator.Pgo.Pgo.COImageTag,
			SecurityContext: util.CreateSecContext(task.Spec.StorageSpec.Fsgroup, task.Spec.StorageSpec.SupplementalGroups),
			DataRoot:        task.Spec.Name,
		}
		log.Debugf("creating rmdata job for pvc [%s]", pvcName)

		var doc2 bytes.Buffer
		err := operator.RmdatajobTemplate.Execute(&doc2, jobFields)
		if err != nil {
			log.Error(err.Error())
			return
		}

		if operator.CRUNCHY_DEBUG {
			operator.RmdatajobTemplate.Execute(os.Stdout, jobFields)
		}

		newjob := v1batch.Job{}
		err = json.Unmarshal(doc2.Bytes(), &newjob)
		if err != nil {
			log.Error("error unmarshalling json into Job " + err.Error())
			return
		}

		kubeapi.CreateJob(clientset, &newjob, namespace)
	}

}
