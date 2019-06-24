package smoketest

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

func TestPool(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
	})

	t.Log("TestPool starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
		cmdoutput string
	}{
		{"pgo pool create", []string{"create", "pgpool", TestClusterName}, "", "pgpool added"},
		{"pgo pool delete", []string{"delete", "pgpool", TestClusterName}, "", "pgpool deleted"},
	}

	selector := config.LABEL_PG_CLUSTER + "=" + TestClusterName
	var deps *v1.DeploymentList
	var err error
	var SLEEP_SECS = 5
	var output []byte

	for _, tt := range tests {
		deps, err = kubeapi.GetDeployments(clientset, selector, Namespace)
		if err != nil {
			t.Error(err.Error())
		}
		beforeDepCount := len(deps.Items)
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

		t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, tt.cmdoutput)
		if !found {
			t.Error(tt.name+" string not found in output")
		}

		time.Sleep(time.Second * time.Duration(SLEEP_SECS))
		deps, err = kubeapi.GetDeployments(clientset, selector, Namespace)
		if err != nil {
			t.Error(err.Error())
		}
		afterDepCount := len(deps.Items)
		//should have more deps after create
		if (afterDepCount <= beforeDepCount && tt.name == "pgo pool create") {
			t.Errorf("deps after was %d", afterDepCount)
		}
		//should have less deps after delete
		if (afterDepCount >= beforeDepCount && tt.name == "pgo pool delete") {
			t.Errorf("deps after was %d", afterDepCount)
		}
		t.Logf("deps after %d", afterDepCount)
	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})

}
