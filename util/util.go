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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"

	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())

}

// GetPodSecurityContext will generate the security context required for a
// Deployment by incorporating the standard fsGroup for the user that runs the
// container (typically the "postgres" user), and adds any supplemental groups
// that may need to be added, e.g. for NFS storage.
//
// Following the legacy method, this returns a JSON string, which will be
// modified in the future. Mainly this is transitioning from the legacy function
// by adding the expected types
func GetPodSecurityContext(supplementalGroups []int64) string {
	// set up the security context struct
	securityContext := v1.PodSecurityContext{
		// we store the PostgreSQL FSGroup in this constant as an int64, so it's just
		// carried over
		FSGroup: &crv1.PGFSGroup,
		// add any supplemental groups that the user passed in
		SupplementalGroups: supplementalGroups,
	}

	// ...convert to JSON. Errors are ignored
	doc, err := json.Marshal(securityContext)

	// if there happens to be an error, warn about it
	if err != nil {
		log.Warn(err)
	}

	// for debug purposes, we can look at the document
	log.Debug(doc)

	// return a string of the security context
	return string(doc)
}

// ThingSpec is a json patch structure
type ThingSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// Patch will patch a particular resource
func Patch(restclient *rest.RESTClient, path string, value string, resource string, name string, namespace string) error {

	things := make([]ThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = path
	things[0].Value = value

	patchBytes, err4 := json.Marshal(things)
	if err4 != nil {
		log.Error("error in converting patch " + err4.Error())
	}
	log.Debug(string(patchBytes))

	_, err6 := restclient.Patch(types.JSONPatchType).
		Namespace(namespace).
		Resource(resource).
		Name(name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}

// CreatePVCSnippet generates the PVC json snippet
func CreatePVCSnippet(storageType string, PVCName string) string {

	var sc bytes.Buffer

	if storageType != "emptydir" {
		sc.WriteString("\"persistentVolumeClaim\": {\n")
		sc.WriteString("\t \"claimName\": \"" + PVCName + "\"")
		sc.WriteString("\n")
	} else {
		sc.WriteString("\"emptyDir\": {")
		sc.WriteString("\n")
	}

	sc.WriteString("}")

	return sc.String()
}

// CreateBackupPVCSnippet generates the PVC definition fragment
func CreateBackupPVCSnippet(backupPVCName string) string {

	var sc bytes.Buffer

	if backupPVCName != "" {
		sc.WriteString("\"persistentVolumeClaim\": {\n")
		sc.WriteString("\t \"claimName\": \"" + backupPVCName + "\"")
		sc.WriteString("\n")
	} else {
		sc.WriteString("\"emptyDir\": {")
		sc.WriteString("\n")
	}

	sc.WriteString("}")

	return sc.String()
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

// PatchClusterCRD patches the pgcluster CRD
func PatchClusterCRD(restclient *rest.RESTClient, labelMap map[string]string, oldCrd *crv1.Pgcluster, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
	if err != nil {
		return err
	}

	if oldCrd.ObjectMeta.Labels == nil {
		oldCrd.ObjectMeta.Labels = make(map[string]string)
	}
	for k, v := range labelMap {
		if len(validation.IsQualifiedName(k)) == 0 && len(validation.IsValidLabelValue(v)) == 0 {
			oldCrd.ObjectMeta.Labels[k] = v
		} else {
			log.Debugf("user label %s:%s does not meet Kubernetes label requirements and will not be used to label "+
				"pgcluster %s", k, v, oldCrd.Spec.Name)
		}
	}

	var newData, patchBytes []byte
	newData, err = json.Marshal(oldCrd)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}

	log.Debug(string(patchBytes))

	_, err6 := restclient.Patch(types.MergePatchType).
		Namespace(namespace).
		Resource(crv1.PgclusterResourcePlural).
		Name(oldCrd.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}

// GetSecretPassword ...
func GetSecretPassword(clientset *kubernetes.Clientset, db, suffix, Namespace string) (string, error) {

	var err error

	selector := "pg-cluster=" + db
	secrets, err := kubeapi.GetSecrets(clientset, selector, Namespace)
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

// CreateBackrestPVCSnippet
func CreateBackrestPVCSnippet(backRestPVCName string) string {

	var sc bytes.Buffer

	if backRestPVCName != "" {
		sc.WriteString("\"persistentVolumeClaim\": {\n")
		sc.WriteString("\t \"claimName\": \"" + backRestPVCName + "\"")
		sc.WriteString("\n")
	} else {
		sc.WriteString("\"emptyDir\": {")
		sc.WriteString("\"medium\": \"Memory\"")
		sc.WriteString("\n")
	}

	sc.WriteString("}")

	return sc.String()
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
