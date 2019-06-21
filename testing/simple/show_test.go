package smoketest

import (
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
	"testing"
)

func TestShowCluster(t *testing.T) {
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
		{"pgo show cluster", []string{"show", "cluster", TestClusterName}, ""},
	}

	t.Log("TestShowCluster starts")
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		cluster = "cluster : " + TestClusterName
		found := strings.Contains(actual, cluster)
		if !found {
			t.Error("could not find cluster : " + TestClusterName + "in output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})
}

func TestShowNamespace(t *testing.T) {
	var clientset *kubernetes.Clientset

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
		{"pgo show namespace", []string{"show", "namespace"}, ""},
	}

	t.Log("TestShowNamespace starts")
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, Namespace)
		if !found {
			t.Error("could not find " + Namespace  + "namespace in output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})
}


func TestShowPvc(t *testing.T) {
	var clientset *kubernetes.Clientset

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
		{"pgo show pvc", []string{"show", "pvc", TestClusterName}, ""},
	}

	t.Log("TestShowPvc starts")
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "pgdata")
		if !found {
			t.Error("could not find pgdata in output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})
}

