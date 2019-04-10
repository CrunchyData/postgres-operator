package loadservice

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
	"bytes"
	"encoding/json"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/policyservice"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	operutil "github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
)

type loadJobTemplateFields struct {
	Name               string
	PGOImagePrefix     string
	PGOImageTag        string
	DbHost             string
	DbDatabase         string
	DbUser             string
	DbPass             string
	DbPort             string
	TableToLoad        string
	FilePath           string
	FileType           string
	PVCName            string
	SecurityContext    string
	ContainerResources string
}

type containerResourcesTemplateFields struct {
	RequestsMemory, RequestsCPU string
	LimitsMemory, LimitsCPU     string
}

// LoadConfig ...
//var LoadConfig string

// Load ...
// pgo load  --policies=jsonload --selector=name=mycluster --load-config=./sample-load-config.json
func Load(request *msgs.LoadRequest, ns string) msgs.LoadResponse {

	var err error
	resp := msgs.LoadResponse{}
	resp.Status.Code = msgs.Ok
	resp.Results = make([]string, 0)
	resp.Status.Msg = ""

	LoadConfigTemplate := loadJobTemplateFields{}

	var LoadCfg LoadConfig
	_, err = LoadCfg.getConf(bytes.NewBufferString(request.LoadConfig))
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	err = LoadCfg.validate()
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	LoadConfigTemplate.PGOImagePrefix = LoadCfg.PGOImagePrefix
	LoadConfigTemplate.PGOImageTag = LoadCfg.PGOImageTag
	LoadConfigTemplate.DbDatabase = LoadCfg.DbDatabase
	LoadConfigTemplate.DbUser = LoadCfg.DbUser
	LoadConfigTemplate.DbPort = LoadCfg.DbPort
	LoadConfigTemplate.TableToLoad = LoadCfg.TableToLoad
	LoadConfigTemplate.FilePath = LoadCfg.FilePath
	LoadConfigTemplate.FileType = LoadCfg.FileType
	LoadConfigTemplate.PVCName = LoadCfg.PVCName
	LoadConfigTemplate.SecurityContext = operutil.CreateSecContext(LoadCfg.FSGroup, LoadCfg.SupplementalGroup)
	LoadConfigTemplate.ContainerResources = ""
	if apiserver.Pgo.DefaultLoadResources != "" {
		tmp, err := apiserver.Pgo.GetContainerResource(apiserver.Pgo.DefaultLoadResources)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		LoadConfigTemplate.ContainerResources = apiserver.GetContainerResourcesJSON(&tmp)
	}

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
			ns)
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
	log.Debugf("policies to apply before loading are %v len=%d", request.Policies, len(policies))

	var jobName string
	for _, arg := range args {
		for _, p := range policies {
			log.Debugf("applying policy %s to %s", p, arg)
			//apply policies to this cluster
			applyReq := msgs.ApplyPolicyRequest{}
			applyReq.Name = p
			applyReq.Namespace = ns
			applyReq.DryRun = false
			applyReq.Selector = "name=" + arg
			applyResp := policyservice.ApplyPolicy(&applyReq, ns)
			if applyResp.Status.Code != msgs.Ok {
				log.Error("error in applying policy " + applyResp.Status.Msg)
				resp.Status.Code = msgs.Error
				resp.Status.Msg = err.Error()
				return resp
			}
		}

		//create the load job for this cluster
		log.Debugf("creating load job for %s", arg)
		jobName, err = createJob(arg, &LoadConfigTemplate, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "created Job "+jobName)

	}

	log.Debugf("on return load results is %v", resp.Results)
	return resp

}

func createJob(clusterName string, template *loadJobTemplateFields, ns string) (string, error) {
	var err error

	randStr := operutil.GenerateRandString(3)
	template.Name = "pgo-load-" + clusterName + "-" + randStr
	template.DbHost = clusterName
	template.DbPass, err = operutil.GetSecretPassword(apiserver.Clientset, clusterName, crv1.RootSecretSuffix, ns)
	if err != nil {
		log.Error(err)
		return "", err
	}

	var doc2 bytes.Buffer
	err = config.LoadTemplate.Execute(&doc2, template)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}
	jobDocString := doc2.String()
	log.Debug(jobDocString)

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("error unmarshalling json into Job " + err.Error())
		return "", err
	}

	var jobName string
	jobName, err = kubeapi.CreateJob(apiserver.Clientset, &newjob, ns)

	return jobName, err

}
