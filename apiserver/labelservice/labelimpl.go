package labelservice

/*
Copyright 2018 Crunchy Data Solutions, Inc.
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
	"errors"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"strings"
)

// Label ... 2 forms ...
// pgo label  myucluser yourcluster --label=env=prod
// pgo label  myucluser yourcluster --label=env=prod --delete-label
// pgo label  --label=env=prod --selector=name=mycluster
func Label(request *msgs.LabelRequest) msgs.LabelResponse {
	var err error
	var labelsMap map[string]string
	resp := msgs.LabelResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	if len(request.Args) == 0 && request.Selector == "" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "no clusters specified"
		return resp
	}

	labelsMap, err = validateLabel(request.LabelCmdLabel)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "lables not formatted correctly"
		return resp
	}

	clusterList := crv1.PgclusterList{}
	myselector := labels.Everything()
	if request.Selector != "" {
		log.Debug("selector is " + request.Selector)

		myselector, err = labels.Parse(request.Selector)
		if err != nil {
			log.Error("could not parse --selector value " + err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "could not parse --selector value " + err.Error()
			return resp
		}

		log.Debugf("label selector is [%s]\n", myselector.String())

		err = apiserver.RESTClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(apiserver.Namespace).
			Param("labelSelector", myselector.String()).
			//LabelsSelectorParam(myselector).
			Do().
			Into(&clusterList)
		if err != nil {
			log.Error("error getting list of clusters" + err.Error())
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "error getting list of clusters" + err.Error()
			return resp
		}
		if len(clusterList.Items) == 0 {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "no clusters found"
			return resp
		}
	} else {
		//each arg represents a cluster name or the special 'all' value
		items := make([]crv1.Pgcluster, 0)
		for _, cluster := range request.Args {
			result := crv1.Pgcluster{}
			err := apiserver.RESTClient.Get().
				Resource(crv1.PgclusterResourcePlural).
				Namespace(apiserver.Namespace).
				Name(cluster).
				Do().
				Into(&result)
			if err != nil {
				log.Error("error getting list of clusters" + err.Error())
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "error getting list of clusters" + err.Error()
				return resp
			}

			//fmt.Println(result.Spec.Name)
			items = append(items, result)
		}
		clusterList.Items = items
	}

	for _, c := range clusterList.Items {
		resp.Results = append(resp.Results, "adding label to "+c.Spec.Name)
	}

	addLabels(clusterList.Items, request.DryRun, request.LabelCmdLabel, labelsMap)

	return resp

}

func addLabels(items []crv1.Pgcluster, DryRun bool, LabelCmdLabel string, newLabels map[string]string) {
	for i := 0; i < len(items); i++ {
		log.Debug("adding label to " + items[i].Spec.Name)
		if DryRun {
			log.Debug("dry run only")
		} else {
			err := PatchPgcluster(LabelCmdLabel, items[i])
			if err != nil {
				log.Error(err.Error())
			}
		}
	}

	for i := 0; i < len(items); i++ {
		//get deployments for this CRD
		lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + items[i].Spec.Name}
		deployments, err := apiserver.Clientset.ExtensionsV1beta1().Deployments(apiserver.Namespace).List(lo)
		if err != nil {
			log.Error("error getting list of deployments" + err.Error())
			return
		}
		for _, d := range deployments.Items {
			//update Deployment with the label
			//fmt.Println(TreeBranch + "deployment : " + d.ObjectMeta.Name)
			if !DryRun {
				err := updateLabels(&d, items[i].Spec.Name, newLabels)
				if err != nil {
					log.Error(err.Error())
				}
			}
		}

	}
}

func updateLabels(deployment *v1beta1.Deployment, clusterName string, newLabels map[string]string) error {

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

	_, err = apiserver.Clientset.ExtensionsV1beta1().Deployments(apiserver.Namespace).Patch(clusterName, types.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debug("error patching deployment " + err.Error())
	}
	return err

}

func PatchPgcluster(newLabel string, oldCRD crv1.Pgcluster) error {

	fields := strings.Split(newLabel, "=")
	labelKey := fields[0]
	labelValue := fields[1]
	oldData, err := json.Marshal(oldCRD)
	if err != nil {
		return err
	}
	if oldCRD.ObjectMeta.Labels == nil {
		oldCRD.ObjectMeta.Labels = make(map[string]string)
	}
	oldCRD.ObjectMeta.Labels[labelKey] = labelValue
	var newData, patchBytes []byte
	newData, err = json.Marshal(oldCRD)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}

	log.Debug(string(patchBytes))
	_, err6 := apiserver.RESTClient.Patch(types.MergePatchType).
		Namespace(apiserver.Namespace).
		Resource(crv1.PgclusterResourcePlural).
		Name(oldCRD.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}

func validateLabel(LabelCmdLabel string) (map[string]string, error) {
	var err error
	labelMap := make(map[string]string)
	userValues := strings.Split(LabelCmdLabel, ",")
	for _, v := range userValues {
		pair := strings.Split(v, "=")
		if len(pair) != 2 {
			log.Error("label format incorrect, requires name=value")
			return labelMap, errors.New("label format incorrect, requires name=value")
		}

		errs := validation.IsDNS1035Label(pair[0])
		if len(errs) > 0 {
			return labelMap, errors.New("label format incorrect, requires name=value " + errs[0])
		}
		errs = validation.IsDNS1035Label(pair[1])
		if len(errs) > 0 {
			return labelMap, errors.New("label format incorrect, requires name=value " + errs[0])
		}

		labelMap[pair[0]] = pair[1]
	}
	return labelMap, err
}
