// Package cmd provides the command line functions of the crunchy CLI
package cmd

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

import (
	"fmt"
	"os"
	"strings"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo-scheduler/scheduler"
	log "github.com/sirupsen/logrus"

	"github.com/crunchydata/postgres-operator/pgo/api"
)

type schedule struct {
	schedule            string
	scheduleType        string
	pvcName             string
	backrestType        string
	backrestStorageType string
	clusterName         string
	selector            string
	policy              string
	database            string
}

func createSchedule(args []string, ns string) {
	log.Debugf("createSchedule called %v", args)

	var clusterName string
	if len(args) > 0 {
		clusterName = args[0]
	}

	s := schedule{
		clusterName:         clusterName,
		backrestType:        PGBackRestType,
		backrestStorageType: BackrestStorageType,
		pvcName:             PVCName,
		schedule:            Schedule,
		selector:            Selector,
		scheduleType:        ScheduleType,
		policy:              SchedulePolicy,
		database:            ScheduleDatabase,
	}

	err := s.validateSchedule()
	if err != nil {
		fmt.Println(err)
		return
	}

	r := &msgs.CreateScheduleRequest{
		ClusterName:         clusterName,
		PGBackRestType:      PGBackRestType,
		BackrestStorageType: BackrestStorageType,
		PVCName:             PVCName,
		ScheduleOptions:     ScheduleOptions,
		Schedule:            Schedule,
		Selector:            Selector,
		ScheduleType:        strings.ToLower(ScheduleType),
		PolicyName:          SchedulePolicy,
		Database:            ScheduleDatabase,
		Secret:              ScheduleSecret,
		Namespace:           ns,
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

func deleteSchedule(args []string, ns string) {
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
		Namespace:    ns,
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

func showSchedule(args []string, ns string) {
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
		Namespace:    ns,
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

func (s *schedule) validateSchedule() error {
	if err := scheduler.ValidateSchedule(s.schedule); err != nil {
		return err
	}

	if err := scheduler.ValidateScheduleType(s.scheduleType); err != nil {
		return err
	}

	if err := scheduler.ValidateBaseBackupSchedule(s.scheduleType, s.pvcName); err != nil {
		return err
	}

	if err := scheduler.ValidateBackRestSchedule(s.scheduleType, s.clusterName, s.selector, s.backrestType,
		s.backrestStorageType); err != nil {
		return err
	}

	if err := scheduler.ValidatePolicySchedule(s.scheduleType, s.policy, s.database); err != nil {
		return err
	}

	return nil
}
