// Package cmd provides the command line functions of the crunchy CLI
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

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
)

func createSchedule(args []string) {
	log.Debugf("createSchedule called %v", args)

	if len(args) == 0 && Selector == "" {
		fmt.Println("Error: The --selector flag or a cluster name is required to create a schedule.")
		return
	}

	if ScheduleType == "pgbasebackup" {
		if PVCName == "" {
			fmt.Println("Error: The --pvc-name flag must be set when scheduling pgBaseBackup backups.")
			return
		}
	}

	if ScheduleType == "pgbackrest" {
		if PGBackRestType == "" {
			fmt.Println("Error: The --pgbackrest-backup-type flag must be set when scheduling pgBackRest backups.")
			return
		}
	}

	var clusterName string
	if Selector != "" {
		clusterName = ""
	} else if len(args) > 0 {
		clusterName = args[0]
	}

	r := &msgs.CreateScheduleRequest{
		ClusterName:     clusterName,
		PGBackRestType:  PGBackRestType,
		PVCName:         PVCName,
		CCPImageTag:     CCPImageTag,
		ScheduleOptions: ScheduleOptions,
		Schedule:        Schedule,
		Selector:        Selector,
		ScheduleType:    ScheduleType,
	}

	response, err := api.CreateSchedule(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.Results) == 0 {
		fmt.Println("No clusters found.")
		return
	}

}

func deleteSchedule(args []string) {
	log.Debugf("deleteSchedule called %v", args)

	if len(args) == 0 && Selector == "" && ScheduleName == "" {
		fmt.Println("Error: cluster name, schedule name or selector is required to delete a schedule.")
		return
	}

	var clusterName string
	if len(args) > 0 {
		clusterName = args[0]
	}

	r := &msgs.DeleteScheduleRequest{
		ClusterName:  clusterName,
		ScheduleName: ScheduleName,
		Selector:     Selector,
	}

	response, err := api.DeleteSchedule(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.Results) == 0 {
		fmt.Println("No schedules found.")
		return
	}

}

func showSchedule(args []string) {
	log.Debugf("showSchedule called %v", args)

	if len(args) == 0 && Selector == "" && ScheduleName == "" {
		fmt.Println("Error: cluster name, schedule name or selector is required to show a schedule.")
		return
	}

	var clusterName string
	if Selector != "" {
		clusterName = ""
	} else if len(args) > 0 {
		clusterName = args[0]
	}

	r := &msgs.ShowScheduleRequest{
		ClusterName:  clusterName,
		ScheduleName: ScheduleName,
		Selector:     Selector,
	}

	response, err := api.ShowSchedule(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if response.Status.Code == msgs.Ok {
		for k := range response.Results {
			fmt.Println(response.Results[k])
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if len(response.Results) == 0 {
		fmt.Println("No schedules found.")
		return
	}
}
