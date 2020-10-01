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

const PGO_VERSION = "4.5.0"

// Ok status
const Ok = "ok"

// Error code string
const Error = "error"

// UpgradeError is the error used for when a command is tried against a cluster that has not
// been upgraded to the current Operator version
const UpgradeError = " has not yet been upgraded. Please upgrade the cluster before running this Postgres Operator command."

// Status ...
// swagger:model Status
type Status struct {
	// status code
	Code string
	// status message
	Msg string
}

// Syntactic sugar for consistency and readibility
func (s *Status) SetError(msg string) {
	s.Code = Error
	s.Msg = msg
}

// BasicAuthCredentials ...
// swagger:model BasicAuthCredentials
type BasicAuthCredentials struct {
	Username     string
	Password     string
	APIServerURL string
}

func (b BasicAuthCredentials) HasUsernameAndPassword() bool {
	return len(b.Username) > 0 && len(b.Password) > 0
}
