package pgadmin

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

// ServerEntry models parts of the pgadmin server table
type ServerEntry struct {
	Name          string // Typically set to the cluster name
	Host          string
	Port          int
	MaintenanceDB string
	SSLMode       string
	Comment       string
	Password      string

	// Not yet used, latest params of 4.18
	//
	// servergroup_id int     // associated at query time
	// user_id        int     // set based on login username
	// username       string  // set based on login username
	//
	// role_text      string
	// discovery_id   string
	// hostaddr       string
	// db_res         string
	// passfile       string
	// sslcert        string
	// sslkey         string
	// sslrootcert    string
	// sslcrl         string
	// sslcompression bool
	// use_ssh_tunnel bool
	// tunnel_host string
	// tunnel_port string
	// tunnel_username string
	// tunnel_authentication bool
	// tunnel_identity_file string
	// connect_timeout int
	// tunnel_password string
}
