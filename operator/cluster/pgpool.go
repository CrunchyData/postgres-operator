package cluster

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
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	//"k8s.io/api/core/v1"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/watch"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"text/template"
)

type PgpoolTemplateFields struct {
	Name               string
	ClusterName        string
	SecretsName        string
	CCPImagePrefix     string
	CCPImageTag        string
	ContainerResources string
	Port               string
	PrimaryServiceName string
	ReplicaServiceName string
}

var pgpoolTemplate *template.Template

func init() {
	pgpoolTemplate = util.LoadTemplate("/operator-conf/pgpool-template.json")
}

const PGPOOL_SUFFIX = "-pgpool"

// ProcessPgpool ...
func AddPgpool(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, namespace, secretName string) {
	var doc bytes.Buffer
	var err error

	clusterName := cl.Spec.Name
	pgpoolName := clusterName + PGPOOL_SUFFIX
	log.Debug("adding a pgpool " + pgpoolName)

	//create the pgpool deployment
	fields := PgpoolTemplateFields{
		Name:           pgpoolName,
		ClusterName:    clusterName,
		CCPImagePrefix: operator.CCPImagePrefix,
		CCPImageTag:    cl.Spec.CCPImageTag,
		Port:           "5432",
		SecretsName:    secretName,
	}

	err = pgpoolTemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug(doc.String())

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(doc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling pgpool json into Deployment " + err.Error())
		return
	}

	var deploymentResult *v1beta1.Deployment
	deploymentResult, err = clientset.ExtensionsV1beta1().Deployments(namespace).Create(&deployment)
	if err != nil {
		log.Error("error creating pgpool Deployment " + err.Error())
		return
	}
	log.Info("created pgpool Deployment " + deploymentResult.Name + " in namespace " + namespace)

	//create a service for the pgpool
	svcFields := ServiceTemplateFields{}
	svcFields.Name = pgpoolName
	svcFields.ClusterName = clusterName
	svcFields.Port = "5432"

	err = CreateService(clientset, &svcFields, namespace)
	if err != nil {
		log.Error(err)
		return
	}
}

// DeletePgpool
func DeletePgpool(clientset *kubernetes.Clientset, clusterName, namespace string) {

	var delProp meta_v1.DeletionPropagation
	delOptions := meta_v1.DeleteOptions{}
	delProp = meta_v1.DeletePropagationBackground
	delOptions.PropagationPolicy = &delProp
	pgpoolDepName := clusterName + "-pgpool"

	log.Debug("deleting pgpool deployment " + pgpoolDepName)

	err := clientset.ExtensionsV1beta1().Deployments(namespace).Delete(pgpoolDepName, &delOptions)
	if err != nil {
		log.Error("error deleting Deployment " + pgpoolDepName + err.Error())
	}

	//delete the service name=<clustename>-pgpool

	err = clientset.Core().Services(namespace).Delete(pgpoolDepName,
		&meta_v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting pgpool Service " + err.Error())
	} else {
		log.Info("deleted pgpool service " + pgpoolDepName)
	}

}
