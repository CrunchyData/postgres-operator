/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package pgadmin

const (
	// tmp volume to hold the nss_wrapper, process and socket files
	// both the '/tmp' mount path and '/etc/httpd/run' mount path
	// mount the 'tmp' volume
	tmpVolume    = "tmp"
	tmpMountPath = "/tmp"
	runMountPath = "/etc/httpd/run"

	// log volume and path where the pgadmin4.log is located
	logVolume    = "pgadmin-log"
	logMountPath = "/var/log/pgadmin"

	// data volume and path to hold persistent pgAdmin data
	dataVolume    = "pgadmin-data"
	dataMountPath = "/var/lib/pgadmin"

	// TODO(tjmoore4): The login and password implementation will be updated in
	// upcoming enhancement work.

	// initial pgAdmin login email address
	loginEmail = "admin"

	// initial pgAdmin login password
	loginPassword = "admin"

	// default pgAdmin port
	defaultPort = 5050
)
