package util

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"

	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const primaryClusterLabel = "master"

// ExecPolicy execute a sql policy against a cluster
func ExecPolicy(clientset *kubernetes.Clientset, restclient *rest.RESTClient, restconfig *rest.Config, namespace string, policyName string, serviceName string) error {
	//fetch the policy sql
	sql, err := GetPolicySQL(restclient, namespace, policyName)

	if err != nil {
		return err
	}

	// prepare the SQL string to be something that can be passed to a STDIN
	// interface
	stdin := strings.NewReader(sql)

	// now, we need to ensure we can get the Pod name of the primary PostgreSQL
	// instance. Thname being passed in is actually the "serviceName" of the Pod
	// We can isolate the exact Pod we want by using this (LABEL_SERVICE_NAME) and
	// the LABEL_PGHA_ROLE labels
	selector := fmt.Sprintf("%s=%s,%s=%s",
		config.LABEL_SERVICE_NAME, serviceName,
		config.LABEL_PGHA_ROLE, primaryClusterLabel)

	podList, err := kubeapi.GetPods(clientset, selector, namespace)

	if err != nil {
		return err
	} else if len(podList.Items) != 1 {
		msg := fmt.Sprintf("could not find the primary pod selector:[%s] pods returned:[%d]",
			selector, len(podList.Items))

		return errors.New(msg)
	}

	// get the primary Pod
	pod := podList.Items[0]

	// in the Pod spec, the first container is always the one with the PostgreSQL
	// instnace. We can use that to build out our execution call
	//
	// But first, let's prepare the command that will execute the SQL.
	// NOTE: this executes as the "postgres" user on the "postgres" database,
	// because that is what the existing functionality does
	//
	// However, unlike the previous implementation, this will connect over a UNIX
	// socket. There are certainly additional improvements that can be made, but
	// this gets us closer to what we want to do
	command := []string{
		"psql",
		"postgres",
		"postgres",
		"-f",
		"-",
	}

	// execute the command! if it fails, return the error
	if _, _, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		command, pod.Spec.Containers[0].Name, pod.Name, namespace, stdin); err != nil {
		return err
	}

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
		log.Debugf("pgpolicy %s was validated", policyName)
	} else if kerrors.IsNotFound(err) {
		log.Debugf("pgpolicy %s not found fail validation", policyName)
	} else {
		log.Error("error getting pgpolicy " + policyName + err.Error())
	}
	return err
}

// UpdatePolicyLabels ...
func UpdatePolicyLabels(clientset *kubernetes.Clientset, clusterName string, namespace string, newLabels map[string]string) error {

	deployment, found, err := kubeapi.GetDeployment(clientset, clusterName, namespace)
	if !found {
		return err
	}

	var patchBytes, newData, origData []byte
	origData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	accessor, err2 := meta.Accessor(deployment)
	if err2 != nil {
		return err2
	}

	objLabels := accessor.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}

	//update the deployment labels
	for key, value := range newLabels {
		objLabels[key] = value
	}
	log.Debugf("updated labels are %v\n", objLabels)

	accessor.SetLabels(objLabels)
	newData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	createdPatch := err == nil
	if err != nil {
		return err
	}
	if createdPatch {
		log.Debug("created merge patch")
	}

	_, err = clientset.AppsV1().Deployments(namespace).Patch(clusterName, types.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debug("error patching deployment " + err.Error())
	}
	return err

}
