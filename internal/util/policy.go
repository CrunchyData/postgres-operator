package util

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"

	"io/ioutil"

	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// ExecPolicy execute a sql policy against a cluster
func ExecPolicy(clientset kubeapi.Interface, restconfig *rest.Config, namespace, policyName, serviceName, port string) error {
	//fetch the policy sql
	sql, err := GetPolicySQL(clientset, namespace, policyName)

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
		config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_PRIMARY)

	podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: selector})

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
		"-p",
		port,
		"postgres",
		"postgres",
		"-f",
		"-",
	}

	// execute the command! if it fails, return the error
	if _, stderr, err := kubeapi.ExecToPodThroughAPI(restconfig, clientset,
		command, pod.Spec.Containers[0].Name, pod.Name, namespace, stdin); err != nil || stderr != "" {
		// log the error from the pod and stderr, but return the stderr
		log.Error(err, stderr)

		return fmt.Errorf(stderr)
	}

	return nil
}

// GetPolicySQL returns the SQL string from a policy
func GetPolicySQL(clientset pgo.Interface, namespace, policyName string) (string, error) {
	p, err := clientset.CrunchydataV1().Pgpolicies(namespace).Get(policyName, metav1.GetOptions{})
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
func ValidatePolicy(clientset pgo.Interface, namespace string, policyName string) error {
	_, err := clientset.CrunchydataV1().Pgpolicies(namespace).Get(policyName, metav1.GetOptions{})
	if err == nil {
		log.Debugf("pgpolicy %s was validated", policyName)
	} else if kerrors.IsNotFound(err) {
		log.Debugf("pgpolicy %s not found fail validation", policyName)
	} else {
		log.Error("error getting pgpolicy " + policyName + err.Error())
	}
	return err
}
