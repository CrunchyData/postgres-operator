package pvcservice

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/util"
	"github.com/spf13/viper"
	"io"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"strings"
	"text/template"
	"time"
)

type lspvcTemplateFields struct {
	Name          string
	COImagePrefix string
	COImageTag    string
	BackupRoot    string
	PVCName       string
}

var lspvcTemplate *template.Template

func init() {
	lspvcTemplate = util.LoadTemplate("/config/pgo.lspvc-template.json")

}

// ShowPVC ...
func ShowPVC(Namespace string, pvcName, PVCRoot string) ([]string, error) {
	pvcList := make([]string, 1)

	pvc, err := apiserver.Clientset.CoreV1().PersistentVolumeClaims(Namespace).Get(pvcName, meta_v1.GetOptions{})
	if err != nil {
		log.Error("\nPVC %s\n", pvcName+" is not found")
		log.Error(err.Error())
	} else {
		log.Debug("\nPVC %s\n", pvc.Name+" is found")
		pvcList, err = printPVCListing(Namespace, pvc.Name, PVCRoot)
	}

	return pvcList, err

}

// printPVCListing ...
func printPVCListing(namespace, pvcName, PVCRoot string) ([]string, error) {
	newlines := make([]string, 1)
	var err error
	var doc2 bytes.Buffer
	var podName = "lspvc-" + pvcName

	//delete lspvc pod if it was not deleted for any reason prior
	_, err = apiserver.Clientset.CoreV1().Pods(namespace).Get(podName, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {
		//
	} else if err != nil {
		log.Error(err.Error())
		return newlines, err
	} else {
		log.Debug("deleting prior pod " + podName)
		err = apiserver.Clientset.Core().Pods(namespace).Delete(podName,
			&meta_v1.DeleteOptions{})
		if err != nil {
			log.Error("delete pod error " + err.Error()) //TODO this is debug info
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
		COImagePrefix: viper.GetString("Pgo.COImagePrefix"),
		COImageTag:    viper.GetString("Pgo.COImageTag"),
		BackupRoot:    pvcRoot,
		PVCName:       pvcName,
	}

	err = lspvcTemplate.Execute(&doc2, pvcFields)
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
	var resultPod *v1.Pod
	resultPod, err = apiserver.Clientset.CoreV1().Pods(namespace).Create(&newpod)
	if err != nil {
		log.Error("error creating lspvc Pod " + err.Error())
		return newlines, err
	}
	log.Debug("created pod " + resultPod.Name)
	timeout := time.Duration(6 * time.Second)
	lo := meta_v1.ListOptions{LabelSelector: "name=lspvc,pvcname=" + pvcName}
	podPhase := v1.PodSucceeded
	err = util.WaitUntilPod(apiserver.Clientset, lo, podPhase, timeout, namespace)
	if err != nil {
		log.Error("error waiting on lspvc pod to complete" + err.Error())
	}

	time.Sleep(5000 * time.Millisecond)

	//get lspvc pod output
	logOptions := v1.PodLogOptions{}
	req := apiserver.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &logOptions)
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
	err = apiserver.Clientset.CoreV1().Pods(namespace).Delete(podName,
		&meta_v1.DeleteOptions{})
	if err != nil {
		log.Error(err.Error())
		log.Error("error deleting lspvc pod " + podName)
	}

	return newlines, err

}
