package fake

/*
 Copyright 2020 - 2023 Crunchy Data Solutions, Inc.
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
	"io/ioutil"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekube "k8s.io/client-go/kubernetes/fake"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	fakecrunchy "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned/fake"
)

const (
	// defaultPGOInstallationName is the default installation name for a fake PGO client
	defaultPGOInstallationName = "test"
	// pgoNamespace is the default operator namespace for a fake PGO client
	defaultPGONamespace = "pgo"
	// defaultTargetNamespaces are the default target namespaces for a fake PGO client
	defaultTargetNamespaces = "pgouser1,pgouser2"
)

var (
	// pgoRoot represents the root of the PostgreSQL Operator project repository
	pgoRoot = os.Getenv("PGOROOT")
	// templatePath defines the default location for the PostgreSQL Operator templates relative to
	// env var PGOROOT
	templatePath = pgoRoot + "/installers/ansible/roles/pgo-operator/files/pgo-configs/"
	// pgoYAMLPath defines the default location for the default pgo.yaml configuration file
	// relative to env var PGOROOT
	pgoYAMLPath = pgoRoot + "/conf/postgres-operator/pgo.yaml"
)

// NewFakePGOClient creates a fake PostgreSQL Operator client.  Specifically, it creates
// a fake client containing a 'pgo-config' ConfigMap as needed to initialize the Operator
// (i.e. call the 'operator' packages 'Initialize()' function).  This allows for the proper
// initialization of the Operator in various unit tests where the various resources loaded
// during initialization (e.g. templates, config and/or global variables) are required.
func NewFakePGOClient() (kubeapi.Interface, error) {
	if pgoRoot == "" {
		return nil, errors.New("Environment variable PGOROOT must be set to the root directory " +
			"of the PostgreSQL Operator project repository in order to create a fake client")
	}

	os.Setenv("CRUNCHY_DEBUG", "false")
	os.Setenv("NAMESPACE", defaultTargetNamespaces)
	os.Setenv("PGO_INSTALLATION_NAME", defaultPGOInstallationName)
	os.Setenv("PGO_OPERATOR_NAMESPACE", defaultPGONamespace)

	// create a fake 'pgo-config' ConfigMap containing the operator templates and pgo.yaml
	pgoConfig, err := createMockPGOConfigMap(defaultPGONamespace)
	if err != nil {
		return nil, err
	}

	// now create and return a fake client containing the ConfigMap
	return &Clientset{
		Clientset:    fakekube.NewSimpleClientset(pgoConfig),
		PGOClientset: fakecrunchy.NewSimpleClientset(),
	}, nil
}

// createMockPGOConfigMap creates a mock 'pgo-config' ConfigMap containing the default pgo.yaml
// and templates included in the PostgreSQL Operator project repository.  This ConfigMap can be
// utilized when testing to similate and environment containing the various PostgreSQL Operator
// configuration files (e.g. templates) required to run the Operator.
func createMockPGOConfigMap(pgoNamespace string) (*v1.ConfigMap, error) {
	// create a configMap that will hold the default configs
	pgoConfigMap := &v1.ConfigMap{
		Data: make(map[string]string),
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.CustomConfigMapName,
			Namespace: pgoNamespace,
		},
	}

	// get all templates from the default template directory
	templates, err := ioutil.ReadDir(templatePath)
	if err != nil {
		return nil, err
	}

	// grab all file content so that it can be added to the ConfigMap
	fileContent := make(map[string]string)
	for _, t := range templates {
		content, err := ioutil.ReadFile(templatePath + t.Name())
		if err != nil {
			return nil, err
		}
		fileContent[t.Name()] = string(content)
	}

	// add the default pgo.yaml
	pgoContent, err := ioutil.ReadFile(pgoYAMLPath)
	if err != nil {
		return nil, err
	}
	fileContent["pgo.yaml"] = string(pgoContent)

	pgoConfigMap.Data = fileContent

	return pgoConfigMap, nil
}
