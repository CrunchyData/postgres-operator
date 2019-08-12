package rmdata

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Request struct {
	RESTConfig   *rest.Config
	RESTClient   *rest.RESTClient
	Clientset    *kubernetes.Clientset
	RemoveData   bool
	RemoveBackup bool
	IsBackup     bool
	IsReplica    bool
	ClusterName  string
	Namespace    string
}

func (x Request) String() string {
	msg := fmt.Sprintf("Request: Cluster [%s] Namespace [%s] RemoveData [%t] RemoveBackup [%t] IsReplica [%t] IsBackup [%t]", x.ClusterName, x.Namespace, x.RemoveData, x.RemoveBackup, x.IsReplica, x.IsBackup)
	return msg
}
