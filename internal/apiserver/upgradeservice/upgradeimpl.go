package upgradeservice

/*
Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"time"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// Currently supported version information for upgrades
const (
	REQUIRED_MAJOR_PGO_VERSION = 4
	MINIMUM_MINOR_PGO_VERSION  = 1
)

// CreateUpgrade accepts the CreateUpgradeRequest performs the necessary validation checks and
// organizes the needed upgrade information before creating the required pgtask
// Command format: pgo upgrade mycluster
func CreateUpgrade(request *msgs.CreateUpgradeRequest, ns, pgouser string) msgs.CreateUpgradeResponse {
	ctx := context.TODO()
	response := msgs.CreateUpgradeResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	response.Results = make([]string, 0)

	log.Debugf("createUpgrade called %v", request)

	if request.Selector != "" {
		// use the selector instead of an argument list to filter on

		myselector, err := labels.Parse(request.Selector)
		if err != nil {
			log.Error("could not parse selector flag")
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debugf("myselector is %s", myselector.String())

		// get the clusters list

		clusterList, err := apiserver.Clientset.
			CrunchydataV1().Pgclusters(ns).
			List(ctx, metav1.ListOptions{LabelSelector: request.Selector})
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}

		// check that the cluster can be found
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

	for _, clusterName := range request.Args {
		log.Debugf("create upgrade called for %s", clusterName)

		// build the pgtask for the upgrade
		spec := crv1.PgtaskSpec{}
		spec.TaskType = crv1.PgtaskUpgrade
		// set the status as created
		spec.Status = crv1.PgtaskUpgradeCreated
		spec.Parameters = make(map[string]string)
		spec.Parameters[config.LABEL_PG_CLUSTER] = clusterName
		spec.Parameters[crv1.PgtaskWorkflowSubmittedStatus] = time.Now().Format(time.RFC3339)

		u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
		if err != nil {
			log.Error(err)
			response.Status.Code = msgs.Error
			response.Status.Msg = fmt.Sprintf("Could not generate UUID for upgrade task. Error: %s", err.Error())
			return response
		}
		spec.Parameters[crv1.PgtaskWorkflowID] = string(u[:len(u)-1])

		if request.UpgradeCCPImageTag != "" {
			// pass the PostGIS CCP Image Tag provided with the upgrade command
			spec.Parameters[config.LABEL_CCP_IMAGE_KEY] = request.UpgradeCCPImageTag
		} else {
			// pass the CCP Image Tag from the apiserver
			spec.Parameters[config.LABEL_CCP_IMAGE_KEY] = apiserver.Pgo.Cluster.CCPImageTag
		}
		// pass the PGO version for the upgrade
		spec.Parameters[config.LABEL_PGO_VERSION] = msgs.PGO_VERSION
		// pass the PGO username for use in the updated CR if missing
		spec.Parameters[config.LABEL_PGOUSER] = pgouser

		spec.Name = clusterName + "-" + config.LABEL_UPGRADE
		labels := make(map[string]string)
		labels[config.LABEL_PG_CLUSTER] = clusterName
		labels[config.LABEL_PGOUSER] = pgouser
		labels[crv1.PgtaskWorkflowID] = spec.Parameters[crv1.PgtaskWorkflowID]

		newInstance := &crv1.Pgtask{
			ObjectMeta: metav1.ObjectMeta{
				Name:   spec.Name,
				Labels: labels,
			},
			Spec: spec,
		}

		// remove any existing pgtask for this upgrade
		task, err := apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Get(ctx, spec.Name, metav1.GetOptions{})

		if err == nil && task.Spec.Status != crv1.CompletedStatus {
			response.Status.Code = msgs.Error
			response.Status.Msg = fmt.Sprintf("Could not upgrade cluster: there exists an ongoing upgrade task: [%s]. If you believe this is an error, try deleting this pgtask CR.", task.Spec.Name)
			return response
		}

		// validate the cluster name and ensure autofail is turned off for each cluster.
		cl, err := apiserver.Clientset.CrunchydataV1().Pgclusters(ns).Get(ctx, clusterName, metav1.GetOptions{})
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = clusterName + " is not a valid pgcluster"
			return response
		}

		// for the upgrade procedure, we only upgrade to the current image used by the
		// Postgres Operator. As such, we will validate that the Postgres Operator version is
		// is supported by the upgrade, unless the --ignore-validation flag is set.
		if !supportedOperatorVersion(cl.ObjectMeta.Labels[config.LABEL_PGO_VERSION]) && !request.IgnoreValidation {
			response.Status.Code = msgs.Error
			response.Status.Msg = "Cannot upgrade " + clusterName + " from Postgres Operator version " + cl.ObjectMeta.Labels[config.LABEL_PGO_VERSION]
			return response
		}

		// for the upgrade procedure, we only upgrade to the current image used by the
		// Postgres Operator. As such, we will validate that the Postgres Operator's configured
		// image tag (first value) is compatible (i.e. is the same Major PostgreSQL version) as the
		// existing cluster's PG value, unless the --ignore-validation flag is set or the --post-gis-image-tag
		// flag is used
		if !upgradeTagValid(cl.Spec.CCPImageTag, spec.Parameters[config.LABEL_CCP_IMAGE_KEY]) && !request.IgnoreValidation && spec.Parameters[config.LABEL_CCP_IMAGE_KEY] != "" {
			log.Debugf("Cannot upgrade from %s to %s. Image must be the same base OS and the upgrade must be within the same major PG version.", cl.Spec.CCPImageTag, spec.Parameters[config.LABEL_CCP_IMAGE_KEY])
			response.Status.Code = msgs.Error
			response.Status.Msg = fmt.Sprintf("cannot upgrade from %s to %s, upgrade task failed.", cl.Spec.CCPImageTag, spec.Parameters[config.LABEL_CCP_IMAGE_KEY])
			return response
		}

		// Create an instance of our CRD
		_, err = apiserver.Clientset.CrunchydataV1().Pgtasks(ns).Create(ctx, newInstance, metav1.CreateOptions{})
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			response.WorkflowID = spec.Parameters[crv1.PgtaskWorkflowID]
			return response
		}

		msg := "created upgrade task for " + clusterName
		response.Results = append(response.Results, msg)
		response.WorkflowID = spec.Parameters[crv1.PgtaskWorkflowID]
	}

	return response
}

// supportedOperatorVersion validates the Postgres Operator version
// information for the candidate pgcluster. If this value is in the
// required range, return true so that the upgrade may continue. Otherwise,
// return false.
func supportedOperatorVersion(version string) bool {
	// get the Operator version
	operatorVersionRegex := regexp.MustCompile(`^(\d)\.(\d)\.(\d)`)
	operatorVersion := operatorVersionRegex.FindStringSubmatch(version)

	// if this regex passes, the returned array should always contain
	// 4 values. At 0, the full match, then 1-3 are the three defined groups
	// If this is not true, the upgrade cannot continue (and we won't want to
	// reference potentially missing array items).
	if len(operatorVersion) != 4 {
		return false
	}

	// if the first group does not equal the current major version
	// then the upgrade cannot continue
	if major, err := strconv.Atoi(operatorVersion[1]); err != nil {
		log.Error(err)
		return false
	} else if major != REQUIRED_MAJOR_PGO_VERSION {
		return false
	}

	// if the second group does is not in the supported range,
	// then the upgrade cannot continue
	minor, err := strconv.Atoi(operatorVersion[2])
	if err != nil {
		log.Errorf("Cannot convert Postgres Operator's minor version to an integer. Error: %v", err)
		return false
	}

	// If none of the above is true, the upgrade can continue
	return minor >= MINIMUM_MINOR_PGO_VERSION
}

// upgradeTagValid compares and validates the PostgreSQL version values stored
// in the image tag of the existing pgcluster CR against the values set in the
// Postgres Operator's configuration
// A typical example tag is `ubi8-12.9-4.7.4`, so we want to extract and
// compare that `12.9` to make sure that we are only allowing minor upgrades.
// For major upgrades, see PGOv5.1.
func upgradeTagValid(upgradeFrom, upgradeTo string) bool {
	log.Debugf("Validating upgrade from %s to %s", upgradeFrom, upgradeTo)

	versionRegex := regexp.MustCompile(`-(\d+)\.(\d+)\.?(\d+)?-`)

	// get the PostgreSQL version values
	upgradeFromValue := versionRegex.FindStringSubmatch(upgradeFrom)
	upgradeToValue := versionRegex.FindStringSubmatch(upgradeTo)

	// if this regex passes, the returned array should always contain
	// 4 values:
	// 	-At 0, the full match;
	// 	-At 1, the major version of PG;
	//	-At 2, the minor version of PG, which needs to be compared as ints;
	// 	-At 3, the patch version, which can be null, but if not, should be compared as ints;
	// (Note the `?` in the regex after the last capture group.)
	if len(upgradeFromValue) != 4 || len(upgradeToValue) != 4 {
		return false
	}

	// if the first group does not match (i.e., the PG major version), or if a value is
	// missing, then the upgrade cannot continue
	if upgradeFromValue[1] != upgradeToValue[1] && upgradeToValue[1] != "" {
		return false
	}

	// if the above check passed, and there is no patch version value, then the PG
	// version has only two digits (e.g. PG 12.6, 12.10), meaning this is a minor upgrade.
	// After validating the second value is at least equal (this is to allow for multiple executions of the
	// upgrade in case an error occurs), the upgrade can continue
	// In order to compare correctly, these values have to be ints.
	// Note: thanks to the regex capture, we know these second values consist of digits,
	// so we can skip testing the error.

	upgradeFromInt, _ := strconv.Atoi(upgradeFromValue[2])
	upgradeToInt, _ := strconv.Atoi(upgradeToValue[2])
	if upgradeFromValue[3] == "" && upgradeToValue[3] == "" && upgradeFromInt <= upgradeToInt {
		return true
	}

	// finally, if the second group matches and is not empty, then, based on the
	// possibilities remaining for Operator container image tags, this is either PG 9.5 or 9.6.
	// if the second group value matches, check that the third value is not empty and
	// at least equal (this is to allow for multiple executions of the
	// upgrade in case an error occurs). If so, the upgrade can continue.
	if upgradeFromValue[2] == upgradeToValue[2] && upgradeToValue[2] != "" &&
		upgradeFromValue[3] != "" && upgradeToValue[3] != "" {
		upgradeFromInt, _ = strconv.Atoi(upgradeFromValue[3])
		upgradeToInt, _ = strconv.Atoi(upgradeToValue[3])
		if upgradeFromInt <= upgradeToInt {
			return true
		}
	}

	// if none of the above conditions are met, a two digit Major version upgrade is likely being
	// attempted, or a tag value or general error occurred, so we cannot continue
	return false
}
