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
	"context"
	"errors"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pkg/events"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Label ... 2 forms ...
// pgo label  myucluser yourcluster --label=env=prod
// pgo label  --label=env=prod --selector=name=mycluster
func Label(request *msgs.LabelRequest, ns, pgouser string) msgs.LabelResponse {
	ctx := context.TODO()
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
		resp.Status.Msg = "labels not formatted correctly"
		return resp
	}

	clusterList := crv1.PgclusterList{}
	if len(request.Args) > 0 && request.Args[0] == "all" {
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{})
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

		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{LabelSelector: request.Selector})
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
		// each arg represents a cluster name
		items := make([]crv1.Pgcluster, 0)
		for _, cluster := range request.Args {
			result, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Get(ctx, cluster, metav1.GetOptions{})
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
	ctx := context.TODO()
	patchBytes, err := kubeapi.NewMergePatch().Add("metadata", "labels")(newLabels).Bytes()
	if err != nil {
		log.Error(err.Error())
		return
	}

	for i := 0; i < len(items); i++ {
		if DryRun {
			log.Debug("dry run only")
		} else {
			log.Debugf("patching cluster %s: %s", items[i].Spec.Name, patchBytes)
			_, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).
				Patch(ctx, items[i].Spec.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
			if err != nil {
				log.Error(err.Error())
			}

			// publish event for create label
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
		// get deployments for this CRD
		selector := config.LABEL_PG_CLUSTER + "=" + items[i].Spec.Name
		deployments, err := apiserver.Clientset.
			AppsV1().Deployments(ns).
			List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return
		}

		for _, d := range deployments.Items {
			// update Deployment with the label
			if !DryRun {
				log.Debugf("patching deployment %s: %s", d.Name, patchBytes)
				_, err := apiserver.Clientset.AppsV1().Deployments(ns).
					Patch(ctx, d.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
				if err != nil {
					log.Error(err.Error())
				}
			}
		}

	}
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

// DeleteLabel ...
// pgo delete label  mycluster yourcluster --label=env=prod
// pgo delete label  --label=env=prod --selector=group=somegroup
func DeleteLabel(request *msgs.DeleteLabelRequest, ns string) msgs.LabelResponse {
	ctx := context.TODO()
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
		resp.Status.Msg = "labels not formatted correctly"
		return resp
	}

	clusterList := crv1.PgclusterList{}
	if len(request.Args) > 0 && request.Args[0] == "all" {
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{})
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
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).List(ctx, metav1.ListOptions{LabelSelector: request.Selector})
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
		// each arg represents a cluster name
		items := make([]crv1.Pgcluster, 0)
		for _, cluster := range request.Args {
			result, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Get(ctx, cluster, metav1.GetOptions{})
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

	err = deleteLabels(clusterList.Items, labelsMap, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	return resp
}

func deleteLabels(items []crv1.Pgcluster, labelsMap map[string]string, ns string) error {
	ctx := context.TODO()
	patch := kubeapi.NewMergePatch()
	for key := range labelsMap {
		patch.Remove("metadata", "labels", key)
	}
	patchBytes, err := patch.Bytes()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	for i := 0; i < len(items); i++ {
		log.Debugf("patching cluster %s: %s", items[i].Spec.Name, patchBytes)
		_, err = apiserver.Clientset.CrunchydataV1().Pgclusters(ns).
			Patch(ctx, items[i].Spec.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}

	for i := 0; i < len(items); i++ {
		// get deployments for this CRD
		selector := config.LABEL_PG_CLUSTER + "=" + items[i].Spec.Name
		deployments, err := apiserver.Clientset.
			AppsV1().Deployments(ns).
			List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return err
		}

		for _, d := range deployments.Items {
			log.Debugf("patching deployment %s: %s", d.Name, patchBytes)
			_, err = apiserver.Clientset.AppsV1().Deployments(ns).
				Patch(ctx, d.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
			if err != nil {
				log.Error(err.Error())
				return err
			}
		}

	}
	return err
}
