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

package util

import (
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"text/template"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
)

func CreateSecContext(FS_GROUP string, SUPP string) string {

	var sc bytes.Buffer
	var fsgroup = false
	var supp = false

	if FS_GROUP != "" {
		fsgroup = true
	}
	if SUPP != "" {
		supp = true
	}
	if fsgroup || supp {
		sc.WriteString("\"securityContext\": {\n")
	}
	if fsgroup {
		sc.WriteString("\t \"fsGroup\": " + FS_GROUP)
		if fsgroup && supp {
			sc.WriteString(",")
		}
		sc.WriteString("\n")
	}

	if supp {
		sc.WriteString("\t \"supplementalGroups\": [" + SUPP + "]\n")
	}

	//closing of securityContext
	if fsgroup || supp {
		sc.WriteString("},")
	}

	return sc.String()
}

func LoadTemplate(path string) *template.Template {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	return template.Must(template.New(path).Parse(string(buf)))

}

type ThingSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func Patch(tprclient *rest.RESTClient, path string, value string, resource string, name string, namespace string) error {

	things := make([]ThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = path
	things[0].Value = value

	patchBytes, err4 := json.Marshal(things)
	if err4 != nil {
		log.Error("error in converting patch " + err4.Error())
	}
	log.Debug(string(patchBytes))

	_, err6 := tprclient.Patch(api.JSONPatchType).
		Namespace(api.NamespaceDefault).
		Resource(resource).
		Name(name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}
