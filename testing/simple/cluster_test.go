package smoketest

import (
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
	"testing"
)

/**
func TestCreateCluster(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")

	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset = SetupKube()
	})

	t.Log("TestCreateCluster starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo", []string{"create", "cluster", "foo"}, ""},
	}

	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		//lines := strings.Split(string(actual), "\n")

		lo := meta_v1.ListOptions{}
		pods, err := clientset.CoreV1().Pods(Namespace).List(lo)
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Logf("pods found %d", len(pods.Items))
	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})

}
*/

func TestDf(t *testing.T) {
	//var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		//clientset = SetupKube()
	})
	t.Log("TestDf starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo df", []string{"df", TestClusterName}, ""},
	}

	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		//lines := strings.Split(string(actual), "\n")
		found := strings.Contains(actual, "PGSIZE")
		if !found {
			t.Error("could not find PGSIZE string in output")
		}
		found = strings.Contains(actual, "up")
		if !found {
			t.Error("could not find up string in output")
		}

		/**
		lo := meta_v1.ListOptions{}
		pods, err := clientset.CoreV1().Pods(Namespace).List(lo)
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Logf("pods found %d", len(pods.Items))
		*/
	}
	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})
}

func TestReload(t *testing.T) {
	//var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		//clientset = SetupKube()
	})
	t.Log("TestReload starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo reload", []string{"reload", TestClusterName, "--no-prompt"}, ""},
	}

	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		//lines := strings.Split(string(actual), "\n")

		found := strings.Contains(actual, "reload")
		if !found {
			t.Error("could not find reload string in output")
		}
	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})
}
