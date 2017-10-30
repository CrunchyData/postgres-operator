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
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
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

	log.Debugf("response is %v\n", resp)

	var response msgs.ApplyPolicyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		log.Error(err)
		log.Println(err)
		return
	}

	if DryRun {
		fmt.Println("would have applied policy on " + "something")
	}

	if response.Status.Code == msgs.Ok {
		for _, v := range response.Name {
			fmt.Println("applied policy on " + v)
		}
	} else {
		fmt.Println(RED(response.Status.Msg))
		os.Exit(2)
	}

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

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Do: ", err)
			return
		}

		//log.Info("here after Do2")
		defer resp.Body.Close()

		var response msgs.ShowPolicyResponse

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

	r := new(msgs.CreatePolicyRequest)
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

	var response msgs.CreatePolicyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Error(err)
		log.Println(err)
		return
	}

	if response.Status.Code == msgs.Ok {
		fmt.Println("created policy")
	} else {
		fmt.Println(RED(response.Status.Msg))
		os.Exit(2)
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

	log.Debugf("deletePolicy called %v\n", args)

	if Namespace == "" {
		log.Error("Namespace can not be empty")
		return
	}

	for _, arg := range args {
		log.Debug("deleting policy " + arg)

		url := APIServerURL + "/policies/" + arg + "?namespace=" + Namespace

		log.Debug("delete policy called [" + url + "]")

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
		var response msgs.DeletePolicyResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			log.Error(err)
			log.Println(err)
			return
		}

		if response.Status.Code == msgs.Ok {
			fmt.Println("policy deleted")
		} else {
			fmt.Println(RED(response.Status.Msg))
		}

	}
}
