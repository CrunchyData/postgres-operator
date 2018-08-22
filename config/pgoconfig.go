package config

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
	"errors"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strconv"
)

const PGO_VERSION = "3.2"

type ClusterStruct struct {
	CCPImagePrefix  string `yaml:"CCPImagePrefix"`
	CCPImageTag     string `yaml:"CCPImageTag"`
	Policies        string `yaml:"Policies"`
	Metrics         bool   `yaml:"Metrics"`
	Badger          bool   `yaml:"Badger"`
	Port            string `yaml:"Port"`
	ArchiveTimeout  string `yaml:"ArchiveTimeout"`
	ArchiveMode     string `yaml:"ArchiveMode"`
	User            string `yaml:"User"`
	Database        string `yaml:"Database"`
	PasswordAgeDays string `yaml:"PasswordAgeDays"`
	PasswordLength  string `yaml:"PasswordLength"`
	Strategy        string `yaml:"Strategy"`
	Replicas        string `yaml:"Replicas"`
	ServiceType     string `yaml:"ServiceType"`
	Backrest        bool   `yaml:"Backrest"`
	Autofail        bool   `yaml:"Autofail"`
}

type StorageStruct struct {
	AccessMode         string `yaml:"AccessMode"`
	Size               string `yaml:"Size"`
	StorageType        string `yaml:"StorageType"`
	StorageClass       string `yaml:"StorageClass"`
	Fsgroup            string `yaml:"Fsgroup"`
	SupplementalGroups string `yaml:"SupplementalGroups"`
	MatchLabels        string `yaml:"MatchLabels"`
}

type ContainerResourcesStruct struct {
	RequestsMemory string `yaml:"RequestsMemory"`
	RequestsCPU    string `yaml:"RequestsCPU"`
	LimitsMemory   string `yaml:"LimitsMemory"`
	LimitsCPU      string `yaml:"LimitsCPU"`
}

type PgoStruct struct {
	AutofailSleepSeconds string `yaml:"AutofailSleepSeconds"`
	Audit                bool   `yaml:"Audit"`
	LSPVCTemplate        string `yaml:"LSPVCTemplate"`
	LoadTemplate         string `yaml:"LoadTemplate"`
	COImagePrefix        string `yaml:"COImagePrefix"`
	COImageTag           string `yaml:"COImageTag"`
}

type PgoConfig struct {
	BasicAuth                 string                              `yaml:"BasicAuth"`
	Cluster                   ClusterStruct                       `yaml:"Cluster"`
	Pgo                       PgoStruct                           `yaml:"Pgo"`
	ContainerResources        map[string]ContainerResourcesStruct `yaml:"ContainerResources"`
	PrimaryStorage            string                              `yaml:"PrimaryStorage"`
	BackupStorage             string                              `yaml:"BackupStorage"`
	ReplicaStorage            string                              `yaml:"ReplicaStorage"`
	Storage                   map[string]StorageStruct            `yaml:"Storage"`
	DefaultContainerResources string                              `yaml:"DefaultContainerResources"`
}

const DEFAULT_AUTOFAIL_SLEEP_SECONDS = "30"
const DEFAULT_SERVICE_TYPE = "ClusterIP"
const LOAD_BALANCER_SERVICE_TYPE = "LoadBalancer"

func (c *PgoConfig) Validate() error {
	var err error
	log.Info("pgo.yaml Cluster.Backrest is %v", c.Cluster.Backrest)
	_, ok := c.Storage[c.PrimaryStorage]
	if !ok {
		return errors.New("PrimaryStorage setting required")
	}
	_, ok = c.Storage[c.BackupStorage]
	if !ok {
		return errors.New("BackupStorage setting required")
	}
	_, ok = c.Storage[c.ReplicaStorage]
	if !ok {
		return errors.New("ReplicaStorage setting required")
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
	if c.Pgo.AutofailSleepSeconds == "" {
		log.Warn("Pgo.AutofailSleepSeconds not set, using default ")
		c.Pgo.AutofailSleepSeconds = DEFAULT_AUTOFAIL_SLEEP_SECONDS
	}
	_, err = strconv.Atoi(c.Pgo.AutofailSleepSeconds)
	if err != nil {
		return errors.New("Pgo.AutofailSleepSeconds invalid int value found")
	}

	if c.DefaultContainerResources != "" {
		_, ok = c.ContainerResources[c.DefaultContainerResources]
		if !ok {
			return errors.New("DefaultContainerResources setting invalid")
		}
	}

	if c.Cluster.ArchiveMode == "" {
		log.Info("Pgo.Cluster.ArchiveMode not set, using 'false'")
		c.Cluster.ArchiveMode = "false"
	} else {
		if c.Cluster.ArchiveMode != "true" && c.Cluster.ArchiveMode != "false" {
			return errors.New("Cluster.ArchiveMode invalid value, can either be 'true' or 'false'")
		}
	}

	if c.Cluster.ArchiveTimeout == "" {
		log.Info("Pgo.Cluster.ArchiveTimeout not set, using '60'")
	} else {
		_, err := strconv.Atoi(c.Cluster.ArchiveTimeout)
		if err != nil {
			return errors.New("Cluster.ArchiveTimeout invalid int value found")
		}
	}

	if c.Cluster.ServiceType == "" {
		log.Warn("Cluster.ServiceType not set, using default, ClusterIP ")
		c.Cluster.ServiceType = DEFAULT_SERVICE_TYPE
	} else {
		if c.Cluster.ServiceType != DEFAULT_SERVICE_TYPE &&
			c.Cluster.ServiceType != LOAD_BALANCER_SERVICE_TYPE {
			return errors.New("Cluster.ServiceType is required to be either ClusterIP or LoadBalancer")
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

func (c *PgoConfig) GetConf() *PgoConfig {

	yamlFile, err := ioutil.ReadFile("config/pgo.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func (c *PgoConfig) GetStorageSpec(name string) (crv1.PgStorageSpec, error) {
	var err error
	storage := crv1.PgStorageSpec{}

	s, ok := c.Storage[name]
	if !ok {
		err = errors.New("invalid Storage name " + name)
		log.Error(err)
		return storage, err
	}

	storage.StorageClass = s.StorageClass
	storage.AccessMode = s.AccessMode
	storage.Size = s.Size
	storage.StorageType = s.StorageType
	storage.Fsgroup = s.Fsgroup
	storage.MatchLabels = s.MatchLabels
	storage.SupplementalGroups = s.SupplementalGroups

	return storage, err

}

func (c *PgoConfig) GetContainerResource(name string) (crv1.PgContainerResources, error) {
	var err error
	r := crv1.PgContainerResources{}

	s, ok := c.ContainerResources[name]
	if !ok {
		err = errors.New("invalid Container Resources name " + name)
		log.Error(err)
		return r, err
	}
	r.RequestsMemory = s.RequestsMemory
	r.RequestsCPU = s.RequestsCPU
	r.LimitsMemory = s.LimitsMemory
	r.LimitsCPU = s.LimitsCPU

	return r, err

}
