package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/evanphx/json-patch"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
)

func main() {
	flag.Parse()
	// uses the current context in kubeconfig
	var namespace = "demo"
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
	depName := "eggs"
	var deployment *v1beta1.Deployment
	deployment, err = clientset.ExtensionsV1beta1().Deployments(namespace).Get(depName, meta_v1.GetOptions{})
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
	newLabels["policytest2"] = "jeffsays"

	for key, value := range newLabels {
		objLabels[key] = value
	}
	fmt.Printf("updated labels are %v\n", objLabels)

	accessor.SetLabels(objLabels)

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

	_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Patch(depName, types.MergePatchType, patchBytes, "")
	if err != nil {
		panic("error patching deployment " + err.Error())
	}

}
