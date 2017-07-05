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

package util

import (
	"github.com/crunchydata/postgres-operator/tpr"
	kerrors "k8s.io/client-go/pkg/api/errors"

	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"net/http"
)

// execute a sql policy against a cluster
func ExecPolicy(clientset *kubernetes.Clientset, tprclient *rest.RESTClient, namespace string, policyName string, clusterName string) error {
	//fetch the policy sql
	sqlString, err := GetPolicySQL(tprclient, namespace, policyName)
	if err != nil {
		return err
	}
	secretName := clusterName + "-pgroot-secret"
	//get the postgres user password
	var password string
	password, err = GetPasswordFromSecret(clientset, namespace, secretName)
	if err != nil {
		return err
	}
	//get the host ip address
	var service *v1.Service
	service, err = clientset.Services(namespace).Get(clusterName)
	if err != nil {
		log.Error(err)
		return err
	}

	//lastly, run the psql script
	log.Debugf("running psql password=%s ip=%s sql=[%s]\n", password, service.Spec.ClusterIP, sqlString)
	RunPsql(password, service.Spec.ClusterIP, sqlString)
	//labels[v] = "pgpolicy"

	return nil

}

func GetPolicySQL(tprclient *rest.RESTClient, namespace, policyName string) (string, error) {
	p := tpr.PgPolicy{}
	err := tprclient.Get().
		Resource(tpr.POLICY_RESOURCE).
		Namespace(namespace).
		Name(policyName).
		Do().
		Into(&p)
	if err == nil {
		if p.Spec.Repo != "" {
			return readSQLFromURL(p.Spec.Repo)
		} else {
			return p.Spec.Sql, err
		}
	} else if kerrors.IsNotFound(err) {
		log.Error("getPolicySQL policy not found using " + policyName)
		return "", err
	} else {
		log.Error(err)
		return "", err
	}
}

func readSQLFromURL(urlstring string) (string, error) {
	var bodyBytes []byte
	response, err := http.Get(urlstring)
	if err != nil {
		log.Error(err)
		return "", err
	} else {
		bodyBytes, err = ioutil.ReadAll(response.Body)
		if err != nil {
			log.Error(err)
			return "", err
		}

		defer response.Body.Close()
	}

	return string(bodyBytes), err

}

func ValidatePolicy(tprclient *rest.RESTClient, namespace string, policyName string) error {
	result := tpr.PgPolicy{}
	err := tprclient.Get().
		Resource(tpr.POLICY_RESOURCE).
		Namespace(namespace).
		Name(policyName).
		Do().
		Into(&result)
	if err == nil {
		log.Debug("pgpolicy " + policyName + " was validated")
	} else if kerrors.IsNotFound(err) {
		log.Debug("pgpolicy " + policyName + " not found fail validation")
	} else {
		log.Error("error getting pgpolicy " + policyName + err.Error())
	}
	return err
}
