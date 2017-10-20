package upgradeservice

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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
)

// ShowUpgrade ...
func ShowUpgrade(namespace string, name string) msgs.ShowUpgradeResponse {
	response := msgs.ShowUpgradeResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if name == "all" {
		//get a list of all upgrades
		err := apiserver.RestClient.Get().
			Resource(crv1.PgupgradeResourcePlural).
			Namespace(namespace).
			Do().Into(&response.UpgradeList)
		if err != nil {
			log.Error("error getting list of upgrades" + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debug("upgrades found len is %d\n", len(response.UpgradeList.Items))
	} else {
		upgrade := crv1.Pgupgrade{}
		err := apiserver.RestClient.Get().
			Resource(crv1.PgupgradeResourcePlural).
			Namespace(namespace).
			Name(name).
			Do().Into(&upgrade)
		if err != nil {
			log.Error("error getting upgrade" + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.UpgradeList.Items = make([]crv1.Pgupgrade, 1)
		response.UpgradeList.Items[0] = upgrade
	}

	return response

}
