package apiservermsgs

/*
Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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

import ()

type UpdateClusterLoginState int

// set the different values around whether or not to disable/enable a user's
// ability to login
const (
	UpdateUserLoginDoNothing UpdateClusterLoginState = iota
	UpdateUserLoginEnable
	UpdateUserLoginDisable
)

// CreateUserRequest contains the parameters that are passed in when an Operator
// user requests to create a new PostgreSQL user
// swagger:model
type CreateUserRequest struct {
	AllFlag         bool
	Clusters        []string
	ClientVersion   string
	ManagedUser     bool
	Namespace       string
	Password        string
	PasswordAgeDays int
	PasswordLength  int
	Selector        string
	Username        string
}

// CreateUserResponse is the response to a create user request
// swagger:model
type CreateUserResponse struct {
	Results []UserResponseDetail
	Status
}

// DeleteUserRequest
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

// ShowUserRequest finds information about users in various PostgreSQL clusters
// swagger:model
type ShowUserRequest struct {
	AllFlag            bool
	Clusters           []string
	ClientVersion      string
	Expired            int
	Namespace          string
	Selector           string
	ShowSystemAccounts bool
}

// ShowUsersResponse ...
// swagger:model
type ShowUserResponse struct {
	Results []UserResponseDetail
	Status
}

// UpdateUserRequest is the API to allow an Operator user to update information
// about a PostgreSQL user
// swagger:model
type UpdateUserRequest struct {
	AllFlag             bool
	ClientVersion       string
	Clusters            []string
	Expired             int
	ExpireUser          bool
	LoginState          UpdateClusterLoginState
	ManagedUser         bool
	Namespace           string
	Password            string
	PasswordAgeDays     int
	PasswordLength      int
	PasswordValidAlways bool
	RotatePassword      bool
	Selector            string
	Username            string
}

// UpdateUserResponse contains the response after an update user request
// swagger:model
type UpdateUserResponse struct {
	Results []UserResponseDetail
	Status
}

// UserResponseDetail returns specific information about the user that
// was updated, including password, expiration time, etc.
// swagger:model
type UserResponseDetail struct {
	ClusterName  string
	Error        bool
	ErrorMessage string
	Password     string
	Username     string
	ValidUntil   string
}
