package cmd

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	"fmt"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
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

		if Namespace == "" {
			Namespace = PGONamespace
		}

		if Selector == "" {
			fmt.Println("Error: Selector is required to apply a policy.")
			return
		}
		if len(args) == 0 {
			fmt.Println("Error: You must specify the name of a policy to apply.")
		} else {
			applyPolicy(args, Namespace)
		}
	},
}

func init() {
	RootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringVarP(&Selector, "selector", "s", "", "The selector to use for cluster filtering.")
	applyCmd.Flags().BoolVarP(&DryRun, "dry-run", "", false, "Shows the clusters that the label would be applied to, without labelling them.")

}

func applyPolicy(args []string, ns string) {
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
	r.Namespace = ns
	r.DryRun = DryRun
	r.ClientVersion = msgs.PGO_VERSION

	response, err := api.ApplyPolicy(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
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
func showPolicy(args []string, ns string) {

	r := new(msgs.ShowPolicyRequest)
	r.Selector = Selector
	r.Namespace = ns
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION

	if len(args) == 0 && AllFlag {
		args = []string{""}
	}

	for _, v := range args {
		r.Policyname = v

		response, err := api.ShowPolicy(httpclient, &SessionCredentials, r)

		if err != nil {
			fmt.Println("Error: " + err.Error())
			os.Exit(2)
		}

		if response.Status.Code != msgs.Ok {
			fmt.Println("Error: " + response.Status.Msg)
			os.Exit(2)
		}

		if len(response.PolicyList.Items) == 0 {
			fmt.Println("No policies found.")
			return
		}

		log.Debugf("response = %v", response)

		for _, policy := range response.PolicyList.Items {
			fmt.Println("")
			fmt.Println("policy : " + policy.Spec.Name)
			fmt.Println(TreeBranch + "url : " + policy.Spec.URL)
			fmt.Println(TreeBranch + "status : " + policy.Spec.Status)
			fmt.Println(TreeTrunk + "sql : " + policy.Spec.SQL)
		}
	}

}

func createPolicy(args []string, ns string) {

	if len(args) == 0 {
		fmt.Println("Error: A poliicy name argument is required.")
		return
	}
	var err error
	//create the request
	r := new(msgs.CreatePolicyRequest)
	r.Name = args[0]
	r.Namespace = ns
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

	response, err := api.CreatePolicy(httpclient, &SessionCredentials, r)

	log.Debugf("response is %v", response)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
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

func deletePolicy(args []string, ns string) {

	log.Debugf("deletePolicy called %v", args)

	r := msgs.DeletePolicyRequest{}
	r.Selector = Selector
	r.AllFlag = AllFlag
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns
	if AllFlag {
		args = make([]string, 1)
		args[0] = "all"
	}

	for _, arg := range args {
		r.PolicyName = arg
		log.Debugf("deleting policy %s", arg)

		response, err := api.DeletePolicy(httpclient, &r, &SessionCredentials)
		if err != nil {
			fmt.Println("Error: " + err.Error())
		}

		if response.Status.Code == msgs.Ok {
			for _, v := range response.Results {
				fmt.Println(v)
			}
		} else {
			fmt.Println("Error: " + response.Status.Msg)
		}

	}
}
