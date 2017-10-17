package util

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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	//"k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"net/http"
)

// ExecPolicy execute a sql policy against a cluster
func ExecPolicy(clientset *kubernetes.Clientset, restclient *rest.RESTClient, namespace string, policyName string, clusterName string) error {
	//fetch the policy sql
	sqlString, err := GetPolicySQL(restclient, namespace, policyName)
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
	options := meta_v1.GetOptions{}
	service, err = clientset.Core().Services(namespace).Get(clusterName, options)
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

// GetPolicySQL returns the SQL string from a policy
func GetPolicySQL(restclient *rest.RESTClient, namespace, policyName string) (string, error) {
	p := crv1.Pgpolicy{}
	err := restclient.Get().
		Name(policyName).
		Namespace(namespace).
		Resource(crv1.PgpolicyResourcePlural).
		Do().
		Into(&p)
	if err == nil {
		if p.Spec.URL != "" {
			return readSQLFromURL(p.Spec.URL)
		}
		return p.Spec.SQL, err
	}

	if kerrors.IsNotFound(err) {
		log.Error("getPolicySQL policy not found using " + policyName + " in namespace " + namespace)
	}
	log.Error(err)
	return "", err
}

// readSQLFromURL returns the SQL string from a URL
func readSQLFromURL(urlstring string) (string, error) {
	var bodyBytes []byte
	response, err := http.Get(urlstring)
	if err == nil {
		bodyBytes, err = ioutil.ReadAll(response.Body)
		defer response.Body.Close()
	}

	if err != nil {
		log.Error(err)
		return "", err
	}

	return string(bodyBytes), err

}

// ValidatePolicy tests to see if a policy exists
func ValidatePolicy(restclient *rest.RESTClient, namespace string, policyName string) error {
	result := crv1.Pgpolicy{}
	err := restclient.Get().
		Resource(crv1.PgpolicyResourcePlural).
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
