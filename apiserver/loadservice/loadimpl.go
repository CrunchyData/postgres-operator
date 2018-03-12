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
	"errors"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/policyservice"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	operutil "github.com/crunchydata/postgres-operator/util"
	"github.com/spf13/viper"
	"io/ioutil"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"os"
	"strings"
	"text/template"
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
var LoadConfig string

// LoadConfigTemplate ....
var LoadConfigTemplate loadJobTemplateFields

// LoadTemplatePath ...
var LoadTemplatePath string

// JobTemplate ...
var JobTemplate *template.Template

func init() {

	log.Debug("loadimpl init called")

	LoadTemplatePath = viper.GetString("Pgo.LoadTemplate")
	if LoadTemplatePath == "" {
		log.Error("Pgo.LoadTemplate not defined in pgo config 1.")
		os.Exit(2)
	}

	//get the job template
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(LoadTemplatePath)
	if err != nil {
		log.Error("error loading pgo-load job template..." + err.Error())
		os.Exit(2)
	}
	JobTemplate = template.Must(template.New("pgo-load job template").Parse(string(buf)))
}

// Load ...
// pgo load  --policies=jsonload --selector=name=mycluster --load-config=./sample-load-config.json
func Load(request *msgs.LoadRequest) msgs.LoadResponse {

	var err error
	resp := msgs.LoadResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	/**
	LoadTemplatePath = viper.GetString("Pgo.LoadTemplate")
	if LoadTemplatePath == "" {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = "Pgo.LoadTemplate not defined in pgo config 2."
		return resp
	}
	*/

	LoadConfigTemplate = loadJobTemplateFields{}

	viper.SetConfigType("yaml")

	viper.ReadConfig(bytes.NewBufferString(request.LoadConfig))

	err = validateConfig()
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	LoadConfigTemplate.COImagePrefix = viper.GetString("COImagePrefix")
	LoadConfigTemplate.COImageTag = viper.GetString("COImageTag")
	LoadConfigTemplate.DbDatabase = viper.GetString("DbDatabase")
	LoadConfigTemplate.DbUser = viper.GetString("DbUser")
	LoadConfigTemplate.DbPort = viper.GetString("DbPort")
	LoadConfigTemplate.TableToLoad = viper.GetString("TableToLoad")
	LoadConfigTemplate.FilePath = viper.GetString("FilePath")
	LoadConfigTemplate.FileType = viper.GetString("FileType")
	LoadConfigTemplate.PVCName = viper.GetString("PVCName")
	LoadConfigTemplate.SecurityContext = viper.GetString("SecurityContext")

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
		err = apiserver.RESTClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(apiserver.Namespace).
			Param("labelSelector", myselector.String()).
			//LabelsSelectorParam(myselector).
			Do().
			Into(&clusterList)
		if err != nil {
			log.Error("error getting cluster list" + err.Error())
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

	log.Debug("policies to apply before loading are %v\n", request.Policies)
	policies := strings.Split(request.Policies, ",")

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
			}
		}

		//create the load job for this cluster
		log.Debug("created load for " + arg)
		err = createJob(arg, LoadConfigTemplate.FileType, LoadConfigTemplate.FilePath)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

	}

	return resp

}

func createJob(clusterName, filetype, filepath string) error {
	var err error
	randStr := operutil.GenerateRandString(3)
	LoadConfigTemplate.Name = "pgo-load-" + clusterName + "-" + randStr
	LoadConfigTemplate.DbHost = clusterName
	LoadConfigTemplate.FilePath = filepath
	LoadConfigTemplate.FileType = filetype
	LoadConfigTemplate.DbPass, err = operutil.GetSecretPassword(apiserver.Clientset, clusterName, crv1.RootSecretSuffix, apiserver.Namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	var doc2 bytes.Buffer
	err = JobTemplate.Execute(&doc2, LoadConfigTemplate)
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

	resultJob, err := apiserver.Clientset.Batch().Jobs(apiserver.Namespace).Create(&newjob)
	if err != nil {
		log.Error("error creating Job " + err.Error())
		return err
	}
	log.Debug("created load Job " + resultJob.Name)
	return err

}

func validateConfig() error {
	var err error
	if viper.GetString("COImagePrefix") == "" {
		return errors.New("COImagePrefix is not supplied")
	}
	if viper.GetString("COImageTag") == "" {
		return errors.New("COImageTag is not supplied")
	}
	if viper.GetString("DbDatabase") == "" {
		return errors.New("DbDatabase is not supplied")
	}
	if viper.GetString("DbUser") == "" {
		return errors.New("DbUser is not supplied")
	}
	if viper.GetString("DbPort") == "" {
		return errors.New("DbPort is not supplied")
	}
	if viper.GetString("TableToLoad") == "" {
		return errors.New("TableToLoad is not supplied")
	}
	if viper.GetString("FilePath") == "" {
		return errors.New("FilePath is not supplied")
	}
	if viper.GetString("PVCName") == "" {
		return errors.New("PVCName is not supplied")
	}
	return err
}
