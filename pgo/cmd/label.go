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
	"fmt"
	log "github.com/Sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"os"
	//"github.com/crunchydata/postgres-operator/operator/util"
	"encoding/json"
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/kubernetes"
	//"github.com/spf13/viper"
	//"io/ioutil"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	//"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	//"os/user"
	"strings"
)

var LabelCmdLabel string
var LabelMap map[string]string
var DeleteLabel bool

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "label a set of clusters",
	Long: `LABEL allows you to add or remove a label on a set of clusters
For example:

pgo label mycluster yourcluster --label=environment=prod 
pgo label mycluster yourcluster --label=environment=prod  --delete-label
pgo label --label=environment=prod --selector=name=mycluster
pgo label --label=environment=prod --selector=status=final --dry-run
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("label called")
		if len(args) == 0 && Selector == "" {
			log.Error("selector or list of clusters is required to label a policy")
			return
		}
		if LabelCmdLabel == "" {
			log.Error(`You must specify the label to apply.`)
		} else {
			validateLabel()
			labelClusters(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(labelCmd)

	labelCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	labelCmd.Flags().StringVarP(&LabelCmdLabel, "label", "l", "", "The new label to apply for any selected or specified clusters")
	labelCmd.Flags().BoolVarP(&DryRun, "dry-run", "d", false, "--dry-run shows clusters that label would be applied to but does not actually label them")
	labelCmd.Flags().BoolVarP(&DeleteLabel, "delete-label", "x", false, "--delete-label deletes a label from matching clusters")

}

func labelClusters(clusters []string) {
	var err error

	if len(clusters) == 0 && Selector == "" {
		fmt.Println("no clusters specified")
		return
	}
	//get filtered list of pgcluster crv1s
	//get a list of all clusters
	clusterList := crv1.PgclusterList{}
	myselector := labels.Everything()
	if Selector != "" {
		log.Debug("selector is " + Selector)
		myselector, err = labels.Parse(Selector)
		if err != nil {
			log.Error("could not parse --selector value " + err.Error())
			return
		}

		log.Debugf("label selector is [%v]\n", myselector)
		err = RestClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(Namespace).
			LabelsSelectorParam(myselector).
			Do().
			Into(&clusterList)
		if err != nil {
			log.Error("error getting list of clusters" + err.Error())
			return
		}
		if len(clusterList.Items) == 0 {
			fmt.Println("no clusters found")
			return
		}
	} else {
		//each arg represents a cluster name or the special 'all' value
		items := make([]crv1.Pgcluster, 0)
		for _, cluster := range clusters {
			result := crv1.Pgcluster{}
			err := RestClient.Get().
				Resource(crv1.PgclusterResourcePlural).
				Namespace(Namespace).
				Name(cluster).
				Do().
				Into(&result)
			if err != nil {
				log.Error("error getting list of clusters" + err.Error())
				return
			}
			fmt.Println(result.Spec.Name)
			items = append(items, result)
		}
		clusterList.Items = items
	}

	addLabels(clusterList.Items)

}

func addLabels(items []crv1.Pgcluster) {
	for i := 0; i < len(items); i++ {
		fmt.Println("adding label to " + items[i].Spec.Name)
		if DryRun {
			fmt.Println("dry run only")
		} else {
			err := PatchPgcluster(RestClient, LabelCmdLabel, items[i], Namespace)
			if err != nil {
				log.Error(err.Error())
			}
		}
	}

	for i := 0; i < len(items); i++ {
		//get deployments for this TPR
		lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + items[i].Spec.Name}
		deployments, err := Clientset.ExtensionsV1beta1().Deployments(Namespace).List(lo)
		if err != nil {
			log.Error("error getting list of deployments" + err.Error())
			return
		}

		for _, d := range deployments.Items {
			//update Deployment with the label
			//fmt.Println(TREE_BRANCH + "deployment : " + d.ObjectMeta.Name)
			if DryRun {
			} else {
				err := updateLabels(&d, Clientset, items[i].Spec.Name, Namespace, LabelMap)
				if err != nil {
					log.Error(err.Error())
				}
			}
		}

	}
}

func updateLabels(deployment *v1beta1.Deployment, clientset *kubernetes.Clientset, clusterName string, namespace string, newLabels map[string]string) error {

	var err error

	log.Debugf("%v is the labels to apply\n", newLabels)

	var patchBytes, newData, origData []byte
	origData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	accessor, err2 := meta.Accessor(deployment)
	if err2 != nil {
		return err2
	}

	objLabels := accessor.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}

	//update the deployment labels
	for key, value := range newLabels {
		objLabels[key] = value
	}
	log.Debugf("updated labels are %v\n", objLabels)

	accessor.SetLabels(objLabels)

	newData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	if err != nil {
		return err
	}

	_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Patch(clusterName, types.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debug("error patching deployment " + err.Error())
	}
	return err

}

func PatchPgcluster(RestClient *rest.RESTClient, newLabel string, oldTpr crv1.Pgcluster, namespace string) error {

	fields := strings.Split(newLabel, "=")
	labelKey := fields[0]
	labelValue := fields[1]
	oldData, err := json.Marshal(oldTpr)
	if err != nil {
		return err
	}
	if oldTpr.ObjectMeta.Labels == nil {
		oldTpr.ObjectMeta.Labels = make(map[string]string)
	}
	oldTpr.ObjectMeta.Labels[labelKey] = labelValue
	var newData, patchBytes []byte
	newData, err = json.Marshal(oldTpr)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}

	log.Debug(string(patchBytes))

	_, err6 := RestClient.Patch(types.MergePatchType).
		Namespace(namespace).
		Resource(crv1.PgclusterResourcePlural).
		Name(oldTpr.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}

func validateLabel() {
	//TODO use  the k8s label parser for this validation
	LabelMap = make(map[string]string)
	userValues := strings.Split(LabelCmdLabel, ",")
	for _, v := range userValues {
		pair := strings.Split(v, "=")
		if len(pair) != 2 {
			log.Error("label format incorrect, requires name=value")
			os.Exit(2)
		}
		LabelMap[pair[0]] = pair[1]
	}
}
