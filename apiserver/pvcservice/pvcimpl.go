package pvcservice

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	"io"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

type lspvcTemplateFields struct {
	Name          string
	ClusterName   string
	COImagePrefix string
	COImageTag    string
	BackupRoot    string
	PVCName       string
}

// ShowPVC ...
func ShowPVC(pvcName, PVCRoot string) ([]string, error) {
	pvcList := make([]string, 1)

	if pvcName == "all" {
		selector := util.LABEL_PGREMOVE + "=true"

		pvcs, err := kubeapi.GetPVCs(apiserver.Clientset, selector, apiserver.Namespace)
		if err != nil {
			return pvcList, err
		}

		log.Debugf("got %d PVCs from ShowPVC query\n", len(pvcs.Items))
		for _, p := range pvcs.Items {
			pvcList = append(pvcList, p.Name)
		}
		return pvcList, err
	}

	pvc, _, err := kubeapi.GetPVC(apiserver.Clientset, pvcName, apiserver.Namespace)
	if err != nil {
		return pvcList, err
	}

	log.Debug("\nPVC %s\n", pvc.Name+" is found")
	pvcList, err = printPVCListing(pvc.ObjectMeta.Labels[util.LABEL_PG_CLUSTER], pvc.Name, PVCRoot)

	return pvcList, err

}

// printPVCListing ...
func printPVCListing(clusterName, pvcName, PVCRoot string) ([]string, error) {
	newlines := make([]string, 1)
	var err error
	var doc2 bytes.Buffer
	var podName = "lspvc-" + pvcName

	//delete lspvc pod if it was not deleted for any reason prior
	_, found, err := kubeapi.GetPod(apiserver.Clientset, podName, apiserver.Namespace)
	if !found {
		//
	} else if err != nil {
		log.Error(err.Error())
		return newlines, err
	} else {
		log.Debug("deleting prior pod " + podName)
		err = kubeapi.DeletePod(apiserver.Clientset, podName, apiserver.Namespace)
		if err != nil {
			return newlines, err
		}
		//sleep a bit for the pod to be deleted
		time.Sleep(2000 * time.Millisecond)
	}

	pvcRoot := "/"
	if PVCRoot != "" {
		log.Debug("using " + PVCRoot + " as the PVC listing root")
		pvcRoot = PVCRoot
		log.Debug(pvcName + "/" + pvcRoot)
	} else {
		log.Debug(pvcName)
	}

	pvcFields := lspvcTemplateFields{
		Name:          podName,
		ClusterName:   clusterName,
		COImagePrefix: apiserver.Pgo.Pgo.COImagePrefix,
		COImageTag:    apiserver.Pgo.Pgo.COImageTag,
		BackupRoot:    pvcRoot,
		PVCName:       pvcName,
	}

	err = apiserver.LspvcTemplate.Execute(&doc2, pvcFields)
	if err != nil {
		log.Error(err.Error())
		return newlines, err
	}
	docString := doc2.String()
	log.Debug(docString)

	//create lspvc pod
	newpod := v1.Pod{}
	err = json.Unmarshal(doc2.Bytes(), &newpod)
	if err != nil {
		log.Error("error unmarshalling json into Pod " + err.Error())
		return newlines, err
	}

	_, err = kubeapi.CreatePod(apiserver.Clientset, &newpod, apiserver.Namespace)
	if err != nil {
		return newlines, err
	}

	timeout := time.Duration(6 * time.Second)
	lo := meta_v1.ListOptions{LabelSelector: "name=lspvc," + util.LABEL_PVCNAME + "=" + pvcName}
	podPhase := v1.PodSucceeded
	err = util.WaitUntilPod(apiserver.Clientset, lo, podPhase, timeout, apiserver.Namespace)
	if err != nil {
		log.Error("error waiting on lspvc pod to complete" + err.Error())
	}

	time.Sleep(5000 * time.Millisecond)

	//get lspvc pod output
	logOptions := v1.PodLogOptions{}
	req := apiserver.Clientset.CoreV1().Pods(apiserver.Namespace).GetLogs(podName, &logOptions)
	if req == nil {
		log.Debug("error in get logs for " + podName)
	} else {
		log.Debug("got the logs for " + podName)
	}
	readCloser, err := req.Stream()
	if err != nil {
		log.Error(err.Error())
		return newlines, err
	}

	defer func() {
		if readCloser != nil {
			readCloser.Close()
		}
	}()
	var buf2 bytes.Buffer
	_, err = io.Copy(&buf2, readCloser)
	log.Debugf("backups are... \n%s", buf2.String())

	log.Debug("pvc=" + pvcName)
	lines := strings.Split(buf2.String(), "\n")
	//chop off last line since its only a newline
	last := len(lines) - 1
	newlines = make([]string, last)
	copy(newlines, lines[:last])

	for k, v := range newlines {
		if k == len(newlines)-1 {
			log.Debugf("%s%s\n", apiserver.TreeTrunk, "/"+v)
		} else {
			log.Debugf("%s%s\n", apiserver.TreeBranch, "/"+v)
		}
	}

	//delete lspvc pod
	err = kubeapi.DeletePod(apiserver.Clientset, podName, apiserver.Namespace)
	return newlines, err

}
