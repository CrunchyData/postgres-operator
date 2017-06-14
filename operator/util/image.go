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
	"context"
	log "github.com/Sirupsen/logrus"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

//from docker, get a map of postgres images plus the full version numbers
func GetFullVersion(imageTag string) string {

	var fullVersion = "unknown"

	log.Debug(time.Now())
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Error("error getting full version" + err.Error())
		return fullVersion
	}

	myfilters := filters.NewArgs()
	myfilters.Add("label", "Vendor=Crunchy Data Solutions")

	options := types.ImageListOptions{}
	options.Filters = myfilters

	var images []types.ImageSummary
	images, err = cli.ImageList(context.Background(), options)
	if err != nil {
		log.Error("error getting full version" + err.Error())
		return fullVersion
	}
	for _, image := range images {
		for _, name := range image.RepoTags {
			if strings.Contains(name, "crunchy-postgres") && strings.Contains(name, imageTag) {
				//fmt.Printf("%s \n", name)
				//fmt.Printf("PostgresFullVersion %s PostgresVersion %s \n", image.Labels["PostgresFullVersion"], image.Labels["PostgresVersion"])
				fullVersion = image.Labels["PostgresFullVersion"]
			}
		}
	}
	log.Debug(time.Now())

	return fullVersion
}
