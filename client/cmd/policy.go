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
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/viper"
	"io/ioutil"
	"k8s.io/client-go/pkg/api"
	kerrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"os/user"
	"strings"
)

func showPolicy(args []string) {
	//get a list of all policies
	policyList := tpr.PgPolicyList{}
	err := Tprclient.Get().
		Resource(tpr.POLICY_RESOURCE).
		Namespace(Namespace).
		Do().Into(&policyList)
	if err != nil {
		log.Error("error getting list of policies" + err.Error())
		return
	}

	if len(policyList.Items) == 0 {
		fmt.Println("no policies found")
		return
	}

	itemFound := false

	//each arg represents a policy name or the special 'all' value
	for _, arg := range args {
		for _, policy := range policyList.Items {
			fmt.Println("")
			if arg == "all" || policy.Spec.Name == arg {
				itemFound = true
				log.Debug("listing policy " + arg)
				fmt.Println("policy : " + policy.Spec.Name)
				fmt.Println(TREE_BRANCH + "url : " + policy.Spec.Url)
				fmt.Println(TREE_BRANCH + "status : " + policy.Spec.Status)
				fmt.Println(TREE_TRUNK + "sql : " + policy.Spec.Sql)
			}
		}
		if !itemFound {
			fmt.Println(arg + " was not found")
		}
		itemFound = false
	}
}

func createPolicy(args []string) {

	var err error

	for _, arg := range args {
		log.Debug("create policy called for " + arg)
		result := tpr.PgPolicy{}

		// error if it already exists
		err = Tprclient.Get().
			Resource(tpr.POLICY_RESOURCE).
			Namespace(Namespace).
			Name(arg).
			Do().
			Into(&result)
		if err == nil {
			log.Debug("pgpolicy " + arg + " was found so we will not create it")
			break
		} else if kerrors.IsNotFound(err) {
			log.Debug("pgpolicy " + arg + " not found so we will create it")
		} else {
			log.Error("error getting pgpolicy " + arg + err.Error())
			break
		}

		// Create an instance of our TPR
		newInstance, err := getPolicyParams(arg)
		if err != nil {
			log.Error(" error in policy parameters ")
			log.Error(err.Error())
			return
		}

		err = Tprclient.Post().
			Resource(tpr.POLICY_RESOURCE).
			Namespace(Namespace).
			Body(newInstance).
			Do().Into(&result)

		if err != nil {
			log.Error(" in creating PgPolicy instance" + err.Error())
		}
		fmt.Println("created PgPolicy " + arg)

	}
}

func getPolicyParams(name string) (*tpr.PgPolicy, error) {

	var err error

	spec := tpr.PgPolicySpec{}
	spec.Name = name

	if PolicyURL != "" {
		spec.Url = PolicyURL
	}
	if PolicyFile != "" {
		spec.Sql, err = getPolicyString(PolicyFile)

		if err != nil {
			return &tpr.PgPolicy{}, err
		}
	}

	newInstance := &tpr.PgPolicy{
		Metadata: api.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}

	return newInstance, err
}

func getPolicyString(filename string) (string, error) {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(buf), err
}

func deletePolicy(args []string) {
	// Fetch a list of our policy TPRs
	policyList := tpr.PgPolicyList{}
	err := Tprclient.Get().Resource(tpr.POLICY_RESOURCE).Do().Into(&policyList)
	if err != nil {
		log.Error("error getting policy list" + err.Error())
		return
	}

	//to remove a policy, you just have to remove
	//the pgpolicy object, the operator will do the actual deletes
	for _, arg := range args {
		policyFound := false
		log.Debug("deleting policy " + arg)
		for _, policy := range policyList.Items {
			if arg == "all" || arg == policy.Spec.Name {
				policyFound = true
				err = Tprclient.Delete().
					Resource(tpr.POLICY_RESOURCE).
					Namespace(Namespace).
					Name(policy.Spec.Name).
					Do().
					Error()
				if err != nil {
					log.Error("error deleting pgpolicy " + arg + err.Error())
				} else {
					fmt.Println("deleted pgpolicy " + policy.Spec.Name)
				}

			}
		}
		if !policyFound {
			fmt.Println("policy " + arg + " not found")
		}
	}
}

func validateConfigPolicies() error {
	var err error
	var configPolicies string
	if PoliciesFlag == "" {
		configPolicies = viper.GetString("CLUSTER.POLICIES")
	} else {
		configPolicies = PoliciesFlag
	}
	if configPolicies == "" {
		log.Debug("no policies are specified")
		return err
	}

	policies := strings.Split(configPolicies, ",")

	for _, v := range policies {
		result := tpr.PgPolicy{}

		// error if it already exists
		err = Tprclient.Get().
			Resource(tpr.POLICY_RESOURCE).
			Namespace(Namespace).
			Name(v).
			Do().
			Into(&result)
		if err == nil {
			log.Debug("policy " + v + " was found in catalog")
		} else if kerrors.IsNotFound(err) {
			log.Error("policy " + v + " specified in configuration was not found")
			return err
		} else {
			log.Error("error getting pgpolicy " + v + err.Error())
			return err
		}

	}

	return err
}

func applyPolicy(policies []string) {
	var err error
	//validate policies
	labels := make(map[string]string)
	for _, p := range policies {
		err = util.ValidatePolicy(Tprclient, Namespace, p)
		if err != nil {
			log.Error("policy " + p + " is not found, cancelling request")
			return
		}

		labels[p] = "pgpolicy"
	}

	//get filtered list of Deployments
	sel := Selector + ",!replica"
	log.Debug("selector string=[" + sel + "]")
	lo := v1.ListOptions{LabelSelector: sel}
	deployments, err := Clientset.Deployments(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return
	}

	var newInstance *tpr.PgPolicylog
	for _, d := range deployments.Items {
		fmt.Println("deployment : " + d.ObjectMeta.Name)
		for _, p := range policies {
			log.Debug("apply policy " + p + " on deployment " + d.ObjectMeta.Name + " based on selector " + sel)

			newInstance, err = getPolicylog(p, d.ObjectMeta.Name)

			result := tpr.PgPolicylog{}
			err = Tprclient.Post().
				Resource(tpr.POLICY_LOG_RESOURCE).
				Namespace(Namespace).
				Body(newInstance).
				Do().Into(&result)
			if err != nil {
				log.Error("error in creating PgPolicylog TPR instance", err.Error())
			} else {
				fmt.Println("created PgPolicylog " + result.Metadata.Name)
			}

		}

	}

}

func getPolicylog(policyname, clustername string) (*tpr.PgPolicylog, error) {
	u, err := user.Current()
	if err != nil {
		log.Error(err.Error())
	}

	spec := tpr.PgPolicylogSpec{}
	spec.PolicyName = policyname
	spec.Username = u.Name
	spec.ClusterName = clustername

	newInstance := &tpr.PgPolicylog{
		Metadata: api.ObjectMeta{
			Name: policyname + clustername,
		},
		Spec: spec,
	}
	return newInstance, err

}
