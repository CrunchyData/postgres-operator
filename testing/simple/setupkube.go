package smoketest

import (
	"flag"
	"fmt"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"os"
	"os/exec"
	"time"
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	kubeconfig      = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
	namespace       = flag.String("namespace", "pgouser1", "namespace to test within ")
	testclustername = flag.String("clustername", "", "cluster name to test with")

	Namespace       = "pgouser1"
	TestClusterName = "foomatic"
	SLEEP_SECS      = 10
)

func SetupKube() (*kubernetes.Clientset, *rest.RESTClient) {
	var RESTClient *rest.RESTClient

	flag.Parse()

	if *namespace == "" {
		val := os.Getenv("PGO_NAMESPACE")
		if val == "" {
			fmt.Println("PGO_NAMESPACE env var is required for smoketest")
			os.Exit(2)
		}
	} else {
		Namespace = *namespace
	}
	if *testclustername != "" {
		TestClusterName = *testclustername
	}

	fmt.Printf("running test in namespace %s\n", Namespace)
	fmt.Printf("running test on cluster %s\n", TestClusterName)
	// uses the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	RESTClient, _, err = util.NewClient(config)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	//createTestCluster(clientset)
	verifyExists(RESTClient)

	return clientset, RESTClient
}

func createTestCluster(clientset *kubernetes.Clientset) {
	dep, _, err := kubeapi.GetDeployment(clientset, TestClusterName, Namespace)
	if kerrors.IsNotFound(err) {
		fmt.Printf("test cluster deployment not found will create: %s", TestClusterName)
		tests := []struct {
			name    string
			args    []string
			fixture string
		}{
			{"pgo", []string{"create", "cluster", TestClusterName}, ""},
		}

		for _, tt := range tests {
			cmd := exec.Command("pgo", tt.args...)
			output, err := cmd.CombinedOutput()
			actual := string(output)

			fmt.Printf("actual %s- ", actual)
			if err != nil {
				fmt.Printf(err.Error())
				os.Exit(2)
			}

		}

		fmt.Println("sleeping to give test cluster Deployment time to start..., TODO add proper wait")
		time.Sleep(time.Second * time.Duration(SLEEP_SECS))

	} else if err != nil {
		fmt.Println(err.Error())
		fmt.Printf("error in getting test cluster deployment: %s", dep.Name)
		os.Exit(2)
	} else {
		fmt.Printf("found test cluster deployment: %s", dep.Name)
	}

	//verify it is in a running state here and wait for a limited
	//time until it starts up

}

// buildConfig ...
func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func verifyExists(RESTClient *rest.RESTClient) {
	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(RESTClient, &cluster, TestClusterName, Namespace)
	if !found || err != nil {
		fmt.Printf("test cluster %s deployment not found can not continue", TestClusterName)
		os.Exit(2)
	}

	fmt.Printf("pgcluster %s is found\n", TestClusterName)

}
