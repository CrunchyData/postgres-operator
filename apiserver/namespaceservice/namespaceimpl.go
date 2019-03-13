package namespaceservice

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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
)

func ShowNamespace(username string) msgs.ShowNamespaceResponse {
	log.Debug("ShowNamespace called")
	response := msgs.ShowNamespaceResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	response.Results = make([]msgs.NamespaceResult, 0)
	namespaceList := util.GetNamespaces()

	for i := 0; i < len(namespaceList); i++ {
		r := msgs.NamespaceResult{
			Namespace:  namespaceList[i],
			UserAccess: apiserver.UserIsPermittedInNamespace(username, namespaceList[i]),
		}
		response.Results = append(response.Results, r)
	}

	return response
}
