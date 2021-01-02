package policyservice

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
	"fmt"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/apiserver/labelservice"
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pkg/events"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePolicy ...
func CreatePolicy(client pgo.Interface, policyName, policyURL, policyFile, ns, pgouser string) (bool, error) {

	log.Debugf("create policy called for %s", policyName)

	// Create an instance of our CRD
	spec := crv1.PgpolicySpec{}
	spec.Namespace = ns
	spec.Name = policyName
	spec.URL = policyURL
	spec.SQL = policyFile

	myLabels := make(map[string]string)
	myLabels[config.LABEL_PGOUSER] = pgouser

	newInstance := &crv1.Pgpolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:   policyName,
			Labels: myLabels,
		},
		Spec: spec,
	}

	_, err := client.CrunchydataV1().Pgpolicies(ns).Create(newInstance)

	if kerrors.IsAlreadyExists(err) {
		log.Debugf("pgpolicy %s was found so we will not create it", policyName)
		return true, nil
	}

	return false, err

}

// ShowPolicy ...
func ShowPolicy(client pgo.Interface, name string, allflags bool, ns string) crv1.PgpolicyList {
	policyList := crv1.PgpolicyList{}

	if allflags {
		//get a list of all policies
		list, err := client.CrunchydataV1().Pgpolicies(ns).List(metav1.ListOptions{})
		if list != nil && err == nil {
			policyList = *list
		}
	} else {
		policy, err := client.CrunchydataV1().Pgpolicies(ns).Get(name, metav1.GetOptions{})
		if policy != nil && err == nil {
			policyList.Items = []crv1.Pgpolicy{*policy}
		}
	}

	return policyList

}

// DeletePolicy ...
func DeletePolicy(client pgo.Interface, policyName, ns, pgouser string) msgs.DeletePolicyResponse {
	resp := msgs.DeletePolicyResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	policyList, err := client.CrunchydataV1().Pgpolicies(ns).List(metav1.ListOptions{})
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	policyFound := false
	log.Debugf("deleting policy %s", policyName)
	for _, policy := range policyList.Items {
		if policyName == "all" || policyName == policy.Spec.Name {
			//update pgpolicy with current pgouser so that
			//we can create an event holding the pgouser
			//that deleted the policy
			policy.ObjectMeta.Labels[config.LABEL_PGOUSER] = pgouser
			_, err = client.CrunchydataV1().Pgpolicies(ns).Update(&policy)

			//ok, now delete the pgpolicy
			policyFound = true
			err = client.CrunchydataV1().Pgpolicies(ns).Delete(policy.Spec.Name, &metav1.DeleteOptions{})
			if err == nil {
				msg := "deleted policy " + policy.Spec.Name
				log.Debug(msg)
				resp.Results = append(resp.Results, msg)
			} else {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}

		}
	}

	if !policyFound {
		log.Debugf("policy %s not found", policyName)
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "policy " + policyName + " not found"
		return resp
	}

	return resp

}

// ApplyPolicy ...
// pgo apply mypolicy --selector=name=mycluster
func ApplyPolicy(request *msgs.ApplyPolicyRequest, ns, pgouser string) msgs.ApplyPolicyResponse {
	var err error

	resp := msgs.ApplyPolicyResponse{}
	resp.Name = make([]string, 0)
	resp.Status.Msg = ""
	resp.Status.Code = msgs.Ok

	//validate policy
	err = util.ValidatePolicy(apiserver.Clientset, ns, request.Name)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "policy " + request.Name + " is not found, cancelling request"
		return resp
	}

	//get filtered list of Deployments
	selector := request.Selector
	log.Debugf("apply policy selector string=[%s]", selector)

	//get a list of all clusters
	clusterList, err := apiserver.Clientset.
		CrunchydataV1().Pgclusters(ns).
		List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}
	log.Debugf("apply policy clusters found len is %d", len(clusterList.Items))

	// Return an error if any clusters identified for the policy are in standby mode.  Standby
	// clusters are in read-only mode, and therefore cannot have policies applied to them
	// until standby mode has been disabled.
	if hasStandby, standbyClusters := apiserver.PGClusterListHasStandby(*clusterList); hasStandby {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = fmt.Sprintf("Request rejected, unable to load clusters %s: %s."+
			strings.Join(standbyClusters, ","), apiserver.ErrStandbyNotAllowed.Error())
		return resp
	}

	var allDeployments []v1.Deployment
	for _, c := range clusterList.Items {
		depSelector := config.LABEL_SERVICE_NAME + "=" + c.Name
		deployments, err := apiserver.Clientset.
			AppsV1().Deployments(ns).
			List(metav1.ListOptions{LabelSelector: depSelector})
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		if len(deployments.Items) < 1 {
			log.Errorf("%s  did not have a deployment for some reason", c.Name)
		} else {
			allDeployments = append(allDeployments, deployments.Items[0])
		}
	}

	if request.DryRun {
		for _, d := range allDeployments {
			log.Debugf("deployment : %s", d.ObjectMeta.Name)
			resp.Name = append(resp.Name, d.ObjectMeta.Name)
		}
		return resp
	}

	labels := make(map[string]string)
	labels[request.Name] = "pgpolicy"

	for _, d := range allDeployments {
		if d.ObjectMeta.Labels[config.LABEL_SERVICE_NAME] != d.ObjectMeta.Labels[config.LABEL_PG_CLUSTER] {
			log.Debugf("skipping apply policy on deployment %s", d.Name)
			continue
			//skip non primary deployments
		}

		log.Debugf("apply policy %s on deployment %s based on selector %s", request.Name, d.ObjectMeta.Name, selector)

		cl, err := apiserver.Clientset.
			CrunchydataV1().Pgclusters(ns).
			Get(d.ObjectMeta.Labels[config.LABEL_SERVICE_NAME], metav1.GetOptions{})
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		if err := util.ExecPolicy(apiserver.Clientset, apiserver.RESTConfig,
			ns, request.Name, d.ObjectMeta.Labels[config.LABEL_SERVICE_NAME], cl.Spec.Port); err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		err = util.UpdatePolicyLabels(apiserver.Clientset, d.ObjectMeta.Name, ns, labels)
		if err != nil {
			log.Error(err)
		}

		//update the pgcluster crd labels with the new policy
		err = labelservice.PatchPgcluster(map[string]string{request.Name: config.LABEL_PGPOLICY}, *cl, ns)
		if err != nil {
			log.Error(err)
		}

		resp.Name = append(resp.Name, d.ObjectMeta.Name)

		//publish event
		topics := make([]string, 1)
		topics[0] = events.EventTopicPolicy

		f := events.EventApplyPolicyFormat{
			EventHeader: events.EventHeader{
				Namespace: ns,
				Username:  pgouser,
				Topic:     topics,
				Timestamp: time.Now(),
				EventType: events.EventApplyPolicy,
			},
			Clustername: d.ObjectMeta.Labels[config.LABEL_PG_CLUSTER],
			Policyname:  request.Name,
		}

		err = events.Publish(f)
		if err != nil {
			log.Error(err.Error())
		}

	}
	return resp

}
