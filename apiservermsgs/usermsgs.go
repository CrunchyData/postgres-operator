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

// UserRequest ...
type UserRequest struct {
	Args                  []string
	Selector              string
	Namespace             string
	PasswordAgeDays       int
	ChangePasswordForUser string
	Password              string
	DeleteUser            string
	ValidDays             string
	UserDBAccess          string
	AddUser               string
	Expired               string
	UpdatePasswords       bool
	ManagedUser           bool
	ClientVersion         string
	PasswordLength        int
}

// DeleteUserResponse ...
type DeleteUserResponse struct {
	Results []string
	Status
}

// UserResponse ...
type UserResponse struct {
	Results []string
	Status
}

// CreateUserRequest ...
type CreateUserRequest struct {
	Name            string
	Namespace       string
	Selector        string
	Password        string
	ManagedUser     bool
	UserDBAccess    string
	PasswordAgeDays int
	ClientVersion   string
	PasswordLength  int
}

// CreateUserResponse ...
type CreateUserResponse struct {
	Results []string
	Status
}

// ShowUserSecret
type ShowUserSecret struct {
	Name     string
	Username string
	Password string
}

// ShowUsersDetail ...
type ShowUserDetail struct {
	Cluster     crv1.Pgcluster
	Secrets     []ShowUserSecret
	ExpiredMsgs []string
}

// ShowUsersResponse ...
type ShowUserResponse struct {
	Results []ShowUserDetail
	Status
}
