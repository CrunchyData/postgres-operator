package smoketest

import (
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
	"testing"
)

func TestStatus(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil")
		}
	})

	t.Log("TestStatus starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo status", []string{"status"}, ""},
	}

	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		//t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "Operator Start")
		if !found {
			t.Error("could not find Operator Start in output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})

}

func TestShowConfig(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil")
		}
	})

	t.Log("TestShowConfig starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo show config", []string{"show", "config"}, ""},
	}
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		//t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "PrimaryStorage")
		if !found {
			t.Error("could not find PrimaryStorage in output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})

}
