package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"sort"
	//client "github.com/crunchydata/postgres-operator/client"
	"github.com/crunchydata/postgres-operator/kubeapi"
	//"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
)

type kv struct {
	Key   string
	Value int
}

func main() {
	flag.Parse()
	// uses the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	kubeClient, err2 := kubernetes.NewForConfig(config)
	if err2 != nil {
		panic(err2.Error())
	}
	if kubeClient != nil {
		log.Println("got kube client")
	}

	//restclient, _, err := client.NewClient(config)
	//if err != nil {
	//	panic(err)
	//}
	//log.Println("got rest client")

	events := GetEvents(kubeClient, "demo")
	log.Info("the results")
	for k, v := range events {
		log.Infof("%s [%d]", k, v)
	}

	var ss []kv
	for k, v := range events {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	for _, kv := range ss {
		log.Infof("%s, %d\n", kv.Key, kv.Value)
	}

}

func GetEvents(clientset *kubernetes.Clientset, namespace string) map[string]int {
	results := make(map[string]int)
	// GetDeployments gets a list of deployments using a label selector
	deps, err := kubeapi.GetDeployments(clientset, "", namespace)
	if err != nil {
		log.Error(err)
		return results
	}

	for _, dep := range deps.Items {

		for k, v := range dep.ObjectMeta.Labels {
			lv := k + "=" + v
			//log.Infof("%s", lv)
			if results[lv] == 0 {
				results[lv] = 1
			} else {
				results[lv] = results[lv] + 1
			}
		}

	}

	//log.Infof("%d deployments", len(deps.Items))
	return results
}
