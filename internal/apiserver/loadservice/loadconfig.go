package loadservice

/*
Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

type LoadConfig struct {
	PGOImagePrefix    string
	PGOImageTag       string
	DbDatabase        string
	DbUser            string
	DbPort            string
	TableToLoad       string
	FilePath          string
	FileType          string
	PVCName           string
	FSGroup           string
	SupplementalGroup string
}

func (c *LoadConfig) validate() error {
	var err error

	if c.PGOImagePrefix == "" {
		return errors.New("PGOImagePrefix is not supplied")
	}
	if c.PGOImageTag == "" {
		return errors.New("PGOImageTag is not supplied")
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

func (c *LoadConfig) getConf(yamlFile *bytes.Buffer) (*LoadConfig, error) {

	err := yaml.Unmarshal(yamlFile.Bytes(), c)
	if err != nil {
		log.Errorf("Unmarshal: %v", err)
		return c, err
	}

	return c, err
}

func (c *LoadConfig) print() {
	log.Println("LoadConfig...")
	log.Println("PGOImagePrefix:" + c.PGOImagePrefix)
	log.Println("PGOImageTag:" + c.PGOImageTag)
	log.Println("DbDatabase:" + c.DbDatabase)
	log.Println("DbUser:" + c.DbUser)
	log.Println("DbPort:" + c.DbPort)
	log.Println("TableToLoad:" + c.TableToLoad)
	log.Println("FilePath:" + c.FilePath)
	log.Println("FileType:" + c.FileType)
	log.Println("PVCName:" + c.PVCName)
	log.Println("FSGroup:" + c.FSGroup)
	log.Println("SupplementalGroup:" + c.SupplementalGroup)

}

var LoadCfg LoadConfig
