package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	clientset "github.com/crunchydata/postgres-operator/client"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"time"
)

var (
	kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
)

type AutoFailoverTask struct {
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

	restclient, _, err := clientset.NewClient(config)
	if err != nil {
		panic(err)
	}
	log.Println("got rest client")

	thing := AutoFailoverTask{}
	thing.AddEvent(restclient, "doggie", "NotReady", "demo")
	thing.Print(restclient, "demo")
	events := thing.GetEvents(restclient, "doggie", "demo")
	log.Infof("%v are the events\n", events)

	thing.Clear(restclient, "doggie", "demo")

}

func (*AutoFailoverTask) AddEvent(restclient *rest.RESTClient, clusterName, eventType, namespace string) {
	var err error
	var found bool

	taskName := clusterName + "-" + util.LABEL_AUTOFAIL
	task := crv1.Pgtask{}
	found, err = kubeapi.Getpgtask(restclient, &task, taskName, namespace)
	if !found {
		task.Name = taskName
		task.ObjectMeta.Labels = make(map[string]string)
		task.ObjectMeta.Labels[util.LABEL_AUTOFAIL] = "true"
		task.Spec.TaskType = crv1.PgtaskAutoFailover
		task.Spec.Parameters = make(map[string]string)
		task.Spec.Parameters[time.Now().String()] = eventType
		err = kubeapi.Createpgtask(restclient, &task, namespace)
		return
	}

	task.Spec.Parameters[time.Now().String()] = eventType
	err = kubeapi.Updatepgtask(restclient, &task, taskName, namespace)
	if err != nil {
		log.Error(err)
	}

}

func (*AutoFailoverTask) Clear(restclient *rest.RESTClient, clusterName, namespace string) {

	taskName := clusterName + "-" + util.LABEL_AUTOFAIL
	kubeapi.Deletepgtask(restclient, taskName, namespace)
}

func (*AutoFailoverTask) Print(restclient *rest.RESTClient, namespace string) {

	log.Infoln("GlobalFailoverMap....")

	tasklist := crv1.PgtaskList{}

	err := kubeapi.GetpgtasksBySelector(restclient, &tasklist, util.LABEL_AUTOFAIL, namespace)
	if err != nil {
		log.Error(err)
		return

	}
	for k, v := range tasklist.Items {
		log.Infof("k=%s v=%v tasktype=%s\n", k, v.Name, v.Spec.TaskType)
		for x, y := range v.Spec.Parameters {
			log.Infof("parameter %s %s\n", x, y)
		}
	}

}

func (*AutoFailoverTask) Exists(restclient *rest.RESTClient, clusterName, namespace string) bool {
	task := crv1.Pgtask{}
	taskName := clusterName + "-" + util.LABEL_AUTOFAIL
	found, _ := kubeapi.Getpgtask(restclient, &task, taskName, namespace)
	return found
}

func (*AutoFailoverTask) GetEvents(restclient *rest.RESTClient, clusterName, namespace string) map[string]string {
	task := crv1.Pgtask{}
	taskName := clusterName + "-" + util.LABEL_AUTOFAIL
	found, _ := kubeapi.Getpgtask(restclient, &task, taskName, namespace)
	if found {
		return task.Spec.Parameters
	}
	return make(map[string]string)
}

func (*AutoFailoverTask) GetAutoFailoverTasks(restclient *rest.RESTClient, namespace string) {

	tasklist := crv1.PgtaskList{}

	err := kubeapi.GetpgtasksBySelector(restclient, &tasklist, util.LABEL_AUTOFAIL, namespace)
	if err != nil {
		log.Error(err)
		return

	}

	for k, v := range tasklist.Items {
		for tasktime, status := range v.Spec.Parameters {
			log.Infof("parameter time: %s status: %s\n", tasktime, status)
		}
	}

}
