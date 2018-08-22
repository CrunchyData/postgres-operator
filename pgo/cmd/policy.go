package cmd

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
	Short: "Apply a policy",
	Long: `APPLY allows you to apply a Policy to a set of clusters. For example:

	pgo apply mypolicy1 --selector=name=mycluster
	pgo apply mypolicy1 --selector=someotherpolicy
	pgo apply mypolicy1 --selector=someotherpolicy --dry-run`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("apply called")
		if Selector == "" {
			fmt.Println("Error: Selector is required to apply a policy.")
			return
		}
		if len(args) == 0 {
			fmt.Println("Error: You must specify the name of a policy to apply.")
		} else {
			applyPolicy(args)
		}
	},
}

func init() {
	RootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	applyCmd.Flags().BoolVarP(&DryRun, "dry-run", "d", false, "Shows the clusters that the label would be applied to, without labelling them.")

}

func applyPolicy(args []string) {
	var err error

	if len(args) == 0 {
		fmt.Println("Error: A policy name argument is required.")
		return
	}

	if Selector == "" {
		fmt.Println("Error: The --selector flag is required.")
		return
	}

	r := new(msgs.ApplyPolicyRequest)
	r.Name = args[0]
	r.Selector = Selector
	r.DryRun = DryRun
	r.ClientVersion = msgs.PGO_VERSION

	jsonValue, _ := json.Marshal(r)

	url := APIServerURL + "/policies/apply"
	log.Debug("applyPolicy called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Println("Error: NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		fmt.Println("Error: Do: ", err)
		return
	}
	log.Debugf("%v\n", resp)
	StatusCheck(resp)

	defer resp.Body.Close()

	log.Debugf("response is %v\n", resp)

	var response msgs.ApplyPolicyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("%v\n", resp.Body)
		fmt.Println("Error: ", err)
		log.Println(err)
		return
	}

	if DryRun {
		fmt.Println("The label would have been applied on the following:")
	}

	if response.Status.Code == msgs.Ok {
		if len(response.Name) == 0 {
			fmt.Println("No clusters found.")
		} else {
			for _, v := range response.Name {
				fmt.Println("Applied policy on " + v)
			}
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

}
func showPolicy(args []string) {

	for _, v := range args {
		url := APIServerURL + "/policies/" + v + "?version=" + msgs.PGO_VERSION
		log.Debug("showPolicy called...[" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			fmt.Println("Error: NewRequest: ", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			fmt.Println("Error: Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)
		StatusCheck(resp)

		defer resp.Body.Close()

		var response msgs.ShowPolicyResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			fmt.Println("Error: ", err)
			log.Println(err)
			return
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

		if len(response.PolicyList.Items) == 0 {
			fmt.Println("No policies found.")
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
		fmt.Println("Error: A poliicy name argument is required.")
		return
	}
	var err error
	//PolicyURL, PolicyFile

	//create the request

	r := new(msgs.CreatePolicyRequest)
	r.Name = args[0]
	r.ClientVersion = msgs.PGO_VERSION

	if PolicyURL != "" {
		r.URL = PolicyURL
	}
	if PolicyFile != "" {
		r.SQL, err = getPolicyString(PolicyFile)

		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
	}

	jsonValue, _ := json.Marshal(r)

	url := APIServerURL + "/policies"
	log.Debug("createPolicy called...[" + url + "]")

	action := "POST"
	req, err := http.NewRequest(action, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Println("Error: NewRequest: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

	resp, err := httpclient.Do(req)
	if err != nil {
		fmt.Println("Error: Do: ", err)
		return
	}
	log.Debugf("%v\n", resp)
	StatusCheck(resp)

	defer resp.Body.Close()

	var response msgs.CreatePolicyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Println("Error: ", err)
		log.Println(err)
		return
	}

	if response.Status.Code == msgs.Ok {
		fmt.Println("Created policy.")
	} else {
		fmt.Println("Error: " + response.Status.Msg)
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

	for _, arg := range args {
		log.Debug("deleting policy " + arg)

		url := APIServerURL + "/policiesdelete/" + arg + "?version=" + msgs.PGO_VERSION

		log.Debug("delete policy called [" + url + "]")

		action := "GET"
		req, err := http.NewRequest(action, url, nil)
		if err != nil {
			fmt.Println("Error: NewRequest: ", err)
			return
		}
		req.SetBasicAuth(BasicAuthUsername, BasicAuthPassword)

		resp, err := httpclient.Do(req)
		if err != nil {
			fmt.Println("Error: Do: ", err)
			return
		}
		log.Debugf("%v\n", resp)
		StatusCheck(resp)

		defer resp.Body.Close()
		var response msgs.DeletePolicyResponse

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("%v\n", resp.Body)
			fmt.Println("Error: ", err)
			log.Println(err)
			return
		}

		if response.Status.Code == msgs.Ok {
			fmt.Println("Policy deleted.")
		} else {
			fmt.Println("Error: " + response.Status.Msg)
		}

	}
}
