package smoketest

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"fmt"
	"k8s.io/client-go/kubernetes"
	//	"k8s.io/client-go/rest"

	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCreateLabel(t *testing.T) {
	var TEST_KEY = "env"
	var TEST_VALUE = "smoketest"

	var clientset *kubernetes.Clientset
	//var restclient *rest.RESTClient
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()

	})

	t.Log("TestCreateLabel starts")

	labelString := "--label=" + TEST_KEY + "=" + TEST_VALUE
	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo label", []string{"label", TestClusterName, labelString}, ""},
	}

	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "applied")
		if !found {
			t.Error("applied not found in label output")
		}

		selector := config.LABEL_PG_CLUSTER + "=" + TestClusterName + "," + config.LABEL_SERVICE_NAME + "=" + TestClusterName
		deps, err := kubeapi.GetDeployments(clientset, selector, Namespace)
		if err != nil {
			t.Error(err.Error())
			os.Exit(2)
		}

		if len(deps.Items) != 1 {
			t.Error("nubmer of deployments was not 1")
			os.Exit(2)
		}

		primaryDeployment := deps.Items[0]
		t.Logf("dep name found was %s", primaryDeployment.Name)
		if primaryDeployment.ObjectMeta.Labels[TEST_KEY] != TEST_VALUE {
			t.Error("could not find label on deployment")
		}
		//fmt.Printf("%v was the labels on the dep", dep.ObjectMeta.Labels)
	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})
}
