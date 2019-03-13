package upgradeservice

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
	"errors"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"strings"
)

// MajorUpgrade major upgrade type
const MajorUpgrade = "major"

// MinorUpgrade minor upgrade type
const MinorUpgrade = "minor"

const separator = "-"

// ShowUpgrade ...
func ShowUpgrade(name, ns string) msgs.ShowUpgradeResponse {
	response := msgs.ShowUpgradeResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if name == "all" {
		//get a list of all upgrades
		err := kubeapi.Getpgupgrades(apiserver.RESTClient,
			&response.UpgradeList, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debugf("upgrades found len is %d", len(response.UpgradeList.Items))
	} else {
		upgrade := crv1.Pgupgrade{}
		found, err := kubeapi.Getpgupgrade(apiserver.RESTClient,
			&upgrade, name, ns)
		if !found {
			response.Status.Code = msgs.Error
			response.Status.Msg = "upgrade not found"
			return response
		}
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.UpgradeList.Items = make([]crv1.Pgupgrade, 1)
		response.UpgradeList.Items[0] = upgrade
	}

	return response

}

// DeleteUpgrade ...
func DeleteUpgrade(name, ns string) msgs.DeleteUpgradeResponse {
	response := msgs.DeleteUpgradeResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 1)

	if name == "all" {
		err := kubeapi.DeleteAllpgupgrade(apiserver.RESTClient,
			ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.Results[0] = "all"
	} else {
		err := kubeapi.Deletepgupgrade(apiserver.RESTClient,
			name, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.Results[0] = name
	}

	return response

}

// CreateUpgrade ...
func CreateUpgrade(request *msgs.CreateUpgradeRequest, ns string) msgs.CreateUpgradeResponse {
	response := msgs.CreateUpgradeResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 1)

	log.Debugf("createUpgrade called %v", request)

	var newInstance *crv1.Pgupgrade

	if request.UpgradeType == MajorUpgrade {
		response.Status.Code = msgs.Error
		response.Status.Msg = "MajorUpgrade not supported yet"
		return response
	}

	if request.Selector != "" {
		//use the selector instead of an argument list to filter on

		myselector, err := labels.Parse(request.Selector)
		if err != nil {
			log.Error("could not parse selector flag")
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debugf("myselector is %s", myselector.String())

		//get the clusters list
		clusterList := crv1.PgclusterList{}
		err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient,
			&clusterList, request.Selector, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		if len(clusterList.Items) == 0 {
			log.Debug("no clusters found")
			response.Status.Msg = "no clusters found"
			return response
		} else {
			newargs := make([]string, 0)
			for _, cluster := range clusterList.Items {
				newargs = append(newargs, cluster.Spec.Name)
			}
			request.Args = newargs
		}
	}

	for _, arg := range request.Args {
		log.Debugf("create upgrade called for %s", arg)
		result := crv1.Pgupgrade{}

		// error if it already exists
		found, err := kubeapi.Getpgupgrade(apiserver.RESTClient,
			&result, arg, ns)
		if err == nil {
			log.Warn("previous pgupgrade " + arg + " was found so we will remove it.")
			delMsg := DeleteUpgrade(arg, ns)
			if delMsg.Status.Code == msgs.Error {
				log.Error("could not delete previous pgupgrade " + arg)
			}
		} else if !found {
			log.Debugf("pgupgrade %s not found so we will create it", arg)
		} else {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		cl := crv1.Pgcluster{}

		found, err = kubeapi.Getpgcluster(apiserver.RESTClient,
			&cl, arg, ns)
		if !found {
			response.Status.Code = msgs.Error
			response.Status.Msg = arg + " is not a valid pgcluster"
			return response
		}

		// Create an instance of our CRD
		newInstance, err = getUpgradeParams(arg, cl.Spec.CCPImageTag, cl.Spec.CCPImage, request, ns)
		if err == nil {
			err = kubeapi.Createpgupgrade(apiserver.RESTClient,
				newInstance, ns)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			} else {
				msg := "created Pgupgrade " + arg
				log.Debug(msg)
				response.Results = append(response.Results, msg)
			}
		} else {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

	}

	return response
}

func getUpgradeParams(name, currentImageTag, currentImage string, request *msgs.CreateUpgradeRequest, ns string) (*crv1.Pgupgrade, error) {

	var err error
	var found bool
	var existingImage, strRep string
	var existingMajorVersion float64

	spec := crv1.PgupgradeSpec{
		Namespace:       ns,
		Name:            name,
		ResourceType:    "cluster",
		UpgradeType:     request.UpgradeType,
		CCPImage:        currentImage,
		CCPImageTag:     apiserver.Pgo.Cluster.CCPImageTag,
		StorageSpec:     crv1.PgStorageSpec{},
		OldDatabaseName: "??",
		NewDatabaseName: "??",
		OldVersion:      "??",
		NewVersion:      "??",
		OldPVCName:      "",
		NewPVCName:      "",
	}

	_, strRep, err = parseMajorVersion(currentImageTag)
	if err != nil {
		log.Error("error getting pgcluster " + name)
		log.Error(err.Error())
		return nil, err
	}
	spec.OldVersion = strRep

	storage, _ := apiserver.Pgo.GetStorageSpec(apiserver.Pgo.PrimaryStorage)
	spec.StorageSpec.AccessMode = storage.AccessMode
	spec.StorageSpec.Size = storage.Size

	if request.CCPImageTag != "" {
		log.Debugf("using CCPImageTag from command line %s", request.CCPImageTag)
		spec.CCPImageTag = request.CCPImageTag
	}

	cluster := crv1.Pgcluster{}
	found, err = kubeapi.Getpgcluster(apiserver.RESTClient,
		&cluster, name, ns)
	if err == nil {
		spec.ResourceType = "cluster"
		spec.OldDatabaseName = cluster.Spec.Name
		spec.NewDatabaseName = cluster.Spec.Name + "-upgrade"
		spec.OldPVCName = cluster.Spec.PrimaryStorage.Name
		spec.NewPVCName = cluster.Spec.PrimaryStorage.Name + "-upgrade"
		spec.BackupPVCName = cluster.Spec.BackupPVCName
		existingImage = cluster.Spec.CCPImageTag
		existingMajorVersion, strRep, err = parseMajorVersion(cluster.Spec.CCPImageTag)
		if err != nil {
			return nil, err
		}
	} else if !found {
		log.Debugf("%s is not a cluster", name)
		return nil, err
	} else {
		log.Error(err.Error())
		return nil, err
	}

	var requestedMajorVersion float64

	if request.CCPImageTag != "" {
		if request.CCPImageTag == existingImage {
			log.Error("CCPImageTag is the same as the cluster here ")
			log.Error("can't upgrade to the same image version")
			log.Error("requested version is " + request.CCPImageTag)
			log.Error("existing version is " + existingImage)
			return nil, errors.New("invalid image tag")
		}
		requestedMajorVersion, strRep, err = parseMajorVersion(request.CCPImageTag)
		if err != nil {
			log.Error(err)
		}
	} else if apiserver.Pgo.Cluster.CCPImageTag == existingImage {
		log.Error("Cluster.CCPImageTag is the same as the cluster")
		log.Error("can't upgrade to the same image version")

		return nil, errors.New("invalid image tag")
	} else {
		requestedMajorVersion, strRep, err = parseMajorVersion(apiserver.Pgo.Cluster.CCPImageTag)
		if err != nil {
			log.Error(err)
		}
	}

	//minor upgrade
	if requestedMajorVersion > existingMajorVersion {
		log.Error("can't do minor upgrade to a newer major version")
		return nil, errors.New("requested minor upgrade to major version is not allowed")
	}

	spec.NewVersion = strRep

	newInstance := &crv1.Pgupgrade{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	return newInstance, err
}

// parseMajorVersion returns a numeric and string representation
func parseMajorVersion(st string) (float64, string, error) {
	var err error
	var strRep string
	parts := strings.Split(st, separator)
	//PG10 makes this a bit harder given its versioning scheme
	// is different than PG9  e.g. 10.0 is sort of like 9.6.0

	fullversion := parts[1]
	fullversionparts := strings.Split(fullversion, ".")
	strippedversion := strings.Replace(fullversion, ".", "", -1)
	numericVersion, err := strconv.ParseFloat(strippedversion, 64)
	if err != nil {
		log.Error(err.Error())
		return numericVersion, strRep, err
	}

	first := strings.Split(fullversion, ".")
	if first[0] == "10" {
		log.Debug("version 10 ")
		numericVersion = +10.0 * 10
		strRep = fullversionparts[0]
	} else {
		log.Debug("assuming version 9")
		numericVersion, err = strconv.ParseFloat(fullversionparts[0]+fullversionparts[1], 64)
		strRep = fullversionparts[0] + "." + fullversionparts[1]
	}

	log.Debugf("parseMajorVersion is %f", numericVersion)

	return numericVersion, strRep, err
}
