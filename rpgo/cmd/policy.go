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
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	//"github.com/crunchydata/kraken/util"
	"encoding/json"
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"github.com/crunchydata/kraken/apiservermsgs"
	"github.com/crunchydata/kraken/util"
	"net/http"
	//"net/url"

	"github.com/spf13/viper"
	"io/ioutil"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/user"
	"strings"
)

func showPolicy(args []string) {

	//	jsonstr, _ := json.Marshal(args)
	//	values := url.Values{"args": {string(jsonstr[:])}}
	//	encoded := values.Encode()

	for _, v := range args {
		//b := new(bytes.Buffer)
		//json.NewEncoder(b).Encode(args)
		//s := new(bytes.Buffer)
		//json.NewEncoder(s).Encode("foodaddy")

		//url := APISERVER_URL + "/policies/somename?" + encoded
		//fmt.Println("labelselector=" + LabelSelector)
		//url := APISERVER_URL + "/policies/" + b.String() + "?selector=" + s.String()
		//read_line := strings.TrimSuffix(b.String(), "\n")
		//url := APISERVER_URL + "/policies/" + read_line + "?selector=name=foodaddy"
		if Namespace == "" {
			log.Error("Namespace can not be empty")
			return
		}
		url := APISERVER_URL + "/policies/" + v + "?namespace=" + Namespace
		log.Debug("showPolicy called...[" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			//log.Info("here after new req")
			log.Fatal("NewRequest: ", err)
			return
		}

		client := &http.Client{}
		//log.Info("here after new client")

		resp, err := client.Do(req)
		//log.Info("here after Do")
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}

		//log.Info("here after Do2")
		defer resp.Body.Close()

		var response apiservermsgs.ShowPolicyResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if len(response.PolicyList.Items) == 0 {
			fmt.Println("no policies found")
			return
		}

		log.Debugf("response = %v\n", response)

		for _, policy := range response.PolicyList.Items {
			fmt.Println("")
			fmt.Println("policy : " + policy.Spec.Name)
			fmt.Println(TREE_BRANCH + "url : " + policy.Spec.Url)
			fmt.Println(TREE_BRANCH + "status : " + policy.Spec.Status)
			fmt.Println(TREE_TRUNK + "sql : " + policy.Spec.Sql)
		}
	}

}

func createPolicy(args []string) {

	if len(args) == 0 {
		log.Error("policy name argument is required")
		return
	}
	var err error
	//PolicyURL, PolicyFile
	if Namespace == "" {
		log.Error("Namespace can not be empty")
		return
	}

	//create the request

	r := new(apiservermsgs.CreatePolicyRequest)
	r.Name = args[0]

	if PolicyURL != "" {
		r.URL = PolicyURL
	}
	if PolicyFile != "" {
		r.SQL, err = getPolicyString(PolicyFile)

		if err != nil {
			log.Error(err)
			return
		}
	}

	r.Namespace = Namespace

	jsonValue, _ := json.Marshal(r)

	url := APISERVER_URL + "/policies"
	log.Debug("createPolicy called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		//log.Info("here after new req")
		log.Fatal("NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	//log.Info("here after new client")

	resp, err := client.Do(req)
	//log.Info("here after Do")
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	/**
	var response apiservermsgs.CreatePolicyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Error(err)
		log.Println(err)
		return
	}
	*/
	log.Printf("response is %v\n", resp)

	fmt.Println("created policy")

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
				Resource(crv1.PgpolicyResourcePlural).
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
				Resource(crv1.PgpolicyResourcePlural).
				Namespace(Namespace).
				Body(newInstance).
				Do().Into(&result)
			if err != nil {
				log.Error("error in creating Pgpolicylog TPR instance", err.Error())
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
