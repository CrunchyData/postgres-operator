package smoketest

import (
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var POLICY = "smoke-policy"

func TestCreatePolicy(t *testing.T) {
	var clientset *kubernetes.Clientset
	//var restClient *rest.RESTClient
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil")
		}
		TestDeletePolicy(t)
	})

	t.Log("TestCreatePolicy starts")

	pathToPolicy := getPath()

	if _, err := os.Stat(pathToPolicy); os.IsNotExist(err) {
		t.Error(pathToPolicy + "does not exist")

	}
	policyFlag := "--in-file=" + pathToPolicy

	t.Logf("looking in %s", pathToPolicy)

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo create policy", []string{"create", "policy", POLICY, policyFlag}, ""},
	}

	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "Created policy")
		if !found {
			t.Error("created policy string not found in output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})

}

func getPath() string {
	root := os.Getenv("PGOROOT")
	return root + "/examples/policy/xrayapp.sql"
}

func TestDeletePolicy(t *testing.T) {
	var clientset *kubernetes.Clientset
	var restClient *rest.RESTClient
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, restClient = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil")
		}
	})

	t.Log("TestDeletePolicy starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo delete policy", []string{"delete", "policy", POLICY, "--no-prompt"}, ""},
	}

	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)

		var found bool
		policy := crv1.Pgpolicy{}
		found, err = kubeapi.Getpgpolicy(restClient, &policy, POLICY, Namespace)
		if found {
			t.Logf(err.Error())
			t.Error("pgpolicy was found ")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})

}

func TestShowPolicy(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil")
		}
		TestCreatePolicy(t)
	})

	t.Log("TestShowPolicy starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo show policy", []string{"show", "policy", POLICY}, ""},
	}

	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "create table xrayapp")
		if !found {
			t.Log(actual)
			t.Error("show policy string not found in output")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})

}
