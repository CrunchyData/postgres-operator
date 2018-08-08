package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/Sirupsen/logrus"
)

type ContainerResource struct {
	Name           string  `json:"Name"`
	RequestsMemory string  `json:"RequestsMemory"`
	RequestsCPU    float64 `json:"RequestsCPU"`
	LimitsMemory   string  `json:"LimitsMemory"`
	LimitsCPU      float64 `json:"LimitsCPU"`
}

type Cluster struct {
	CCPImagePrefix  string `json:"CCPImagePrefix"`
	CCPImageTag     string `json:"CCPImageTag"`
	Port            int    `json:"Port"`
	User            string `json:"User"`
	Database        string `json:"Database"`
	PasswordAgeDays int    `json:"PasswordAgeDays"`
	PasswordLength  int    `json:"PasswordLength"`
	Strategy        int    `json:"Strategy"`
	Replicas        int    `json:"Replicas"`
}

type Storage struct {
	Name        string `json:"Name"`
	AccessMode  string `json:"AccessMode"`
	Size        string `json:"Size"`
	StorageType string `json:"StorageType"`
}
type Pgo struct {
	Audit         bool   `json:"Audit"`
	Metrics       bool   `json:"Metrics"`
	LSPVCTemplate string `json:"LSPVCTemplate"`
	LoadTemplate  string `json:"LoadTemplate"`
	COImagePrefix string `json:"COImagePrefix"`
	COImageTag    string `json:"COImageTag"`
}

type PgoConfig struct {
	ClusterDef               Cluster
	PrimaryStorage           string              `json:"PrimaryStorage"`
	BackupStorage            string              `json:"BackupStorage"`
	ReplicaStorage           string              `json:"ReplicaStorage"`
	StorageDef               []Storage           `json:"Storage"`
	DefaultContainerResource string              `json:"DefaultContainerResource"`
	ContainerResources       []ContainerResource `json:"ContainerResources"`
	PgoDef                   Pgo                 `json:"Pgo"`
}

func main() {
	log.Info("hi")
	path := "./conf/apiserver/pgo.json"
	_, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error("error loading template path=" + path + err.Error())
		panic(err.Error())
	}

	thisconfig := PgoConfig{}
	thisconfig.PrimaryStorage = "storage1"
	thisconfig.StorageDef = make([]Storage, 1)
	thisconfig.StorageDef[0].Name = "storage1"
	thisconfig.StorageDef[0].AccessMode = "ReadWriteMany"
	thisconfig.DefaultContainerResource = "res1"
	thisconfig.ContainerResources = make([]ContainerResource, 1)
	thisconfig.ContainerResources[0].Name = "res1"
	thisconfig.ClusterDef = Cluster{}
	thisconfig.ClusterDef.Port = 5432
	thisconfig.PgoDef.Audit = false
	thisconfig.PgoDef.Metrics = false
	thisconfig.PgoDef.LSPVCTemplate = "this lspvc template"
	thisconfig.PgoDef.LoadTemplate = "loadtempaltepath"
	thisconfig.PgoDef.COImagePrefix = "crunchydata"
	thisconfig.PgoDef.COImageTag = "centos7-3.0"

	b, err := json.MarshalIndent(thisconfig, "", " ")
	if err != nil {
		log.Error(err)
	} else {
		os.Stdout.Write(b)
	}

}
