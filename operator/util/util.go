/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

package util

import (
	"bytes"
)

func CreateSecContext(FS_GROUP string, SUPP string) string {

	var sc bytes.Buffer
	var fsgroup = false
	var supp = false

	if FS_GROUP != "" {
		fsgroup = true
	}
	if SUPP != "" {
		supp = true
	}
	if fsgroup || supp {
		sc.WriteString("\"securityContext\": {\n")
	}
	if fsgroup {
		sc.WriteString("\t \"fsGroup\": " + FS_GROUP)
		if fsgroup && supp {
			sc.WriteString(",")
		}
		sc.WriteString("\n")
	}

	if supp {
		sc.WriteString("\t \"supplementalGroups\": [" + SUPP + "]\n")
	}

	//closing of securityContext
	if fsgroup || supp {
		sc.WriteString("},")
	}

	return sc.String()
}
