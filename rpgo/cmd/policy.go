package cmd

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
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiservermsgs"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os/user"
	"strings"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "apply a Policy",
	Long: `APPLY allows you to apply a Policy to a set of clusters
For example:

pgo apply mypolicy1 --selector=name=mycluster
pgo apply mypolicy1 --selector=someotherpolicy
pgo apply mypolicy1 --selector=someotherpolicy --dry-run
.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("apply called")
		if Selector == "" {
			log.Error("selector is required to apply a policy")
			return
		}
		if len(args) == 0 {
			log.Error(`You must specify the name of a policy to apply.`)
		} else {
			applyPolicy(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering ")
	applyCmd.Flags().BoolVarP(&DryRun, "dry-run", "d", false, "--dry-run shows clusters that policy would be applied to but does not actually apply them")

}

func applyPolicy(args []string) {
	var err error

	if len(args) == 0 {
		log.Error("policy name argument is required")
		return
	}

	if Selector == "" {
		log.Error("--selector flag is required")
		return
	}

	if Namespace == "" {
		log.Error("Namespace can not be empty")
		return
	}

	r := new(msgs.ApplyPolicyRequest)
	r.Name = args[0]
	r.Selector = Selector
	r.DryRun = DryRun
	r.Namespace = Namespace

	jsonValue, _ := json.Marshal(r)

	url := APIServerURL + "/policies/apply"
	log.Debug("applyPolicy called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	log.Printf("response is %v\n", resp)

	if DryRun {
		fmt.Println("would have applied policy on " + "something")
	}
	//for v := range resp.Name {
	//fmt.Println("applied policy on " + v)
	//}

}
func showPolicy(args []string) {
	if Namespace == "" {
		log.Error("Namespace can not be empty")
		return
	}

	for _, v := range args {
		url := APIServerURL + "/policies/" + v + "?namespace=" + Namespace
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
			fmt.Println(TreeBranch + "url : " + policy.Spec.URL)
			fmt.Println(TreeBranch + "status : " + policy.Spec.Status)
			fmt.Println(TreeTrunk + "sql : " + policy.Spec.SQL)
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

	url := APIServerURL + "/policies"
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
