package main

import (
	"flag"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/util/json"
	//"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	//"k8s.io/client-go/pkg/runtime"
	//"k8s.io/client-go/pkg/runtime/serializer"

	//"k8s.io/client-go/pkg/api/unversioned"
	//"k8s.io/client-go/pkg/api/v1"
	//"k8s.io/client-go/rest"
	//"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/pkg/api/meta"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
)

func main() {
	flag.Parse()
	// uses the current context in kubeconfig
	var namespace = "default"
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	//get the deployment
	depName := "janky"
	var deployment *v1beta1.Deployment
	deployment, err = clientset.Deployments(namespace).Get(depName)
	if err != nil {
		panic(err.Error())
	} else {
		fmt.Println("got the deployment" + deployment.Name)
	}
	origData, err5 := json.Marshal(deployment)
	if err != nil {
		panic(err5)
	}

	accessor, err2 := meta.Accessor(deployment)
	if err2 != nil {
		panic(err2.Error())
	}

	objLabels := accessor.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}
	fmt.Printf("current labels are %v\n", objLabels)

	//update the deployment labels
	newLabels := make(map[string]string)
	newLabels["policytest2"] = "pgpolicy"

	for key, value := range newLabels {
		objLabels[key] = value
	}
	fmt.Printf("updated labels are %v\n", objLabels)

	accessor.SetLabels(objLabels)
	//accessor.SetResourceVersion("718408")

	newData, err4 := json.Marshal(deployment)
	if err != nil {
		panic(err4)
	}
	if len(newData) > 0 {
	}
	if len(origData) > 0 {
	}

	patchBytes, err6 := jsonpatch.CreateMergePatch(origData, newData)
	createdPatch := err6 == nil
	if err6 != nil {
		panic(err6.Error())
	}
	if len(patchBytes) > 0 {
	}
	if createdPatch {
		fmt.Println("created merge patch")
	}

	_, err = clientset.Deployments(namespace).Patch(depName, api.MergePatchType, patchBytes, "")
	if err != nil {
		panic("error patching deployment " + err.Error())
	}

}
