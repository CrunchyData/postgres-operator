package policyservice

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
	"errors"
	log "github.com/Sirupsen/logrus"
	"k8s.io/client-go/rest"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	cluster "github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePolicy ...
func CreatePolicy(RESTClient *rest.RESTClient, Namespace, policyName, policyURL, policyFile string) error {
	var err error

	log.Debug("create policy called for " + policyName)
	result := crv1.Pgpolicy{}

	// error if it already exists
	err = RESTClient.Get().
		Resource(crv1.PgpolicyResourcePlural).
		Namespace(Namespace).
		Name(policyName).
		Do().
		Into(&result)
	if err == nil {
		log.Infoln("pgpolicy " + policyName + " was found so we will not create it")
		return err
	} else if kerrors.IsNotFound(err) {
		log.Debug("pgpolicy " + policyName + " not found so we will create it")
	} else {
		log.Error("error getting pgpolicy " + policyName + err.Error())
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

	err = RESTClient.Post().
		Resource(crv1.PgpolicyResourcePlural).
		Namespace(Namespace).
		Body(newInstance).
		Do().Into(&result)

	if err != nil {
		log.Error(" in creating Pgpolicy instance" + err.Error())
		return err
	}
	log.Infoln("created Pgpolicy " + policyName)
	return err

}

// ShowPolicy ...
func ShowPolicy(RESTClient *rest.RESTClient, Namespace string, name string) crv1.PgpolicyList {
	policyList := crv1.PgpolicyList{}

	if name == "all" {
		//get a list of all policies
		err := RESTClient.Get().
			Resource(crv1.PgpolicyResourcePlural).
			Namespace(Namespace).
			Do().Into(&policyList)
		if err != nil {
			log.Error("error getting list of policies" + err.Error())
			return policyList
		}
	} else {
		policy := crv1.Pgpolicy{}
		err := RESTClient.Get().
			Resource(crv1.PgpolicyResourcePlural).
			Namespace(Namespace).
			Name(name).
			Do().Into(&policy)
		if err != nil {
			log.Error("error getting list of policies" + err.Error())
			return policyList
		}
		policyList.Items = make([]crv1.Pgpolicy, 1)
		policyList.Items[0] = policy
	}

	return policyList

}

// DeletePolicy ...
func DeletePolicy(Namespace string, RESTClient *rest.RESTClient, args []string) error {
	var err error
	// Fetch a list of our policy CRDs
	policyList := crv1.PgpolicyList{}
	err = RESTClient.Get().Resource(crv1.PgpolicyResourcePlural).Do().Into(&policyList)
	if err != nil {
		log.Error("error getting policy list" + err.Error())
		return err
	}

	//to remove a policy, you just have to remove
	//the pgpolicy object, the operator will do the actual deletes
	for _, arg := range args {
		policyFound := false
		log.Debug("deleting policy " + arg)
		for _, policy := range policyList.Items {
			if arg == "all" || arg == policy.Spec.Name {
				policyFound = true
				err = RESTClient.Delete().
					Resource(crv1.PgpolicyResourcePlural).
					Namespace(Namespace).
					Name(policy.Spec.Name).
					Do().
					Error()
				if err == nil {
					log.Infoln("deleted pgpolicy " + policy.Spec.Name)
				} else {
					log.Error("error deleting pgpolicy " + arg + err.Error())
					return err
				}

			}
		}
		if !policyFound {
			log.Infoln("policy " + arg + " not found")
		}
	}
	return err

}

// ApplyPolicy ...
// pgo apply mypolicy --selector=name=mycluster
func ApplyPolicy(request *msgs.ApplyPolicyRequest) ([]string, error) {
	var err error
	clusters := make([]string, 0)

	//validate policy
	err = util.ValidatePolicy(apiserver.RESTClient, request.Namespace, request.Name)
	if err != nil {
		return clusters, errors.New("policy " + request.Name + " is not found, cancelling request")
	}

	//get filtered list of Deployments
	sel := request.Selector + ",!replica"
	log.Debug("selector string=[" + sel + "]")
	lo := meta_v1.ListOptions{LabelSelector: sel}
	deployments, err := apiserver.Clientset.ExtensionsV1beta1().Deployments(request.Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return clusters, err
	}

	if request.DryRun {
		for _, d := range deployments.Items {
			log.Infoln("deployment : " + d.ObjectMeta.Name)
			clusters = append(clusters, d.ObjectMeta.Name)
		}
		return clusters, err
	}

	labels := make(map[string]string)
	labels[request.Name] = "pgpolicy"

	for _, d := range deployments.Items {
		log.Debug("apply policy " + request.Name + " on deployment " + d.ObjectMeta.Name + " based on selector " + sel)

		err = util.ExecPolicy(apiserver.Clientset, apiserver.RESTClient, request.Namespace, request.Name, d.ObjectMeta.Name)
		if err != nil {
			log.Error(err)
			return clusters, err
		}

		cl := crv1.Pgcluster{}
		err = apiserver.RESTClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(request.Namespace).
			Name(d.ObjectMeta.Name).
			Do().
			Into(&cl)
		if err != nil {
			log.Error(err)
			return clusters, err

		}

		var strategyMap map[string]cluster.Strategy
		strategyMap = make(map[string]cluster.Strategy)
		strategyMap["1"] = cluster.Strategy1{}

		strategy, ok := strategyMap[cl.Spec.Strategy]
		if ok {
			log.Info("strategy found")
		} else {
			log.Error("invalid Strategy requested for cluster creation" + cl.Spec.Strategy)
			return clusters, err
		}

		err = strategy.UpdatePolicyLabels(apiserver.Clientset, d.ObjectMeta.Name, request.Namespace, labels)
		if err != nil {
			log.Error(err)
		}

		clusters = append(clusters, d.ObjectMeta.Name)

	}
	return clusters, err

}
