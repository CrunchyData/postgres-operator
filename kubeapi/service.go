package kubeapi

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// DeleteService deletes a Service
func DeleteService(clientset *kubernetes.Clientset, name, namespace string) error {
	err := clientset.CoreV1().Services(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error deleting Service " + name)
	}
	log.Info("delete service " + name)
	return err
}

// GetServices gets a list of Services by selector
func GetServices(clientset *kubernetes.Clientset, selector, namespace string) (*v1.ServiceList, error) {

	lo := meta_v1.ListOptions{LabelSelector: selector}

	services, err := clientset.CoreV1().Services(namespace).List(lo)
	if err != nil {
		log.Error(err)
		log.Error("error getting services selector=[" + selector + "]")
		return services, err
	}

	return services, err
}

// GetService gets a Service by name
func GetService(clientset *kubernetes.Clientset, name, namespace string) (*v1.Service, bool, error) {
	svc, err := clientset.CoreV1().Services(namespace).Get(name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return svc, false, err
	}
	if err != nil {
		log.Error(err)
		return svc, false, err
	}

	return svc, true, err
}

// CreateService creates a Service
func CreateService(clientset *kubernetes.Clientset, svc *v1.Service, namespace string) (*v1.Service, error) {
	result, err := clientset.CoreV1().Services(namespace).Create(svc)
	if err != nil {
		log.Error(err)
		log.Error("error creating service " + svc.Name)
		return result, err
	}

	log.Info("created service " + result.Name)
	return result, err
}

func UpdateService(clientset *kubernetes.Clientset, svc *v1.Service, namespace string) error {
	_, err := clientset.CoreV1().Services(namespace).Update(svc)
	if err != nil {
		log.Error(err)
		log.Error("error updating service %s", svc.Name)
	}
	return err

}

// ServicePortPatchSpec holds the relevant information for making a JSON patch to
// add a port to a service
type ServicePortPatchSpec struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value ServicePortSpec `json:"value"`
}

// ServicePortSpec holds the specific port info needed when patching a service during a
// cluster upgrade and is part of the above ServicePortPatchSpec
type ServicePortSpec struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Protocol   string `json:"protocol"`
	TargetPort int    `json:"targetPort"`
}

// PortPatch returns a struct to use when patching in a new port to a service.
func PortPatch(name, protocol string, port, targetport int) []ServicePortSpec {
	portPatch := make([]ServicePortSpec, 1)
	portPatch[0].Name = name
	portPatch[0].Port = port
	portPatch[0].Protocol = protocol
	portPatch[0].TargetPort = targetport

	return portPatch
}

// ServiceSelectorPatchSpec holds the relevant selector information used when making a JSON patch on a service
type ServiceSelectorPatchSpec struct {
	Op    string              `json:"op"`
	Path  string              `json:"path"`
	Value ServiceSelectorSpec `json:"value"`
}

// ServiceSelectorSpec holds the information needed for selecting the appropriate service
// being used by a pgcluster that we want to patch
type ServiceSelectorSpec struct {
	Pgcluster string `json:"pg-cluster"`
	Role      string `json:"role"`
}

// SelectorPatches returns the needed selector struct used when patching a pgcluster's service
func SelectorPatches(servicename, role string) []ServiceSelectorSpec {
	selectorPatch := make([]ServiceSelectorSpec, 2)
	selectorPatch[0].Pgcluster = servicename
	selectorPatch[0].Role = role

	return selectorPatch
}

// PatchServicePort performs a JSON patch on a service to modify the ports defined for a given service
// As it performs a JSON patch, the supported operations should include: “add”, “remove”, “replace”, “move”, “copy” and “test”
func PatchServicePort(clientset *kubernetes.Clientset, servicename, namespace, op, jsonpath string, portpatch ServicePortSpec) {
	var patchBytes []byte
	var err error

	patch := make([]ServicePortPatchSpec, 1)
	patch[0].Op = op
	patch[0].Path = jsonpath
	patch[0].Value = portpatch

	patchBytes, err = json.Marshal(patch)
	if err != nil {
		log.Error("error in converting patch " + err.Error())
	} else {
		applyServicePatch(clientset, namespace, servicename, patchBytes)
	}
}

// PatchServiceSelector replaces the selector section of a service definition with the selector patch provided.
// As it performs a JSON patch, the supported operations should include: “add”, “remove”, “replace”, “move”, “copy” and “test”
func PatchServiceSelector(clientset *kubernetes.Clientset, servicename, namespace, op, jsonpath string, selectorpatch ServiceSelectorSpec) {
	var patchBytes []byte
	var err error

	patch := make([]ServiceSelectorPatchSpec, 1)
	patch[0].Op = op
	patch[0].Path = jsonpath
	patch[0].Value = selectorpatch

	patchBytes, err = json.Marshal(patch)
	if err != nil {
		log.Error("error in converting patch " + err.Error())
	} else {
		//apply the service
		applyServicePatch(clientset, namespace, servicename, patchBytes)
	}
}

// applyServicePatch performs a JSON patch with the patch information provided.
func applyServicePatch(clientset *kubernetes.Clientset, namespace, servicename string, patchBytes []byte) {
	log.Info("patch Service " + servicename)
	if _, err := clientset.CoreV1().Services(namespace).Patch(servicename, types.JSONPatchType, patchBytes); err != nil {
		log.Error(err)
		log.Error("error patching Service " + servicename)
	}
}
