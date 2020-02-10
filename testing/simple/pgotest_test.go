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
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
	"testing"
)

func TestPGOTestCluster(t *testing.T) {
	var clientset *kubernetes.Clientset
	var cluster string

	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil")
		}
	})

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo test cluster", []string{"test", TestClusterName}, ""},
	}

	t.Log("PGOTestCluster starts")
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		// Example line reference
		// psql -p 5432 -h 10.97.101.79 -U postgres postgres is Working
		// psql -p 5432 -h 10.97.101.79 -U postgres postgres is NOT working

		actual := string(output)
		actual_lines := strings.Split(actual, "\n")

		t.Logf("actual %s- ", actual)
		cluster = "cluster : " + TestClusterName
		found := strings.Contains(actual, cluster)
		if !found {
			t.Error("could not find cluster : " + TestClusterName + "in output")
		}
		for i := 0; i < len(actual_lines); i++ {
			//t.Logf("%v actual lines %s- ", i, actual_lines[i])
			if strings.Contains(actual_lines[i], "NOT working") {
				t.Error("output error found : " + actual_lines[i])
			}
		}
	}
	t.Log("PGOTestCluster complete!")
}
