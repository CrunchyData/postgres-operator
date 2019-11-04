package apiservermsgs

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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
)

// UpdateUserRequest ...
// swagger:model
type UpdateUserRequest struct {
	Clusters              []string
	Selector              string
	AllFlag               bool
	ExpireUser            bool
	Namespace             string
	PasswordAgeDays       int
	PasswordAgeDaysUpdate bool
	Username              string
	Password              string
	DeleteUser            string
	ValidDays             string
	UserDBAccess          string
	AddUser               string
	Expired               string
	ManagedUser           bool
	ClientVersion         string
	PasswordLength        int
}

// DeleteUserRequest ...
// swagger:model
type DeleteUserRequest struct {
	Selector      string
	Clusters      []string
	AllFlag       bool
	Username      string
	ClientVersion string
	Namespace     string
}

// DeleteUserResponse ...
// swagger:model
type DeleteUserResponse struct {
	Results []string
	Status
}

// UpdateUserResponse ...
// swagger:model
type UpdateUserResponse struct {
	Results []string
	Status
}

// CreateUserRequest ...
// swagger:model
type CreateUserRequest struct {
	Clusters    []string
	Username    string
	Namespace   string
	Selector    string
	AllFlag     bool
	Password    string
	ManagedUser bool
	//UserDBAccess    string
	PasswordAgeDays int
	ClientVersion   string
	PasswordLength  int
}

// CreateUserResponse ...
// swagger:model
type CreateUserResponse struct {
	Results []string
	Status
}

// ShowUserSecret
// swagger:model
type ShowUserSecret struct {
	Name     string
	Username string
	Password string
}

// ShowUserRequest ...
// swagger:model
type ShowUserRequest struct {
	Clusters      []string
	AllFlag       bool
	ClientVersion string
	Selector      string
	Namespace     string
	Expired       string
}

// ShowUsersDetail ...
// swagger:model
type ShowUserDetail struct {
	Cluster       crv1.Pgcluster
	Secrets       []ShowUserSecret
	ExpiredOutput bool
	ExpiredDays   int
	ExpiredMsgs   []string
}

// ShowUsersResponse ...
// swagger:model
type ShowUserResponse struct {
	Results []ShowUserDetail
	Status
}
