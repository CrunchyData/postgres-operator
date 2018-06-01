package policyservice

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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/labelservice"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	cluster "github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// CreatePolicy ...
func CreatePolicy(RESTClient *rest.RESTClient, policyName, policyURL, policyFile string) error {

	log.Debug("create policy called for " + policyName)
	result := crv1.Pgpolicy{}

	// error if it already exists
	found, err := kubeapi.Getpgpolicy(RESTClient,
		&result, policyName, apiserver.Namespace)
	if err == nil {
		log.Debug("pgpolicy " + policyName + " was found so we will not create it")
		return err
	} else if !found {
		log.Debug("pgpolicy " + policyName + " not found so we will create it")
	} else {
		return err
	}

	// Create an instance of our CRD
	spec := crv1.PgpolicySpec{}
	spec.Name = policyName
	spec.URL = policyURL
	spec.SQL = policyFile

	newInstance := &crv1.Pgpolicy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: policyName,
		},
		Spec: spec,
	}

	err = kubeapi.Createpgpolicy(RESTClient,
		newInstance, apiserver.Namespace)

	if err != nil {
		return err
	}

	return err

}

// ShowPolicy ...
func ShowPolicy(RESTClient *rest.RESTClient, name string) crv1.PgpolicyList {
	policyList := crv1.PgpolicyList{}

	if name == "all" {
		//get a list of all policies
		err := kubeapi.Getpgpolicies(RESTClient,
			&policyList,
			apiserver.Namespace)
		if err != nil {
			return policyList
		}
	} else {
		policy := crv1.Pgpolicy{}
		_, err := kubeapi.Getpgpolicy(RESTClient,
			&policy, name, apiserver.Namespace)
		if err != nil {
			return policyList
		}
		policyList.Items = make([]crv1.Pgpolicy, 1)
		policyList.Items[0] = policy
	}

	return policyList

}

// DeletePolicy ...
func DeletePolicy(RESTClient *rest.RESTClient, policyName string) msgs.DeletePolicyResponse {
	resp := msgs.DeletePolicyResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	var err error

	policyList := crv1.PgpolicyList{}
	err = kubeapi.Getpgpolicies(RESTClient,
		&policyList, apiserver.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	policyFound := false
	log.Debug("deleting policy " + policyName)
	for _, policy := range policyList.Items {
		if policyName == "all" || policyName == policy.Spec.Name {
			policyFound = true
			err = kubeapi.Deletepgpolicy(RESTClient,
				policy.Spec.Name, apiserver.Namespace)
			if err == nil {
				log.Debug("deleted pgpolicy " + policy.Spec.Name)
			} else {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}

		}
	}

	if !policyFound {
		log.Debug("policy " + policyName + " not found")
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "policy " + policyName + " not found"
		return resp
	}
	return resp

}

// ApplyPolicy ...
// pgo apply mypolicy --selector=name=mycluster
func ApplyPolicy(request *msgs.ApplyPolicyRequest) msgs.ApplyPolicyResponse {
	var err error

	resp := msgs.ApplyPolicyResponse{}
	resp.Name = make([]string, 0)
	resp.Status.Msg = ""
	resp.Status.Code = msgs.Ok

	//validate policy
	err = util.ValidatePolicy(apiserver.RESTClient, apiserver.Namespace, request.Name)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "policy " + request.Name + " is not found, cancelling request"
		return resp
	}

	//get filtered list of Deployments
	selector := request.Selector + "," + util.LABEL_REPLICA + "=false"
	log.Debug("selector string=[" + selector + "]")

	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, apiserver.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if request.DryRun {
		for _, d := range deployments.Items {
			log.Debug("deployment : " + d.ObjectMeta.Name)
			resp.Name = append(resp.Name, d.ObjectMeta.Name)
		}
		return resp
	}

	labels := make(map[string]string)
	labels[request.Name] = "pgpolicy"

	for _, d := range deployments.Items {
		log.Debug("apply policy " + request.Name + " on deployment " + d.ObjectMeta.Name + " based on selector " + selector)

		err = util.ExecPolicy(apiserver.Clientset, apiserver.RESTClient, apiserver.Namespace, request.Name, d.ObjectMeta.Name)
		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		cl := crv1.Pgcluster{}
		_, err = kubeapi.Getpgcluster(apiserver.RESTClient,
			&cl, d.ObjectMeta.Name, apiserver.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp

		}

		var strategyMap map[string]cluster.Strategy
		strategyMap = make(map[string]cluster.Strategy)
		strategyMap["1"] = cluster.Strategy1{}

		strategy, ok := strategyMap[cl.Spec.Strategy]
		if ok {
			log.Debug("strategy found")
		} else {
			log.Error("invalid Strategy requested for cluster creation" + cl.Spec.Strategy)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "invalid strategy " + cl.Spec.Strategy
			return resp
		}

		err = strategy.UpdatePolicyLabels(apiserver.Clientset, d.ObjectMeta.Name, apiserver.Namespace, labels)
		if err != nil {
			log.Error(err)
		}

		//update the pgcluster crd labels with the new policy
		err = labelservice.PatchPgcluster(request.Name+"=pgpolicy", cl)
		if err != nil {
			log.Error(err)
		}

		resp.Name = append(resp.Name, d.ObjectMeta.Name)

	}
	return resp

}
