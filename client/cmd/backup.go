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

// Package cmd provides the command line functions of the crunchy CLI
package cmd

import (
	//"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"strings"
	"text/template"
	"time"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "perform a Backup",
	Long: `BACKUP performs a Backup, for example:
			pgo backup mycluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("backup called")
		if len(args) == 0 {
			fmt.Println(`You must specify the cluster to backup.`)
		} else {
			createBackup(args)
		}

	},
}

func init() {
	RootCmd.AddCommand(backupCmd)
}

func showBackup(args []string) {
	log.Debugf("showBackup called %v\n", args)

	//show pod information for job
	for _, arg := range args {
		log.Debug("show backup called for " + arg)
		//pg-database=basic or
		//pgbackup=true
		if arg == "all" {
			lo := v1.ListOptions{LabelSelector: "pgbackup=true"}
			log.Debug("label selector is " + lo.LabelSelector)
			pods, err2 := Clientset.Core().Pods(Namespace).List(lo)
			if err2 != nil {
				log.Error(err2.Error())
				return
			}
			for _, pod := range pods.Items {
				showBackupInfo(pod.ObjectMeta.Labels["pg-database"])
			}

		} else {
			showBackupInfo(arg)

		}

	}

}
func showBackupInfo(name string) {
	fmt.Println("\nbackup information for " + name + "...")
	//print the pgbackups TPR if it exists
	result := tpr.PgBackup{}
	err := Tprclient.Get().
		Resource("pgbackups").
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&result)
	if err == nil {
		printBackupTPR(&result)
	} else if errors.IsNotFound(err) {
		fmt.Println("\npgbackup TPR not found ")
	} else {
		log.Errorf("\npgbackup %s\n", name+" lookup error ")
		log.Error(err.Error())
		return
	}

	//print the backup jobs if any exists
	lo := v1.ListOptions{LabelSelector: "pgbackup=true,pg-database=" + name}
	log.Debug("label selector is " + lo.LabelSelector)
	pods, err2 := Clientset.Core().Pods(Namespace).List(lo)
	if err2 != nil {
		log.Error(err2.Error())
	}
	fmt.Printf("\nbackup job pods for database %s\n", name+"...")

	pvcMap := make(map[string]string)

	for _, p := range pods.Items {

		//get the pgdata volume info
		for _, v := range p.Spec.Volumes {
			if v.Name == "pgdata" {
				fmt.Printf("%s%s (pvc %s)\n\n", TREE_TRUNK, p.Name, v.VolumeSource.PersistentVolumeClaim.ClaimName)
				pvcMap[v.VolumeSource.PersistentVolumeClaim.ClaimName] = v.VolumeSource.PersistentVolumeClaim.ClaimName
			}
		}
		fmt.Println("")

	}

	log.Debugf("ShowPVC is %v\n", ShowPVC)

	if ShowPVC {
		//print pvc information for all jobs
		for key, _ := range pvcMap {
			displayPVC(name, key)
		}
	}
}

func printBackupTPR(result *tpr.PgBackup) {
	fmt.Printf("%s%s\n", "", "")
	fmt.Printf("%s%s\n", "", "pgbackup : "+result.Spec.Name)

	fmt.Printf("%s%s\n", TREE_BRANCH, "PVC Name:\t"+result.Spec.PVC_NAME)
	fmt.Printf("%s%s\n", TREE_BRANCH, "PVC Access Mode:\t"+result.Spec.PVC_ACCESS_MODE)
	fmt.Printf("%s%s\n", TREE_BRANCH, "PVC Size:\t\t"+result.Spec.PVC_SIZE)
	fmt.Printf("%s%s\n", TREE_BRANCH, "CCP_IMAGE_TAG:\t"+result.Spec.CCP_IMAGE_TAG)
	fmt.Printf("%s%s\n", TREE_BRANCH, "Backup Status:\t"+result.Spec.BACKUP_STATUS)
	fmt.Printf("%s%s\n", TREE_BRANCH, "Backup Host:\t"+result.Spec.BACKUP_HOST)
	fmt.Printf("%s%s\n", TREE_BRANCH, "Backup User:\t"+result.Spec.BACKUP_USER)
	fmt.Printf("%s%s\n", TREE_BRANCH, "Backup Pass:\t"+result.Spec.BACKUP_PASS)
	fmt.Printf("%s%s\n", TREE_TRUNK, "Backup Port:\t"+result.Spec.BACKUP_PORT)

}

func createBackup(args []string) {
	log.Debugf("createBackup called %v\n", args)

	var err error
	var newInstance *tpr.PgBackup

	for _, arg := range args {
		log.Debug("create backup called for " + arg)
		result := tpr.PgBackup{}

		// error if it already exists
		err = Tprclient.Get().
			Resource("pgbackups").
			Namespace(Namespace).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			fmt.Println("pgbackup " + arg + " was found so we recreate it")
			dels := make([]string, 1)
			dels[0] = arg
			deleteBackup(dels)
			time.Sleep(2000 * time.Millisecond)
		} else if errors.IsNotFound(err) {
			fmt.Println("pgbackup " + arg + " not found so we will create it")
		} else {
			log.Error("error getting pgbackup " + arg)
			log.Error(err.Error())
			break
		}
		// Create an instance of our TPR
		newInstance, err = getBackupParams(arg)
		if err != nil {
			log.Error("error creating backup")
			break
		}

		err = Tprclient.Post().
			Resource("pgbackups").
			Namespace(Namespace).
			Body(newInstance).
			Do().Into(&result)
		if err != nil {
			log.Error("error in creating PgBackup TPR instance")
			log.Error(err.Error())
		}
		fmt.Println("created PgBackup " + arg)

	}

}

func deleteBackup(args []string) {
	log.Debugf("deleteBackup called %v\n", args)
	var err error
	backupList := tpr.PgBackupList{}
	err = Tprclient.Get().Resource("pgbackups").Do().Into(&backupList)
	if err != nil {
		log.Error("error getting backup list")
		log.Error(err.Error())
		return
	}
	// delete the pgbackup resource instance
	// which will cause the operator to remove the related Job
	for _, arg := range args {
		backupFound := false
		for _, backup := range backupList.Items {
			if arg == "all" || backup.Spec.Name == arg {
				backupFound = true
				err = Tprclient.Delete().
					Resource("pgbackups").
					Namespace(Namespace).
					Name(backup.Spec.Name).
					Do().
					Error()
				if err != nil {
					log.Error("error deleting pgbackup " + arg)
					log.Error(err.Error())
				}
				fmt.Println("deleted pgbackup " + backup.Spec.Name)
			}

		}
		if !backupFound {
			fmt.Println("backup " + arg + " not found")
		}

	}

}

func getBackupParams(name string) (*tpr.PgBackup, error) {
	var newInstance *tpr.PgBackup

	spec := tpr.PgBackupSpec{}
	spec.Name = name
	spec.PVC_NAME = viper.GetString("PVC_NAME")
	spec.PVC_ACCESS_MODE = viper.GetString("CLUSTER.PVC_ACCESS_MODE")
	spec.PVC_SIZE = viper.GetString("CLUSTER.PVC_SIZE")
	spec.CCP_IMAGE_TAG = viper.GetString("CLUSTER.CCP_IMAGE_TAG")
	spec.BACKUP_STATUS = "initial"
	spec.BACKUP_HOST = "basic"
	spec.BACKUP_USER = "master"
	spec.BACKUP_PASS = "password"
	spec.BACKUP_PORT = "5432"

	cluster := tpr.PgCluster{}
	err := Tprclient.Get().
		Resource("pgclusters").
		Namespace(Namespace).
		Name(name).
		Do().
		Into(&cluster)
	if err == nil {
		spec.BACKUP_HOST = cluster.Spec.Name
		spec.BACKUP_USER = cluster.Spec.PG_MASTER_USER
		spec.BACKUP_PASS = cluster.Spec.PG_MASTER_PASSWORD
		spec.BACKUP_PORT = cluster.Spec.Port
	} else if errors.IsNotFound(err) {
		log.Debug(name + " is not a cluster")
		return newInstance, err
	} else {
		log.Error("error getting pgcluster " + name)
		log.Error(err.Error())
		return newInstance, err
	}

	newInstance = &tpr.PgBackup{
		Metadata: api.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance, nil
}

type PodTemplateFields struct {
	Name         string
	CO_IMAGE_TAG string
	BACKUP_ROOT  string
	PVC_NAME     string
}

func displayPVC(name string, pvcName string) {
	var POD_PATH = viper.GetString("PGO.LSPVC_TEMPLATE")
	var PodTemplate *template.Template
	var err error
	var buf []byte
	var doc2 bytes.Buffer
	var podName = "lspvc-" + name

	fmt.Println("PVC " + pvcName + " contains...")

	//delete lspvc pod if it was not deleted for any reason prior
	_, err = Clientset.Core().Pods(Namespace).Get(podName)
	if errors.IsNotFound(err) {
		//
	} else if err != nil {
		log.Error(err.Error())
	} else {
		log.Debug("deleting prior pod " + podName)
		err = Clientset.Core().Pods(Namespace).Delete(podName,
			&v1.DeleteOptions{})
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

	podFields := PodTemplateFields{
		Name:         podName,
		CO_IMAGE_TAG: viper.GetString("PGO.CO_IMAGE_TAG"),
		BACKUP_ROOT:  name + "-backups",
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
		log.Error("error unmarshalling json into Pod ")
		log.Error(err.Error())
		return
	}
	var resultPod *v1.Pod
	resultPod, err = Clientset.Core().Pods(Namespace).Create(&newpod)
	if err != nil {
		log.Error("error creating lspvc Pod ")
		log.Error(err.Error())
		return
	}
	log.Debug("created pod " + resultPod.Name)

	//sleep a bit for the pod to finish, replace later with watch or better
	time.Sleep(3000 * time.Millisecond)

	//get lspvc pod output
	logOptions := v1.PodLogOptions{}
	req := Clientset.Core().Pods(Namespace).GetLogs(podName, &logOptions)
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

	defer readCloser.Close()
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
			fmt.Printf("%s%s\n", TREE_TRUNK, name+"-backups/"+v)
		} else {
			fmt.Printf("%s%s\n", TREE_BRANCH, name+"-backups/"+v)
		}
	}

	//delete lspvc pod
	err = Clientset.Core().Pods(Namespace).Delete(podName,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error(err.Error())
		log.Error("error deleting lspvc pod " + podName)
	}

}
