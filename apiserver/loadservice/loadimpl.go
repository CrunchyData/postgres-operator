package loadservice

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
	//"errors"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/policyservice"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	operutil "github.com/crunchydata/postgres-operator/util"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
)

type loadJobTemplateFields struct {
	Name            string
	COImagePrefix   string
	COImageTag      string
	DbHost          string
	DbDatabase      string
	DbUser          string
	DbPass          string
	DbPort          string
	TableToLoad     string
	FilePath        string
	FileType        string
	PVCName         string
	SecurityContext string
}

// LoadConfig ...
//var LoadConfig string

// Load ...
// pgo load  --policies=jsonload --selector=name=mycluster --load-config=./sample-load-config.json
func Load(request *msgs.LoadRequest) msgs.LoadResponse {

	var err error
	resp := msgs.LoadResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	LoadConfigTemplate := loadJobTemplateFields{}

	var LoadCfg LoadConfig
	LoadCfg.getConf(bytes.NewBufferString(request.LoadConfig))

	err = LoadCfg.validate()
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	LoadConfigTemplate.COImagePrefix = LoadCfg.COImagePrefix
	LoadConfigTemplate.COImageTag = LoadCfg.COImageTag
	LoadConfigTemplate.DbDatabase = LoadCfg.DbDatabase
	LoadConfigTemplate.DbUser = LoadCfg.DbUser
	LoadConfigTemplate.DbPort = LoadCfg.DbPort
	LoadConfigTemplate.TableToLoad = LoadCfg.TableToLoad
	LoadConfigTemplate.FilePath = LoadCfg.FilePath
	LoadConfigTemplate.FileType = LoadCfg.FileType
	LoadConfigTemplate.PVCName = LoadCfg.PVCName
	LoadConfigTemplate.SecurityContext = LoadCfg.SecurityContext

	args := request.Args
	if request.Selector != "" {

		myselector, err := labels.Parse(request.Selector)
		if err != nil {
			log.Error("could not parse selector flag")
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		if myselector == nil {
		}

		//get the clusters list
		clusterList := crv1.PgclusterList{}
		err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
			&clusterList, request.Selector,
			apiserver.Namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		if len(clusterList.Items) == 0 {
			log.Debug("no clusters found")
		} else {
			newargs := make([]string, 0)
			for _, cluster := range clusterList.Items {
				newargs = append(newargs, cluster.Spec.Name)
			}
			args = newargs
		}

	}

	var policies []string
	if request.Policies != "" {
		policies = strings.Split(request.Policies, ",")
	}
	log.Debugf("policies to apply before loading are %v len=%d\n", request.Policies, len(policies))

	for _, arg := range args {
		for _, p := range policies {
			log.Debug("applying policy " + p + " to " + arg)
			//apply policies to this cluster
			applyReq := msgs.ApplyPolicyRequest{}
			applyReq.Name = p
			applyReq.Namespace = apiserver.Namespace
			applyReq.DryRun = false
			applyReq.Selector = "name=" + arg
			applyResp := policyservice.ApplyPolicy(&applyReq)
			if applyResp.Status.Code != msgs.Ok {
				log.Error("error in applying policy " + applyResp.Status.Msg)
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
		}

		//create the load job for this cluster
		log.Debug("created load for " + arg)
		err = createJob(arg, &LoadConfigTemplate)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

	}

	return resp

}

func createJob(clusterName string, template *loadJobTemplateFields) error {
	var err error

	randStr := operutil.GenerateRandString(3)
	template.Name = "pgo-load-" + clusterName + "-" + randStr
	template.DbHost = clusterName
	template.DbPass, err = operutil.GetSecretPassword(apiserver.Clientset, clusterName, crv1.RootSecretSuffix, apiserver.Namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	var doc2 bytes.Buffer
	err = apiserver.JobTemplate.Execute(&doc2, template)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	jobDocString := doc2.String()
	log.Debug(jobDocString)

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return err
	}

	err = kubeapi.CreateJob(apiserver.Clientset, &newjob, apiserver.Namespace)

	return err

}
