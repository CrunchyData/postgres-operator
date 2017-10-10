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
	//"github.com/crunchydata/kraken/util"
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"github.com/crunchydata/kraken/util"

	"github.com/spf13/viper"
	"io/ioutil"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/user"
	"strings"
)

func showPolicy(args []string) {
	//get a list of all policies
	policyList := crv1.PgpolicyList{}
	err := RestClient.Get().
		Resource(crv1.PgpolicyResourcePlural).
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
		result := crv1.Pgpolicy{}

		// error if it already exists
		err = RestClient.Get().
			Resource(crv1.PgpolicyResourcePlural).
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

		// Create an instance of our CRD
		newInstance, err := getPolicyParams(arg)
		if err != nil {
			log.Error(" error in policy parameters ")
			log.Error(err.Error())
			return
		}

		err = RestClient.Post().
			Resource(crv1.PgpolicyResourcePlural).
			Namespace(Namespace).
			Body(newInstance).
			Do().Into(&result)

		if err != nil {
			log.Error(" in creating Pgpolicy instance" + err.Error())
		}
		fmt.Println("created Pgpolicy " + arg)

	}
}

func getPolicyParams(name string) (*crv1.Pgpolicy, error) {

	var err error

	spec := crv1.PgpolicySpec{}
	spec.Name = name

	if PolicyURL != "" {
		spec.Url = PolicyURL
	}
	if PolicyFile != "" {
		spec.Sql, err = getPolicyString(PolicyFile)

		if err != nil {
			return &crv1.Pgpolicy{}, err
		}
	}

	newInstance := &crv1.Pgpolicy{
		ObjectMeta: meta_v1.ObjectMeta{
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
	// Fetch a list of our policy CRDs
	policyList := crv1.PgpolicyList{}
	err := RestClient.Get().Resource(crv1.PgpolicyResourcePlural).Do().Into(&policyList)
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
				err = RestClient.Delete().
					Resource(crv1.PgpolicyResourcePlural).
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
		result := crv1.Pgpolicy{}

		// error if it already exists
		err = RestClient.Get().
			Resource(crv1.PgpolicyResourcePlural).
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
		err = util.ValidatePolicy(RestClient, Namespace, p)
		if err != nil {
			log.Error("policy " + p + " is not found, cancelling request")
			return
		}

		labels[p] = "pgpolicy"
	}

	//get filtered list of Deployments
	sel := Selector + ",!replica"
	log.Debug("selector string=[" + sel + "]")
	lo := meta_v1.ListOptions{LabelSelector: sel}
	deployments, err := Clientset.ExtensionsV1beta1().Deployments(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return
	}

	if DryRun {
		fmt.Println("policy would be applied to the following clusters:")
		for _, d := range deployments.Items {
			fmt.Println("deployment : " + d.ObjectMeta.Name)
		}
		return
	}

	var newInstance *crv1.Pgpolicylog
	for _, d := range deployments.Items {
		fmt.Println("deployment : " + d.ObjectMeta.Name)
		for _, p := range policies {
			log.Debug("apply policy " + p + " on deployment " + d.ObjectMeta.Name + " based on selector " + sel)

			newInstance, err = getPolicylog(p, d.ObjectMeta.Name)

			result := crv1.Pgpolicylog{}
			err = RestClient.Get().
				Resource(crv1.PgpolicylogResourcePlural).
				Namespace(Namespace).
				Name(newInstance.ObjectMeta.Name).
				Do().Into(&result)
			if err == nil {
				fmt.Println(p + " already applied to " + d.ObjectMeta.Name)
				break
			} else {
				if kerrors.IsNotFound(err) {
				} else {
					log.Error(err)
					break
				}
			}

			result = crv1.Pgpolicylog{}
			err = RestClient.Post().
				Resource(crv1.PgpolicylogResourcePlural).
				Namespace(Namespace).
				Body(newInstance).
				Do().Into(&result)
			if err != nil {
				log.Error("error in creating Pgpolicylog CRD instance", err.Error())
			} else {
				fmt.Println("created Pgpolicylog " + result.ObjectMeta.Name)
			}

		}

	}

}

func getPolicylog(policyname, clustername string) (*crv1.Pgpolicylog, error) {
	u, err := user.Current()
	if err != nil {
		log.Error(err.Error())
	}

	spec := crv1.PgpolicylogSpec{}
	spec.PolicyName = policyname
	spec.Username = u.Name
	spec.ClusterName = clustername

	newInstance := &crv1.Pgpolicylog{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: policyname + clustername,
		},
		Spec: spec,
	}
	return newInstance, err

}
