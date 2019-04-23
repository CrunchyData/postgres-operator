package dfservice

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
	"database/sql"
	"errors"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"strings"
)

func DfCluster(name, selector, ns string) msgs.DfResponse {
	var err error

	response := msgs.DfResponse{}
	response.Results = make([]msgs.DfDetail, 0)
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if selector == "" && name == "all" {
		log.Debug("selector is empty and name is all")
	} else {
		if selector == "" {
			selector = "name=" + name
		}
	}
	log.Debugf("df selector is %s", selector)

	//get a list of matching clusters
	clusterList := crv1.PgclusterList{}
	err = kubeapi.GetpgclustersBySelector(apiserver.RESTClient, &clusterList, selector, ns)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	//loop thru each cluster

	log.Debugf("df clusters found len is %d", len(clusterList.Items))

	for _, c := range clusterList.Items {

		selector := config.LABEL_SERVICE_NAME + "=" + c.Spec.Name

		pods, err := kubeapi.GetPods(apiserver.Clientset, selector, ns)
		if err != nil {
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		for _, p := range pods.Items {
			if strings.Contains(p.Name, "-pgbouncer") ||
				strings.Contains(p.Name, "-pgpool") {
				continue
			}

			var pvcName string
			var found bool
			for _, v := range p.Spec.Volumes {
				if v.Name == "pgdata" {
					found = true
					pvcName = v.VolumeSource.PersistentVolumeClaim.ClaimName

				}
			}
			if !found {
				response.Status.Code = msgs.Error
				response.Status.Msg = "pgdata volume not found in pod "
				return response
			}
			log.Debugf("pvc found to be %s", pvcName)

			result := msgs.DfDetail{}
			result.Name = p.Name
			result.Working = true

			pgSizePretty, pgSize, err := getPGSize(c.Spec.Port, p.Status.PodIP, "postgres", c.Spec.Name, ns)
			log.Debugf("podName=%s pgSize=%d pgSize=%s cluster=%s", p.Name, pgSize, pgSizePretty, c.Spec.Name)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
			result.PGSize = pgSizePretty
			result.ClaimSize, err = getClaimCapacity(apiserver.Clientset, pvcName, ns)
			if err != nil {
				response.Status.Code = msgs.Error
				response.Status.Msg = err.Error()
				return response
			}
			diskSize := resource.MustParse(result.ClaimSize)
			diskSizeInt64, _ := diskSize.AsInt64()
			diskSizeFloat := float64(diskSizeInt64)

			result.Pct = int64((float64(pgSize) / diskSizeFloat) * 100.0)

			response.Results = append(response.Results, result)
		}

	}

	return response
}

// getPrimarySecret get only the primary postgres secret
func getPrimarySecret(clusterName, ns string) (string, string, error) {

	selector := "pgpool!=true,pg-cluster=" + clusterName

	secrets, err := kubeapi.GetSecrets(apiserver.Clientset, selector, ns)
	if err != nil {
		return "", "", err
	}

	secretName := clusterName + "-postgres-secret"
	for _, s := range secrets.Items {
		if s.Name == secretName {
			username := string(s.Data["username"][:])
			password := string(s.Data["password"][:])
			return username, password, err
		}

	}

	return "", "", errors.New("secret " + secretName + " not found")
}

// getPrimaryService returns the service IP addresses
func getServices(clusterName, ns string) (map[string]string, error) {

	output := make(map[string]string, 0)
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName

	services, err := kubeapi.GetServices(apiserver.Clientset, selector, ns)
	if err != nil {
		return output, err
	}

	for _, p := range services.Items {
		output[p.Name] = p.Spec.ClusterIP
	}

	return output, err
}

// getPGSize clusterName returns sizestring, error
func getPGSize(port, host, databaseName, clusterName, ns string) (string, int, error) {
	var dbsizePretty string
	var dbsize int
	var conn *sql.DB

	username, password, err := getPrimarySecret(clusterName, ns)
	if err != nil {
		log.Error(err.Error())
		return dbsizePretty, dbsize, err
	}
	//log.Debug("username=" + username + " password=" + password)

	conn, err = sql.Open("postgres", "sslmode=disable user="+username+" host="+host+" port="+port+" dbname="+databaseName+" password="+password)
	if err != nil {
		log.Error(err.Error())
		return dbsizePretty, dbsize, err
	}

	var rows *sql.Rows

	rows, err = conn.Query("select pg_size_pretty(sum(pg_database_size(pg_database.datname))), sum(pg_database_size(pg_database.datname)) from pg_database")
	if err != nil {
		log.Error(err.Error())
		return dbsizePretty, dbsize, err
	}

	defer func() {
		if conn != nil {
			conn.Close()
		}
		if rows != nil {
			rows.Close()
		}
	}()
	for rows.Next() {
		if err = rows.Scan(&dbsizePretty, &dbsize); err != nil {
			log.Error(err.Error())
			return "", 0, err
		}
		log.Debugf("returned %s %d\n", dbsizePretty, dbsize)
		return dbsizePretty, dbsize, err
	}

	return dbsizePretty, dbsize, err

}

func getClaimCapacity(clientset *kubernetes.Clientset, pvcName, ns string) (string, error) {
	var err error
	var found bool
	var pvc *v1.PersistentVolumeClaim

	log.Debugf("in df pvc name found to be %s", pvcName)

	pvc, found, err = kubeapi.GetPVC(clientset, pvcName, ns)
	if err != nil {
		if !found {
			log.Debugf("pvc %s not found", pvcName)
		}
		return "", err
	}
	qty := pvc.Status.Capacity[v1.ResourceStorage]
	log.Debugf("storage cap string value %s\n", qty.String())

	return qty.String(), err

}
