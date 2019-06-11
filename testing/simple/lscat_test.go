package smoketest

import (
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
	"testing"
)

func TestLs(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil")
		}
	})

	t.Log("TestLs starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo ls", []string{"ls", TestClusterName}, ""},
	}
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		//t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "total")
		if !found {
			t.Error("total not found in pgo ls output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})

}

func TestCat(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil")
		}
	})

	t.Log("TestCat starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo cat", []string{"cat", TestClusterName, "/pgdata/foo/pg_hba.conf"}, ""},
	}
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		//t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "TYPE")
		if !found {
			t.Error("TYPE not found in pgo cat output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})
}
