package policyservice

/*
Copyright 2017-2019 Crunchy Data Solutions, Inc.
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
func CreatePolicy(RESTClient *rest.RESTClient, policyName, policyURL, policyFile, ns string) (bool, error) {

	var found bool
	log.Debugf("create policy called for %s", policyName)
	result := crv1.Pgpolicy{}

	// error if it already exists
	found, err := kubeapi.Getpgpolicy(RESTClient,
		&result, policyName, ns)
	if err == nil {
		log.Debugf("pgpolicy %s was found so we will not create it", policyName)
		return true, err
	} else if !found {
		log.Debugf("pgpolicy %s was not found so we will create it", policyName)
	} else {
		return false, err
	}

	// Create an instance of our CRD
	spec := crv1.PgpolicySpec{}
	spec.Namespace = ns
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
		newInstance, ns)

	if err != nil {
		return false, err
	}

	return false, err

}

// ShowPolicy ...
func ShowPolicy(RESTClient *rest.RESTClient, name, ns string) crv1.PgpolicyList {
	policyList := crv1.PgpolicyList{}

	if name == "all" {
		//get a list of all policies
		err := kubeapi.Getpgpolicies(RESTClient,
			&policyList,
			ns)
		if err != nil {
			return policyList
		}
	} else {
		policy := crv1.Pgpolicy{}
		_, err := kubeapi.Getpgpolicy(RESTClient,
			&policy, name, ns)
		if err != nil {
			return policyList
		}
		policyList.Items = make([]crv1.Pgpolicy, 1)
		policyList.Items[0] = policy
	}

	return policyList

}

// DeletePolicy ...
func DeletePolicy(RESTClient *rest.RESTClient, policyName, ns string) msgs.DeletePolicyResponse {
	resp := msgs.DeletePolicyResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	var err error

	policyList := crv1.PgpolicyList{}
	err = kubeapi.Getpgpolicies(RESTClient,
		&policyList, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	policyFound := false
	log.Debugf("deleting policy %s", policyName)
	for _, policy := range policyList.Items {
		if policyName == "all" || policyName == policy.Spec.Name {
			policyFound = true
			err = kubeapi.Deletepgpolicy(RESTClient,
				policy.Spec.Name, ns)
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
func ApplyPolicy(request *msgs.ApplyPolicyRequest, ns string) msgs.ApplyPolicyResponse {
	var err error

	resp := msgs.ApplyPolicyResponse{}
	resp.Name = make([]string, 0)
	resp.Status.Msg = ""
	resp.Status.Code = msgs.Ok

	//validate policy
	err = util.ValidatePolicy(apiserver.RESTClient, ns, request.Name)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "policy " + request.Name + " is not found, cancelling request"
		return resp
	}

	//get filtered list of Deployments
	//selector := request.Selector + "," + util.LABEL_PRIMARY + "=true"
	selector := request.Selector
	log.Debugf("selector string=[%s]", selector)

	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	if request.DryRun {
		for _, d := range deployments.Items {
			log.Debugf("deployment : %s", d.ObjectMeta.Name)
			resp.Name = append(resp.Name, d.ObjectMeta.Name)
		}
		return resp
	}

	labels := make(map[string]string)
	labels[request.Name] = "pgpolicy"

	for _, d := range deployments.Items {
		if d.ObjectMeta.Labels[util.LABEL_SERVICE_NAME] != d.ObjectMeta.Labels[util.LABEL_PG_CLUSTER] {
			log.Debug("skipping apply policy on deployment %s", d.Name)
			continue
			//skip non primary deployments
		}

		log.Debugf("apply policy %s on deployment %s based on selector %s", request.Name, d.ObjectMeta.Name, selector)

		err = util.ExecPolicy(apiserver.Clientset, apiserver.RESTClient, ns, request.Name, d.ObjectMeta.Labels[util.LABEL_SERVICE_NAME])
		if err != nil {
			log.Error(err)
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		cl := crv1.Pgcluster{}
		_, err = kubeapi.Getpgcluster(apiserver.RESTClient,
			&cl, d.ObjectMeta.Labels[util.LABEL_SERVICE_NAME], ns)
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

		err = strategy.UpdatePolicyLabels(apiserver.Clientset, d.ObjectMeta.Name, ns, labels)
		if err != nil {
			log.Error(err)
		}

		//update the pgcluster crd labels with the new policy
		err = labelservice.PatchPgcluster(request.Name+"="+util.LABEL_PGPOLICY, cl, ns)
		if err != nil {
			log.Error(err)
		}

		resp.Name = append(resp.Name, d.ObjectMeta.Name)

	}
	return resp

}
