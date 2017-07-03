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

package cluster

import (
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"
	"k8s.io/client-go/kubernetes"
	kerrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
	"os"
	"strings"
)

func ProcessPolicies(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, stopchan chan struct{}, namespace string) {

	lo := v1.ListOptions{LabelSelector: "pg-cluster,!replica"}
	fw, err := clientset.Deployments(namespace).Watch(lo)
	if err != nil {
		log.Error("fatal error in ProcessPolicies " + err.Error())
		os.Exit(2)
	}

	_, err4 := watch.Until(0, fw, func(event watch.Event) (bool, error) {
		log.Infof("got a processpolicies watch event %v\n", event.Type)

		switch event.Type {
		case watch.Added:
			//deployment := event.Object.(*v1beta1.Deployment)
			//log.Infof("deployment processpolicy added=%s\n", dep.Name)
		case watch.Deleted:
			//deployment := event.Object.(*v1beta1.Deployment)
			//log.Infof("deployment processpolicy deleted=%s\n", deployment.Name)
		case watch.Error:
			log.Infof("deployment processpolicy error event")
		case watch.Modified:
			deployment := event.Object.(*v1beta1.Deployment)
			//log.Infof("deployment processpolicy modified=%s\n", deployment.Name)
			log.Infof("status available replicas=%d\n", deployment.Status.AvailableReplicas)
			if deployment.Status.AvailableReplicas > 0 {
				applyPolicies(namespace, clientset, tprclient, deployment)
			}
		default:
			log.Infoln("processpolices unknown watch event %v\n", event.Type)
		}

		return false, nil
	})

	if err4 != nil {
		log.Error("error in ProcessPolicies " + err4.Error())
	}

}

func applyPolicies(namespace string, clientset *kubernetes.Clientset, tprclient *rest.RESTClient, dep *v1beta1.Deployment) {
	//get the tpr which holds the requested labels if any
	cl := tpr.PgCluster{}
	err := tprclient.Get().
		Resource("pgclusters").
		Namespace(namespace).
		Name(dep.Name).
		Do().
		Into(&cl)
	if err == nil {
	} else if kerrors.IsNotFound(err) {
		log.Error("could not get cluster in policy processing using " + dep.Name)
		return
	} else {
		log.Error("error in policy processing " + err.Error())
		return
	}

	if cl.Spec.Policies == "" {
		log.Debug("no policies to apply to " + dep.Name)
		return
	}
	log.Debug("policies to apply to " + dep.Name + " are " + cl.Spec.Policies)
	policies := strings.Split(cl.Spec.Policies, ",")

	//apply the policies
	var sqlString, password, secretName string
	labels := make(map[string]string)

	for _, v := range policies {
		//fetch the policy sql
		sqlString, err = getPolicySQL(tprclient, namespace, v)
		if err != nil {
			break
		}
		secretName = cl.Spec.Name + "-pgroot-secret"
		//get the postgres user password
		password, err = util.GetPasswordFromSecret(clientset, namespace, secretName)
		if err != nil {
			break
		}
		//get the host ip address
		service, err2 := clientset.Services(namespace).Get(cl.Spec.Name)
		if err2 != nil {
			log.Error(err2)
			break
		}

		//lastly, run the psql script
		log.Debugf("running psql password=%s ip=%s sql=[%s]\n", password, service.Spec.ClusterIP, sqlString)
		util.RunPsql(password, service.Spec.ClusterIP, sqlString)
		labels[v] = "pgpolicy"

	}

	//update the deployment's labels to show applied policies
	err = util.UpdateDeploymentLabels(clientset, dep.Name, namespace, labels)
	if err != nil {
		log.Error(err)
	}
}

func getPolicySQL(tprclient *rest.RESTClient, namespace, policyName string) (string, error) {
	p := tpr.PgPolicy{}
	err := tprclient.Get().
		Resource(tpr.POLICY_RESOURCE).
		Namespace(namespace).
		Name(policyName).
		Do().
		Into(&p)
	if err == nil {
		return p.Spec.Sql, err
	} else if kerrors.IsNotFound(err) {
		log.Error("getPolicySQL policy not found using " + policyName)
		return "", err
	} else {
		log.Error(err)
		return "", err
	}
}
