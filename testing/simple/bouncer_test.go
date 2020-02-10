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
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestBouncer(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
	})

	t.Log("TestBouncer starts")

	tests := []struct {
		name      string
		args      []string
		fixture   string
		cmdoutput string
	}{
		{"pgo bouncer create", []string{"create", "pgbouncer", TestClusterName}, "", "pgbouncer added"},
		{"pgo bouncer delete", []string{"delete", "pgbouncer", TestClusterName, "--no-prompt"}, "", "pgbouncer deleted"},
	}

	selector := config.LABEL_PG_CLUSTER + "=" + TestClusterName
	var deps *v1.DeploymentList
	var err error
	var SLEEP_SECS = 5
	var output []byte
	beforeDepCount := 0
	afterDepCount := 0

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			deps, err = kubeapi.GetDeployments(clientset, selector, Namespace)
			if err != nil {
				t.Error(err.Error())
			}
			beforeDepCount = len(deps.Items)
			if beforeDepCount == 0 {
				t.Error("deps before was zero")
			}
			t.Logf("deps before %d", beforeDepCount)

			cmd := exec.Command("pgo", tt.args...)
			output, err = cmd.CombinedOutput()
			if err != nil {
				//t.Fatal(err)
			}

			actual := string(output)

			t.Logf("actual command response: %s- ", actual)
			found := strings.Contains(actual, tt.cmdoutput)
			if !found {
				t.Error(tt.name + " string not found in output")
			}

			MAX_TRIES := 10
			complete := false
			for i := 1; i <= MAX_TRIES; i++ {
				time.Sleep(time.Second * time.Duration(SLEEP_SECS))
				t.Logf("sleeping while job completes, attempt #%d", i)
				deps, err = kubeapi.GetDeployments(clientset, selector, Namespace)
				if err != nil {
					t.Error(err.Error())
				}
				afterDepCount = len(deps.Items)
				//should have more deps after create
				if afterDepCount > beforeDepCount && strings.Contains(tt.name, "create") {
					t.Log(tt.name + " complete!")
					complete = true
					break
				}
				//should have less deps after delete
				if afterDepCount < beforeDepCount && strings.Contains(tt.name, "delete") {
					t.Log(tt.name + " complete!")
					complete = true
					break
				}
			}
			if !complete {
				t.Errorf("%v job did not succeed after retries. Deps after was %d", tt.name, afterDepCount)
			}
			t.Logf("deps after %d", afterDepCount)

		})

	}
}
