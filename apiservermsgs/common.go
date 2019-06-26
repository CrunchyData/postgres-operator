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

import ()

const PGO_VERSION = "4.0.1"

// Ok status
const Ok = "ok"
const Error = "error"

// Status ...
type Status struct {
	Code string
	Msg  string
}

type BasicAuthCredentials struct {
	Username     string
	Password     string
	APIServerURL string
}

func (b BasicAuthCredentials) HasUsernameAndPassword() bool {
	return len(b.Username) > 0 && len(b.Password) > 0
}
