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
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	//"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/policyservice"
	"github.com/crunchydata/postgres-operator/tpr"
	"github.com/spf13/viper"
	"io/ioutil"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	//"os/user"
	"strings"
)

func showPolicy(args []string) {
	//itemFound := false

	//each arg represents a policy name or the special 'all' value
	for _, arg := range args {
		fmt.Println("showing policy " + arg)
		url := "http://localhost:8080/policies/somename?showsecrets=true&other=thing"

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}

		client := &http.Client{}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}

		defer resp.Body.Close()

		var response policyservice.ShowPolicyResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Println(err)
		}

		fmt.Println("Name = ", response.Items[0].Name)

	}
}

func createPolicy(args []string) {

	//var err error

	for _, arg := range args {
		log.Debug("create policy called for " + arg)

		url := "http://localhost:8080/policies"

		cl := new(policyservice.CreatePolicyRequest)
		cl.Name = "newpolicy"
		jsonValue, _ := json.Marshal(cl)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
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
		fmt.Printf("%v\n", resp)

		fmt.Println("created PgPolicy " + arg)

	}
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

	for _, arg := range args {
		fmt.Println("deleting policy " + arg)
		url := "http://localhost:8080/policies/somename?showsecrets=true&other=thing"

		action := "DELETE"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			log.Fatal("NewRequest: ", err)
			return
		}

		client := &http.Client{}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}

		defer resp.Body.Close()

		var response policyservice.ShowPolicyResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Println(err)
		}

		fmt.Println("Name = ", response.Items[0].Name)

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

	url := "http://localhost:8080/policies/apply/somename"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	var response policyservice.ApplyResults

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
	}

	fmt.Printf("apply results %v\n", response.Results)

}
