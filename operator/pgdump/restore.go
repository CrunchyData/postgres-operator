package pgdump

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
	"k8s.io/client-go/kubernetes"
)

type restorejobTemplateFields struct {
	JobName             string
	ClusterName         string
	ToClusterPVCName    string
	SecurityContext     string
	COImagePrefix       string
	COImageTag          string
	CommandOpts         string
	PITRTarget          string
	PgbackrestStanza    string
	PgbackrestDBPath    string
	PgbackrestRepo1Path string
	PgbackrestRepo1Host string
}

// Restore ...
func Restore(namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	log.Infof(" PgDump Restore not implemented %s, %s", namespace, task.Name)

}
