package cmd

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
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"os"
)

func showConfig(args []string) {

	log.Debugf("showConfig called %v", args)

	response, err := api.ShowConfig(httpclient, &SessionCredentials)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(2)
	}

	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}

	if OutputFormat == "json" {
		b, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println(string(b))
		return
	}

	pgo := response.Result

	fmt.Printf("%s%s\n", "BasicAuth:  ", pgo.BasicAuth)
	fmt.Printf("%s\n", "Cluster:")
	fmt.Printf("%s%s\n", "  PrimaryNodeLabel:  ", pgo.Cluster.PrimaryNodeLabel)
	fmt.Printf("%s%s\n", "  ReplicaNodeLabel:  ", pgo.Cluster.ReplicaNodeLabel)
	fmt.Printf("%s%s\n", "  CCPImagePrefix:  ", pgo.Cluster.CCPImagePrefix)
	fmt.Printf("%s%s\n", "  CCPImageTag:  ", pgo.Cluster.CCPImageTag)
	fmt.Printf("%s%s\n", "  ServiceType:  ", pgo.Cluster.ServiceType)
	fmt.Printf("%s%t\n", "  Metrics:  ", pgo.Cluster.Metrics)
	fmt.Printf("%s%t\n", "  Backrest:  ", pgo.Cluster.Backrest)
	fmt.Printf("%s%t\n", "  Autofail:  ", pgo.Cluster.Autofail)
	fmt.Printf("%s%t\n", "  AutofailReplaceReplica:  ", pgo.Cluster.AutofailReplaceReplica)
	fmt.Printf("%s%t\n", "  Badger:  ", pgo.Cluster.Badger)
	fmt.Printf("%s%s\n", "  Policies:  ", pgo.Cluster.Policies)
	fmt.Printf("%s%s\n", "  Port:  ", pgo.Cluster.Port)
	fmt.Printf("%s%s\n", "  ArchiveTimeout:  ", pgo.Cluster.ArchiveTimeout)
	fmt.Printf("%s%s\n", "  ArchiveMode:  ", pgo.Cluster.ArchiveMode)
	fmt.Printf("%s%s\n", "  User:  ", pgo.Cluster.User)
	fmt.Printf("%s%s\n", "  Database:  ", pgo.Cluster.Database)
	fmt.Printf("%s%s\n", "  PasswordAgeDays:  ", pgo.Cluster.PasswordAgeDays)
	fmt.Printf("%s%s\n", "  PasswordLength:  ", pgo.Cluster.PasswordLength)
	fmt.Printf("%s%s\n", "  Strategy:  ", pgo.Cluster.Strategy)
	fmt.Printf("%s%s\n", "  Replicas:  ", pgo.Cluster.Replicas)

	fmt.Printf("%s%s\n", "PrimaryStorage:  ", pgo.PrimaryStorage)
	fmt.Printf("%s%s\n", "ArchiveStorage:  ", pgo.ArchiveStorage)
	fmt.Printf("%s%s\n", "BackupStorage:  ", pgo.BackupStorage)
	fmt.Printf("%s%s\n", "ReplicaStorage:  ", pgo.ReplicaStorage)
	fmt.Printf("%s%s\n", "BackrestStorage:  ", pgo.BackrestStorage)
	fmt.Printf("%s\n", "Storage:")
	for k, v := range pgo.Storage {
		fmt.Printf("  %s:\n", k)
		fmt.Printf("%s%s\n", "    AccessMode:  ", v.AccessMode)
		fmt.Printf("%s%s\n", "    Size:  ", v.Size)
		if v.MatchLabels != "" {
			fmt.Printf("%s%s\n", "    MatchLabels:  ", v.MatchLabels)
		}
		fmt.Printf("%s%s\n", "    StorageType:  ", v.StorageType)
		if v.StorageClass != "" {
			fmt.Printf("%s%s\n", "    StorageClass:  ", v.StorageClass)
		}
		if v.Fsgroup != "" {
			fmt.Printf("%s%s\n", "    Fsgroup:  ", v.Fsgroup)
		}
		if v.SupplementalGroups != "" {
			fmt.Printf("%s%s\n", "    SupplementalGroups:  ", v.SupplementalGroups)
		}
	}

	fmt.Printf("%s%s\n", "DefaultContainerResources:", pgo.DefaultContainerResources)
	fmt.Printf("%s\n", "ContainerResources:")
	for k, v := range pgo.ContainerResources {
		fmt.Printf("  %s:\n", k)
		fmt.Printf("%s%s\n", "    RequestsMemory:  ", v.RequestsMemory)
		fmt.Printf("%s%s\n", "    RequestsCPU:  ", v.RequestsCPU)
		fmt.Printf("%s%s\n", "    LimitsMemory:  ", v.LimitsMemory)
		fmt.Printf("%s%s\n", "    LimitsCPU:  ", v.LimitsCPU)
	}

	fmt.Printf("%s\n", "Pgo:")
	fmt.Printf("%s%s\n", "  AutofailSleepSeconds:  ", pgo.Pgo.AutofailSleepSeconds)
	fmt.Printf("%s%t\n", "  Audit:  ", pgo.Pgo.Audit)
	fmt.Printf("%s%s\n", "  LSPVCTemplate:  ", pgo.Pgo.LSPVCTemplate)
	fmt.Printf("%s%s\n", "  LoadTemplate:  ", pgo.Pgo.LoadTemplate)
	fmt.Printf("%s%s\n", "  COImagePrefix:  ", pgo.Pgo.COImagePrefix)
	fmt.Printf("%s%s\n", "  COImageTag:  ", pgo.Pgo.COImageTag)

	fmt.Println("")

}
