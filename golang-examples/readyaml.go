package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type ClusterStruct struct {
	CCPImagePrefix  string `yaml:"CCPImagePrefix"`
	CCPImageTag     string `yaml:"CCPImageTag"`
	Policies        string `yaml:"Policies"`
	Port            int    `yaml:"Port"`
	User            string `yaml:"User"`
	Database        string `yaml:"Database"`
	PasswordAgeDays int    `yaml:"PasswordAgeDays"`
	PasswordLength  int    `yaml:"PasswordLength"`
	Strategy        int    `yaml:"Strategy"`
	Replicas        int    `yaml:"Replicas"`
}

type StorageStruct struct {
	StorageClass       string `yaml:"StorageClass"`
	AccessMode         string `yaml:"AccessMode"`
	Size               string `yaml:"Size"`
	StorageType        string `yaml:"StorageType"`
	Fsgroup            string `yaml:"Fsgroup"`
	SupplementalGroups string `yaml:"SupplementalGroups"`
}

type ContainerResourcesStruct struct {
	RequestsMemory string  `yaml:"RequestsMemory"`
	RequestsCPU    float64 `yaml:"RequestsCPU"`
	LimitsMemory   string  `yaml:"LimitsMemory"`
	LimitsCPU      float64 `yaml:"LimitsCPU"`
}

type PgoConfig struct {
	Audit         bool   `yaml:"Audit"`
	Metrics       bool   `yaml:"Metrics"`
	LSPVCTemplate string `yaml:"LSPVCTemplate"`
	LoadTemplate  string `yaml:"LoadTemplate"`
	COImagePrefix string `yaml:"COImagePrefix"`
	COImageTag    string `yaml:"COImageTag"`
}

type Config struct {
	BasicAuth                 bool                                `yaml:"BasicAuth"`
	Cluster                   ClusterStruct                       `yaml:"Cluster"`
	Pgo                       PgoConfig                           `yaml:"Pgo"`
	ContainerResources        map[string]ContainerResourcesStruct `yaml:"ContainerResources"`
	PrimaryStorage            string                              `yaml:"PrimaryStorage"`
	BackupStorage             string                              `yaml:"BackupStorage"`
	ReplicaStorage            string                              `yaml:"ReplicaStorage"`
	Storage                   map[string]StorageStruct            `yaml:"Storage"`
	DefaultContainerResources string                              `yaml:"DefaultContainerResources"`
}

func (c *Config) validate() error {
	var err error
	_, ok := c.Storage[c.PrimaryStorage]
	if !ok {
		return errors.New("invalid PrimaryStorage setting")
	}
	_, ok = c.Storage[c.BackupStorage]
	if !ok {
		return errors.New("invalid BackupStorage setting")
	}
	_, ok = c.Storage[c.ReplicaStorage]
	if !ok {
		return errors.New("invalid ReplicaStorage setting")
	}

	if c.Pgo.LSPVCTemplate == "" {
		return errors.New("Pgo.LSPVCTemplate is required")
	}
	if c.Pgo.LoadTemplate == "" {
		return errors.New("Pgo.LoadTemplate is required")
	}
	if c.Pgo.COImagePrefix == "" {
		return errors.New("Pgo.COImagePrefix is required")
	}
	if c.Pgo.COImageTag == "" {
		return errors.New("Pgo.COImageTag is required")
	}

	if c.DefaultContainerResources != "" {
		_, ok = c.ContainerResources[c.DefaultContainerResources]
		if !ok {
			return errors.New("DefaultContainerResources setting invalid")
		}
	}

	if c.Cluster.CCPImagePrefix == "" {
		return errors.New("Cluster.CCPImagePrefix is required")
	}

	if c.Cluster.CCPImageTag == "" {
		return errors.New("Cluster.CCPImageTag is required")
	}
	return err
}

func (c *Config) getConf() *Config {

	yamlFile, err := ioutil.ReadFile("conf/apiserver/pgo.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

var Pgo Config

func main() {
	Pgo.getConf()

	fmt.Println(Pgo.Cluster.Policies)
	fmt.Println(Pgo.Cluster.CCPImageTag)
	fmt.Println(Pgo.PrimaryStorage)
	fmt.Println(Pgo.Pgo.COImageTag)
	fmt.Printf("%v is BasicAuth\n", Pgo.BasicAuth)
	fmt.Println(Pgo.ContainerResources)
	fmt.Println(Pgo.Storage)
	fmt.Printf("length of storage %d\n", len(Pgo.Storage))
	s, err := getStorageSpec("storage2")
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(s.AccessMode)
	x, err := getContainerResource("small")
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(x.RequestsMemory)

	Pgo.validate()
}

func getStorageSpec(name string) (StorageStruct, error) {
	var err error

	s, ok := Pgo.Storage[name]
	if !ok {
		return s, errors.New("invalid Storage name " + name)
	}
	return s, err

}
func getContainerResource(name string) (ContainerResourcesStruct, error) {
	var err error

	s, ok := Pgo.ContainerResources[name]
	if !ok {
		return s, errors.New("invalid ContainerResources name " + name)
	}
	return s, err

}
