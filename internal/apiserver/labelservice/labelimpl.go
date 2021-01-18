package labelservice

/*
Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pkg/events"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Label ... 2 forms ...
// pgo label  myucluser yourcluster --label=env=prod
// pgo label  --label=env=prod --selector=name=mycluster
func Label(request *msgs.LabelRequest, ns, pgouser string) msgs.LabelResponse {
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

	labelsMap, err = apiserver.ValidateLabel(request.LabelCmdLabel)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	clusterList := crv1.PgclusterList{}
	if len(request.Args) > 0 && request.Args[0] == "all" {
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(metav1.ListOptions{})
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
		clusterList = *cl

	} else if request.Selector != "" {
		log.Debugf("label selector is %s and ns is %s", request.Selector, ns)

		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(metav1.ListOptions{LabelSelector: request.Selector})
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
		clusterList = *cl
	} else {
		//each arg represents a cluster name
		items := make([]crv1.Pgcluster, 0)
		for _, cluster := range request.Args {
			result, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Get(cluster, metav1.GetOptions{})
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "error getting list of clusters" + err.Error()
				return resp
			}

			items = append(items, *result)
		}
		clusterList.Items = items
	}

	for _, c := range clusterList.Items {
		resp.Results = append(resp.Results, c.Spec.Name)
	}

	addLabels(clusterList.Items, request.DryRun, request.LabelCmdLabel, labelsMap, ns, pgouser)

	return resp

}

func addLabels(items []crv1.Pgcluster, DryRun bool, LabelCmdLabel string, newLabels map[string]string, ns, pgouser string) {
	for i := 0; i < len(items); i++ {
		if DryRun {
			log.Debug("dry run only")
		} else {
			log.Debugf("adding label to cluster %s", items[i].Spec.Name)
			err := PatchPgcluster(newLabels, items[i], ns)
			if err != nil {
				log.Error(err.Error())
			}

			//publish event for create label
			topics := make([]string, 1)
			topics[0] = events.EventTopicCluster

			f := events.EventCreateLabelFormat{
				EventHeader: events.EventHeader{
					Namespace: ns,
					Username:  pgouser,
					Topic:     topics,
					EventType: events.EventCreateLabel,
				},
				Clustername: items[i].Spec.Name,
				Label:       LabelCmdLabel,
			}

			err = events.Publish(f)
			if err != nil {
				log.Error(err.Error())
			}

		}
	}

	for i := 0; i < len(items); i++ {
		//get deployments for this CRD
		selector := config.LABEL_PG_CLUSTER + "=" + items[i].Spec.Name
		deployments, err := apiserver.Clientset.
			AppsV1().Deployments(ns).
			List(metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return
		}

		for _, d := range deployments.Items {
			//update Deployment with the label
			if !DryRun {
				//err := updateLabels(&d, items[i].Spec.Name, newLabels)
				err := updateLabels(&d, d.Name, newLabels, ns)
				if err != nil {
					log.Error(err.Error())
				}
			}
		}

	}
}

func updateLabels(deployment *v1.Deployment, clusterName string, newLabels map[string]string, ns string) error {

	var err error

	log.Debugf("%v are the labels to apply", newLabels)

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
	log.Debugf("updated labels are %v", objLabels)

	accessor.SetLabels(objLabels)
	newData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	if err != nil {
		return err
	}

	_, err = apiserver.Clientset.AppsV1().Deployments(ns).Patch(clusterName, types.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debugf("error updating patching deployment %s", err.Error())
	}
	return err

}

func PatchPgcluster(newLabels map[string]string, oldCRD crv1.Pgcluster, ns string) error {

	oldData, err := json.Marshal(oldCRD)
	if err != nil {
		return err
	}
	if oldCRD.ObjectMeta.Labels == nil {
		oldCRD.ObjectMeta.Labels = make(map[string]string)
	}
	for key, value := range newLabels {
		oldCRD.ObjectMeta.Labels[key] = value
	}
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
	_, err6 := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Patch(oldCRD.Spec.Name, types.MergePatchType, patchBytes)

	return err6

}

// DeleteLabel ...
// pgo delete label  mycluster yourcluster --label=env=prod
// pgo delete label  --label=env=prod --selector=group=somegroup
func DeleteLabel(request *msgs.DeleteLabelRequest, ns string) msgs.LabelResponse {
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

	labelsMap, err = apiserver.ValidateLabel(request.LabelCmdLabel)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "labels not formatted correctly"
		return resp
	}

	clusterList := crv1.PgclusterList{}
	if len(request.Args) > 0 && request.Args[0] == "all" {
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(metav1.ListOptions{})
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
		clusterList = *cl

	} else if request.Selector != "" {
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(metav1.ListOptions{LabelSelector: request.Selector})
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
		clusterList = *cl
	} else {
		//each arg represents a cluster name
		items := make([]crv1.Pgcluster, 0)
		for _, cluster := range request.Args {
			result, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Get(cluster, metav1.GetOptions{})
			if err != nil {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "error getting list of clusters" + err.Error()
				return resp
			}

			items = append(items, *result)
		}
		clusterList.Items = items
	}

	for _, c := range clusterList.Items {
		resp.Results = append(resp.Results, "deleting label from "+c.Spec.Name)
	}

	err = deleteLabels(clusterList.Items, request.LabelCmdLabel, labelsMap, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	return resp

}

func deleteLabels(items []crv1.Pgcluster, LabelCmdLabel string, labelsMap map[string]string, ns string) error {
	var err error

	for i := 0; i < len(items); i++ {
		log.Debugf("deleting label from %s", items[i].Spec.Name)
		err = deletePatchPgcluster(labelsMap, items[i], ns)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}

	for i := 0; i < len(items); i++ {
		//get deployments for this CRD
		selector := config.LABEL_PG_CLUSTER + "=" + items[i].Spec.Name
		deployments, err := apiserver.Clientset.
			AppsV1().Deployments(ns).
			List(metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return err
		}

		for _, d := range deployments.Items {
			err = deleteTheLabel(&d, items[i].Spec.Name, labelsMap, ns)
			if err != nil {
				log.Error(err.Error())
				return err
			}
		}

	}
	return err
}

func deletePatchPgcluster(labelsMap map[string]string, oldCRD crv1.Pgcluster, ns string) error {

	oldData, err := json.Marshal(oldCRD)
	if err != nil {
		return err
	}
	if oldCRD.ObjectMeta.Labels == nil {
		oldCRD.ObjectMeta.Labels = make(map[string]string)
	}
	for k := range labelsMap {
		delete(oldCRD.ObjectMeta.Labels, k)
	}

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
	_, err6 := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Patch(oldCRD.Spec.Name, types.MergePatchType, patchBytes)

	return err6

}

func deleteTheLabel(deployment *v1.Deployment, clusterName string, labelsMap map[string]string, ns string) error {

	var err error

	log.Debugf("%v are the labels to delete", labelsMap)

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

	for k := range labelsMap {
		delete(objLabels, k)
	}
	log.Debugf("revised labels after delete are %v", objLabels)

	accessor.SetLabels(objLabels)
	newData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	if err != nil {
		return err
	}

	_, err = apiserver.Clientset.AppsV1().Deployments(ns).Patch(deployment.Name, types.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debugf("error patching deployment: %v", err.Error())
	}
	return err

}
