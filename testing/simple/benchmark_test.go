package smoketest

import (
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestBenchmark(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil")
		}
	})

	t.Log("TestBenchmark starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo benchmark", []string{"benchmark", TestClusterName}, ""},
	}
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "Created")
		if !found {
			t.Error("Created not found in output")
		}

	}

	time.Sleep(time.Second * time.Duration(30))

	t.Log("TestShowBenchmark starts")

	tests = []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo show benchmark", []string{"show", "benchmark", TestClusterName}, ""},
	}
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		//t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "Results")
		if !found {
			t.Error("could not find Results in output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})

}
