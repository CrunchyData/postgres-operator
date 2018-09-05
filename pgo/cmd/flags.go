package cmd

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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

//flags used by more than 1 command
var DeleteData bool
var Query bool
var Target string

var OutputFormat string
var APIServerURL string
var Labelselector string
var DebugFlag bool
var Selector string
var DryRun bool

var BackupType string

var RED func(a ...interface{}) string
var GREEN func(a ...interface{}) string
