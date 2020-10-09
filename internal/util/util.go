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
	"math/rand"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())

}

// GetLabels ...
func GetLabels(name, clustername string, replica bool) string {
	var output string
	if replica {
		output += fmt.Sprintf("\"primary\": \"%s\",\n", "false")
	}
	output += fmt.Sprintf("\"name\": \"%s\",\n", name)
	output += fmt.Sprintf("\"pg-cluster\": \"%s\"\n", clustername)
	return output
}

//CurrentPrimaryUpdate prepares the needed data structures with the correct current primary value
//before passing them along to be patched into the current pgcluster CRD's annotations
func CurrentPrimaryUpdate(clientset pgo.Interface, cluster *crv1.Pgcluster, currentPrimary, namespace string) error {
	//create a new map
	metaLabels := make(map[string]string)
	//copy the relevant values into the new map
	for k, v := range cluster.ObjectMeta.Labels {
		metaLabels[k] = v
	}
	//update this map with the new deployment label
	metaLabels[config.LABEL_DEPLOYMENT_NAME] = currentPrimary

	//Update CRD with the current primary name and the new deployment to point to after the failover
	if err := PatchClusterCRD(clientset, metaLabels, cluster, currentPrimary, namespace); err != nil {
		log.Errorf("failoverlogic: could not patch pgcluster %s with the current primary", currentPrimary)
	}

	return nil
}

// PatchClusterCRD patches the pgcluster CRD with any updated labels, or an updated current
// primary annotation value. As this uses a JSON merge patch, it will only updates those
// values that are different between the old and new CRD values.
func PatchClusterCRD(clientset pgo.Interface, labelMap map[string]string, oldCrd *crv1.Pgcluster, currentPrimary, namespace string) error {
	patch := kubeapi.NewMergePatch()

	// update our pgcluster annotation with the correct current primary value
	patch.Add("metadata", "annotations", config.ANNOTATION_CURRENT_PRIMARY)(currentPrimary)
	patch.Add("metadata", "annotations", config.ANNOTATION_PRIMARY_DEPLOYMENT)(currentPrimary)

	// update the stored primary storage value to match the current primary and deployment name
	patch.Add("spec", "PrimaryStorage", "name")(currentPrimary)

	for k, v := range labelMap {
		if len(validation.IsQualifiedName(k)) == 0 && len(validation.IsValidLabelValue(v)) == 0 {
			patch.Add("metadata", "labels", k)(v)
		} else {
			log.Debugf("user label %s:%s does not meet Kubernetes label requirements and will not be used to label "+
				"pgcluster %s", k, v, oldCrd.Spec.Name)
		}
	}

	patchBytes, err := patch.Bytes()
	if err != nil {
		return err
	}

	log.Debug(string(patchBytes))

	_, err6 := clientset.CrunchydataV1().Pgclusters(namespace).Patch(oldCrd.Spec.Name, types.MergePatchType, patchBytes)

	return err6

}

// GetValueOrDefault checks whether the first value given is set. If it is,
// that value is returned. If not, the second, default value is returned instead
func GetValueOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

// GetSecretPassword ...
func GetSecretPassword(clientset kubernetes.Interface, db, suffix, Namespace string) (string, error) {

	var err error

	selector := "pg-cluster=" + db
	secrets, err := clientset.
		CoreV1().Secrets(Namespace).
		List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return "", err
	}

	log.Debugf("secrets for %s", db)
	secretName := db + suffix
	for _, s := range secrets.Items {
		log.Debugf("secret : %s", s.ObjectMeta.Name)
		if s.ObjectMeta.Name == secretName {
			log.Debug("pgprimary password found")
			return string(s.Data["password"][:]), err
		}
	}

	log.Error("primary secret not found for " + db)
	return "", errors.New("primary secret not found for " + db)

}

// RandStringBytesRmndr ...
func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// IsStringOneOf tests to see string testVal is included in the list
// of strings provided using acceptedVals
func IsStringOneOf(testVal string, acceptedVals ...string) bool {
	isOneOf := false
	for _, val := range acceptedVals {
		if testVal == val {
			isOneOf = true
			break
		}
	}
	return isOneOf
}

// SQLQuoteIdentifier quotes an "identifier" (e.g. a table or a column name) to
// be used as part of an SQL statement.
//
// Any double quotes in name will be escaped.  The quoted identifier will be
// case sensitive when used in a query.  If the input string contains a zero
// byte, the result will be truncated immediately before it.
//
// Implementation borrowed from lib/pq: https://github.com/lib/pq which is
// licensed under the MIT License
func SQLQuoteIdentifier(identifier string) string {
	end := strings.IndexRune(identifier, 0)

	if end > -1 {
		identifier = identifier[:end]
	}

	return `"` + strings.Replace(identifier, `"`, `""`, -1) + `"`
}

// SQLQuoteLiteral quotes a 'literal' (e.g. a parameter, often used to pass literal
// to DDL and other statements that do not accept parameters) to be used as part
// of an SQL statement.
//
// Any single quotes in name will be escaped. Any backslashes (i.e. "\") will be
// replaced by two backslashes (i.e. "\\") and the C-style escape identifier
// that PostgreSQL provides ('E') will be prepended to the string.
//
// Implementation borrowed from lib/pq: https://github.com/lib/pq which is
// licensed under the MIT License. Curiously, @jkatz and @cbandy were the ones
// who worked on the patch to add this, prior to being at Crunchy Data
func SQLQuoteLiteral(literal string) string {
	// This follows the PostgreSQL internal algorithm for handling quoted literals
	// from libpq, which can be found in the "PQEscapeStringInternal" function,
	// which is found in the libpq/fe-exec.c source file:
	// https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/interfaces/libpq/fe-exec.c
	//
	// substitute any single-quotes (') with two single-quotes ('')
	literal = strings.Replace(literal, `'`, `''`, -1)
	// determine if the string has any backslashes (\) in it.
	// if it does, replace any backslashes (\) with two backslashes (\\)
	// then, we need to wrap the entire string with a PostgreSQL
	// C-style escape. Per how "PQEscapeStringInternal" handles this case, we
	// also add a space before the "E"
	if strings.Contains(literal, `\`) {
		literal = strings.Replace(literal, `\`, `\\`, -1)
		literal = ` E'` + literal + `'`
	} else {
		// otherwise, we can just wrap the literal with a pair of single quotes
		literal = `'` + literal + `'`
	}
	return literal
}
