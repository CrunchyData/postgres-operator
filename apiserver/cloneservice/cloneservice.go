package cloneservice

/*
Copyright 2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	"net/http"
)

// CloneResults ...
type CloneResults struct {
	Results []string
}

// CreateCloneRequest ...
type CreateCloneRequest struct {
	Name string
}

// CreateCloneHandler ...
func CreateCloneHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("cloneservice.CreateCloneHandler called")
	var request CreateCloneRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("cloneservice.CreateCloneHandler got request " + request.Name)
}
