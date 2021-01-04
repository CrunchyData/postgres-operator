package cmd

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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

// flags used by more than 1 command
var DeleteData bool

// KeepData, If set to "true", indicates that cluster data should be stored
// even after a cluster is deleted. This is DEPRECATED
var KeepData bool

var (
	// Force indicates that the "force" action should be taken for that step. This
	// is different than NoPrompt as "Force" is for indicating that the API server
	// must try at all costs
	Force bool

	// Query indicates that the attempted request is "querying" information
	// instead of taking some action
	Query bool
)

var (
	Target  string
	Targets []string
)

var (
	OutputFormat  string
	Labelselector string
	DebugFlag     bool
	Selector      string
	DryRun        bool
	ScheduleName  string
	NodeLabel     string
)

var (
	BackupType          string
	RestoreType         string
	BackupOpts          string
	BackrestStorageType string
)

var (
	RED    func(a ...interface{}) string
	YELLOW func(a ...interface{}) string
	GREEN  func(a ...interface{}) string
)

var (
	Namespace                                    string
	PGONamespace                                 string
	APIServerURL                                 string
	PGO_CA_CERT, PGO_CLIENT_CERT, PGO_CLIENT_KEY string
	PGO_DISABLE_TLS                              bool
	EXCLUDE_OS_TRUST                             bool
)
