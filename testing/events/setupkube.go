package eventtest

import (
	"flag"
	"fmt"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	"os"
	//"os/exec"
	//"time"
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig      = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
	namespace       = flag.String("namespace", "pgouser1", "namespace to test within ")
	username        = flag.String("username", "pgouser1", "username to test within ")
	testclustername = flag.String("clustername", "", "cluster name to test with")

	Namespace       = "pgouser1"
	Username        = "pgouser1"
	TestClusterName = "foo"
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
	if *username != "" {
		Username = *username
	}

	fmt.Printf("running test in namespace %s\n", Namespace)
	fmt.Printf("running test as user %s\n", Username)
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
