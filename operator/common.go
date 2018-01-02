package operator

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

import (
	log "github.com/Sirupsen/logrus"
	"os"
)

var COImagePrefix string
var COImageTag string
var CCPImagePrefix string

func Initialize() {
	CCPImagePrefix = os.Getenv("CCP_IMAGE_PREFIX")
	if CCPImagePrefix == "" {
		log.Debug("CCP_IMAGE_PREFIX not set, using default")
		CCPImagePrefix = "crunchydata"
	} else {
		log.Debug("CCP_IMAGE_PREFIX set, using " + CCPImagePrefix)
	}
	COImagePrefix = os.Getenv("CO_IMAGE_PREFIX")
	if COImagePrefix == "" {
		log.Debug("CO_IMAGE_PREFIX not set, using default")
		COImagePrefix = "crunchydata"
	} else {
		log.Debug("CO_IMAGE_PREFIX set, using " + COImagePrefix)
	}
	COImageTag = os.Getenv("CO_IMAGE_TAG")
	if COImageTag == "" {
		log.Error("CO_IMAGE_TAG not set, required ")
		panic("CO_IMAGE_TAG env var not set")
	}

}
