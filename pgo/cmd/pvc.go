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

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/kraken/util"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	//"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"strings"
	"text/template"
	"time"
)

func showPVC(args []string) {
	log.Debugf("showPVC called %v\n", args)

	//args are a list of pvc names
	for _, arg := range args {
		log.Debug("show pvc called for " + arg)
		printPVC(arg)

	}

}
func printPVC(pvcName string) {

	var pvc *v1.PersistentVolumeClaim
	var err error

	pvc, err = Clientset.CoreV1().PersistentVolumeClaims(Namespace).Get(pvcName, meta_v1.GetOptions{})
	if err != nil {
		fmt.Printf("\nPVC %s\n", pvcName+" is not found")
		fmt.Println(err.Error())
	} else {
		log.Debug("\nPVC %s\n", pvc.Name+" is found")
		PrintPVCListing(pvc.Name)
	}

}

func PrintPVCListing(pvcName string) {
	var POD_PATH = viper.GetString("PGO.LSPVC_TEMPLATE")
	var PodTemplate *template.Template
	var err error
	var buf []byte
	var doc2 bytes.Buffer
	var podName = "lspvc-" + pvcName

	//delete lspvc pod if it was not deleted for any reason prior
	_, err = Clientset.CoreV1().Pods(Namespace).Get(podName, meta_v1.GetOptions{})
	if errors.IsNotFound(err) {
		//
	} else if err != nil {
		log.Error(err.Error())
	} else {
		log.Debug("deleting prior pod " + podName)
		err = Clientset.Core().Pods(Namespace).Delete(podName,
			&meta_v1.DeleteOptions{})
		if err != nil {
			log.Error("delete pod error " + err.Error()) //TODO this is debug info
		}
		//sleep a bit for the pod to be deleted
		time.Sleep(2000 * time.Millisecond)
	}

	buf, err = ioutil.ReadFile(POD_PATH)
	if err != nil {
		log.Error("error reading lspvc_template file")
		log.Error("make sure it is specified in your .pgo.yaml config")
		log.Error(err.Error())
		return
	}
	PodTemplate = template.Must(template.New("pod template").Parse(string(buf)))

	pvcRoot := "/"
	if PVCRoot != "" {
		log.Debug("using " + PVCRoot + " as the PVC listing root")
		pvcRoot = PVCRoot
		fmt.Println(pvcName + "/" + pvcRoot)
	} else {
		fmt.Println(pvcName)
	}

	podFields := PodTemplateFields{
		Name:         podName,
		CO_IMAGE_TAG: viper.GetString("PGO.CO_IMAGE_TAG"),
		BACKUP_ROOT:  pvcRoot,
		PVC_NAME:     pvcName,
	}

	err = PodTemplate.Execute(&doc2, podFields)
	if err != nil {
		log.Error(err.Error())
		return
	}
	podDocString := doc2.String()
	log.Debug(podDocString)

	//template name is lspvc-pod.json
	//create lspvc pod
	newpod := v1.Pod{}
	err = json.Unmarshal(doc2.Bytes(), &newpod)
	if err != nil {
		log.Error("error unmarshalling json into Pod " + err.Error())
		return
	}
	var resultPod *v1.Pod
	resultPod, err = Clientset.CoreV1().Pods(Namespace).Create(&newpod)
	if err != nil {
		log.Error("error creating lspvc Pod " + err.Error())
		return
	}
	log.Debug("created pod " + resultPod.Name)

	timeout := time.Duration(6 * time.Second)
	lo := meta_v1.ListOptions{LabelSelector: "name=lspvc,pvcname=" + pvcName}
	podPhase := v1.PodSucceeded
	err = util.WaitUntilPod(Clientset, lo, podPhase, timeout, Namespace)
	if err != nil {
		log.Error("error waiting on lspvc pod to complete" + err.Error())
	}

	time.Sleep(5000 * time.Millisecond)

	//get lspvc pod output
	logOptions := v1.PodLogOptions{}
	req := Clientset.CoreV1().Pods(Namespace).GetLogs(podName, &logOptions)
	if req == nil {
		log.Debug("error in get logs for " + podName)
	} else {
		log.Debug("got the logs for " + podName)
	}

	readCloser, err := req.Stream()
	if err != nil {
		log.Error(err.Error())
		return
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
	newlines := make([]string, last)
	copy(newlines, lines[:last])

	for k, v := range newlines {
		if k == len(newlines)-1 {
			fmt.Printf("%s%s\n", TREE_TRUNK, "/"+v)
		} else {
			fmt.Printf("%s%s\n", TREE_BRANCH, "/"+v)
		}
	}

	//delete lspvc pod
	err = Clientset.CoreV1().Pods(Namespace).Delete(podName,
		&meta_v1.DeleteOptions{})
	if err != nil {
		log.Error(err.Error())
		log.Error("error deleting lspvc pod " + podName)
	}

}
