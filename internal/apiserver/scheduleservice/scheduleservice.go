package scheduleservice

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"net/http"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

type PgScheduleSpec struct {
	Version    string `json:"version"`
	Name       string `json:"name"`
	Cluster    string `json:"cluster"`
	Created    string `json:"created"`
	Schedule   string `json:"schedule"`
	Namespace  string `json:"namespace"`
	Type       string `json:"type"`
	PGBackRest `json:"pgbackrest,omitempty"`
	Policy     `json:"policy,omitempty"`
}

type Policy struct {
	Name        string `json:"name,omitempty"`
	Database    string `json:"database,omitempty"`
	Secret      string `json:"secret,omitempty"`
	ImagePrefix string `json:"imagePrefix,omitempty"`
	ImageTag    string `json:"imageTag,omitempty"`
}

type PGBackRest struct {
	Deployment  string `json:"deployment,omitempty"`
	Label       string `json:"label,omitempty"`
	Container   string `json:"container,omitempty"`
	Type        string `json:"type,omitempty"`
	StorageType string `json:"storageType,omitempty"`
	Options     string `json:"options,omitempty"`
}

// CreateScheduleHandler ...
func CreateScheduleHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /schedule scheduleservice schedule
	/*```
	  Schedule creates a cron-like scheduled task
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create Schedule Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreateScheduleRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreateScheduleResponse"
	var err error
	var username, ns string

	log.Debug("scheduleservice.CreateScheduleHandler called")

	var request msgs.CreateScheduleRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err = apiserver.Authn(apiserver.CREATE_SCHEDULE_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp := msgs.CreateScheduleResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
			Results: make([]string, 0),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := CreateSchedule(&request, ns)
	json.NewEncoder(w).Encode(resp)
}

func DeleteScheduleHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /scheduledelete scheduleservice scheduledelete
	/*```
	  Delete a cron-like schedule
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Delete Schedule Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DeleteScheduleRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DeleteScheduleResponse"
	var err error
	var username, ns string

	log.Debug("scheduleservice.DeleteScheduleHandler called")

	var request msgs.DeleteScheduleRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err = apiserver.Authn(apiserver.DELETE_SCHEDULE_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp := &msgs.DeleteScheduleResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
			Results: make([]string, 0),
		}
		json.NewEncoder(w).Encode(resp)
		return

	}

	resp := DeleteSchedule(&request, ns)
	json.NewEncoder(w).Encode(resp)
}

func ShowScheduleHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /scheduleshow scheduleservice scheduleshow
	/*```
	  Show cron-like schedules
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Show Schedule Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/ShowScheduleRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowScheduleResponse"
	var err error
	var username, ns string

	log.Debug("scheduleservice.ShowScheduleHandler called")

	var request msgs.ShowScheduleRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err = apiserver.Authn(apiserver.SHOW_SCHEDULE_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp := &msgs.ShowScheduleResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
			Results: make([]string, 0),
		}

		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := ShowSchedule(&request, ns)
	json.NewEncoder(w).Encode(resp)
}
