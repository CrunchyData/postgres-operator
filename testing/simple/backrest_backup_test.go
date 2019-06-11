package smoketest

import (
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestBackrestBackup(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()
		if clientset == nil {
			t.Error("clientset is nil ")
		}

		//remove any existing jobs
		selector := config.LABEL_PG_CLUSTER + "=" + TestClusterName
		err := kubeapi.DeleteJobs(clientset, selector, Namespace)
		if err != nil {
			t.Error(err.Error())
		}
		time.Sleep(time.Second * time.Duration(5))
	})

	t.Log("TestBackrestBackup starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo backup", []string{"backup", TestClusterName}, ""},
	}
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "created")
		if !found {
			t.Error("created not found in output")
		}

		SLEEP_SECS := 5
		time.Sleep(time.Second * time.Duration(SLEEP_SECS))
		//wait for the job to complete
		jobName := "backrest-backup-" + TestClusterName
		var job *v1batch.Job
		job, found = kubeapi.GetJob(clientset, jobName, Namespace)
		if !found {
			t.Errorf("could not find job %s", jobName)
		}

		MAX_TRIES := 20
		found = false
		for i := 0; i < MAX_TRIES; i++ {
			time.Sleep(time.Second * time.Duration(SLEEP_SECS))
			fmt.Println("sleeping while backrest job runs to completion")
			job, found = kubeapi.GetJob(clientset, jobName, Namespace)
			if !found {
				t.Errorf("could not find job %s", jobName)
			}
			if job.Status.Succeeded > 0 {
				fmt.Println("backrest job succeeded")
				found = true
				break
			}
		}
		if !found {
			t.Error("backup job did not succeed after retries")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})
}

func TestBackrestShow(t *testing.T) {
	var clientset *kubernetes.Clientset
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		clientset, _ = SetupKube()

		if clientset == nil {
			t.Error("clientset is nil")
		}
	})

	t.Log("TestBackrestShow starts")

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"pgo show backup", []string{"show", "backup", TestClusterName}, ""},
	}
	for _, tt := range tests {
		cmd := exec.Command("pgo", tt.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//t.Fatal(err)
		}

		actual := string(output)

		//t.Logf("actual %s- ", actual)
		found := strings.Contains(actual, "stanza")
		if !found {
			t.Error("could not find stanza in outout")
		}
		found = strings.Contains(actual, "error")
		if found {
			t.Error("error found in outout")
		}

	}

	t.Run("teardown", func(t *testing.T) {
		t.Log("some teardown code")
	})
}
