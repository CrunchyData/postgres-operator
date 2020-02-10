package rmdata

/*
Copyright 2019 - 2020 Crunchy Data
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Request struct {
	RESTConfig       *rest.Config
	RESTClient       *rest.RESTClient
	Clientset        *kubernetes.Clientset
	RemoveData       bool
	RemoveBackup     bool
	IsBackup         bool
	IsReplica        bool
	ClusterName      string
	ClusterPGHAScope string
	ReplicaName      string
	Namespace        string
}

func (x Request) String() string {
	msg := fmt.Sprintf("Request: Cluster [%s] ClusterPGHAScope [%s] Namespace [%s] ReplicaName [%] RemoveData [%t] RemoveBackup [%t] IsReplica [%t] IsBackup [%t]", x.ClusterName, x.ClusterPGHAScope, x.Namespace, x.ReplicaName, x.RemoveData, x.RemoveBackup, x.IsReplica, x.IsBackup)
	return msg
}
