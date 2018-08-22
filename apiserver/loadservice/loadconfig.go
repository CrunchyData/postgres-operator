package loadservice

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
	"bytes"
	"errors"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type LoadConfig struct {
	COImagePrefix   string `yaml:"COImagePrefix"`
	COImageTag      string `yaml:"COImageTag"`
	DbDatabase      string `yaml:"DbDatabase"`
	DbUser          string `yaml:"DbUser"`
	DbPort          string `yaml:"DbPort"`
	TableToLoad     string `yaml:"TableToLoad"`
	FilePath        string `yaml:"FilePath"`
	FileType        string `yaml:"FileType"`
	PVCName         string `yaml:"PVCName"`
	SecurityContext string `yaml:"SecurityContext"`
}

func (c *LoadConfig) validate() error {
	var err error

	if c.COImagePrefix == "" {
		return errors.New("COImagePrefix is not supplied")
	}
	if c.COImageTag == "" {
		return errors.New("COImageTag is not supplied")
	}
	if c.DbDatabase == "" {
		return errors.New("DbDatabase is not supplied")
	}
	if c.DbUser == "" {
		return errors.New("DbUser is not supplied")
	}
	if c.DbPort == "" {
		return errors.New("DbPort is not supplied")
	}
	if c.TableToLoad == "" {
		return errors.New("TableToLoad is not supplied")
	}
	if c.FilePath == "" {
		return errors.New("FilePath is not supplied")
	}
	if c.PVCName == "" {
		return errors.New("PVCName is not supplied")
	}

	return err
}

func (c *LoadConfig) getConf(yamlFile *bytes.Buffer) *LoadConfig {

	err := yaml.Unmarshal(yamlFile.Bytes(), c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func (c *LoadConfig) print() {
	log.Println("LoadConfig...")
	log.Println("COImagePrefix:" + c.COImagePrefix)
	log.Println("COImageTag:" + c.COImageTag)
	log.Println("DbDatabase:" + c.DbDatabase)
	log.Println("DbUser:" + c.DbUser)
	log.Println("DbPort:" + c.DbPort)
	log.Println("TableToLoad:" + c.TableToLoad)
	log.Println("FilePath:" + c.FilePath)
	log.Println("FileType:" + c.FileType)
	log.Println("PVCName:" + c.PVCName)
	log.Println("SecurityContext:" + c.SecurityContext)

}

var LoadCfg LoadConfig
