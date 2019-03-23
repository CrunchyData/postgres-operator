package benchmarkservice

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
	"encoding/json"
	"net/http"

	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

// ShowBenchmarkHandler ...
func ShowBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("benchmarkservice.ShowBenchmarkHandler called")

	var request msgs.ShowBenchmarkRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.SHOW_SCHEDULE_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	ns, err := apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp := msgs.CreateBenchmarkResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
			Results: make([]string, 0),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	request.Namespace = ns

	resp := ShowBenchmark(&request)
	json.NewEncoder(w).Encode(resp)
}

// DeleteBenchmarkHandler ...
func DeleteBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("benchmarkservice.DeleteBenchmarkHandler called")

	var request msgs.DeleteBenchmarkRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.DELETE_SCHEDULE_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	ns, err := apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp := msgs.CreateBenchmarkResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
			Results: make([]string, 0),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	request.Namespace = ns

	resp := DeleteBenchmark(&request)
	json.NewEncoder(w).Encode(resp)
}

// CreateBackupHandler ...
func CreateBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var username, ns string

	log.Debug("benchmarkservice.CreateBenchmarkHandler called")

	var request msgs.CreateBenchmarkRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err = apiserver.Authn(apiserver.CREATE_SCHEDULE_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp := msgs.CreateBenchmarkResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
			Results: make([]string, 0),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := CreateBenchmark(&request, ns)
	json.NewEncoder(w).Encode(resp)
}
