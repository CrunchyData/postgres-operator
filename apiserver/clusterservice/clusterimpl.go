package clusterservice

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	_ "github.com/lib/pq"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ShowCluster ...
func ShowCluster(namespace, name, selector string) msgs.ShowClusterResponse {
	var err error

	response := msgs.ShowClusterResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	myselector := labels.Everything()
	log.Debug("selector is " + selector)
	if selector != "" {
		name = "all"
		myselector, err = labels.Parse(selector)
		if err != nil {
			log.Error("could not parse --selector value " + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
	}

	log.Debugf("label selector is [%v]\n", myselector)

	if name == "all" {
		//get a list of all clusters
		err := apiserver.RESTClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(namespace).
			LabelsSelectorParam(myselector).
			Do().Into(&response.ClusterList)
		if err != nil {
			log.Error("error getting list of clusters" + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		log.Debug("clusters found len is %d\n", len(response.ClusterList.Items))
	} else {
		cluster := crv1.Pgcluster{}
		err := apiserver.RESTClient.Get().
			Resource(crv1.PgclusterResourcePlural).
			Namespace(namespace).
			Name(name).
			Do().Into(&cluster)
		if err != nil {
			log.Error("error getting cluster" + err.Error())
			response.Status.Code = msgs.Error
			response.Status.Msg = err.Error()
			return response
		}
		response.ClusterList.Items = make([]crv1.Pgcluster, 1)
		response.ClusterList.Items[0] = cluster
	}

	return response

}

// TestCluster ...
func TestCluster(namespace, name string) msgs.ClusterTestResponse {
	var err error

	response := msgs.ClusterTestResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	cluster := crv1.Pgcluster{}
	err = apiserver.RESTClient.Get().
		Resource(crv1.PgclusterResourcePlural).
		Namespace(namespace).
		Name(name).
		Do().Into(&cluster)

	if kerrors.IsNotFound(err) {
		log.Error("no clusters found")
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	if err != nil {
		log.Error("error getting cluster" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=" + cluster.Spec.Name}
	services, err := apiserver.Clientset.CoreV1().Services(namespace).List(lo)
	if err != nil {
		log.Error("error getting list of services" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	lo = meta_v1.ListOptions{LabelSelector: "pg-database=" + cluster.Spec.Name}
	secrets, err := apiserver.Clientset.Core().Secrets(namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	response.Items = make([]msgs.ClusterTestDetail, 0)

	for _, service := range services.Items {
		for _, s := range secrets.Items {
			item := msgs.ClusterTestDetail{}
			username := string(s.Data["username"][:])
			password := string(s.Data["password"][:])
			database := "postgres"
			if username == cluster.Spec.User {
				database = cluster.Spec.Database
			}
			item.PsqlString = "psql -p " + cluster.Spec.Port + " -h " + service.Spec.ClusterIP + " -U " + username + " " + database
			log.Debug(item.PsqlString)
			status := query(username, service.Spec.ClusterIP, cluster.Spec.Port, database, password)
			item.Working = false
			if status {
				item.Working = true
			}
			response.Items = append(response.Items, item)
		}
	}

	return response
}

func query(dbUser, dbHost, dbPort, database, dbPassword string) bool {
	var conn *sql.DB
	var err error

	conn, err = sql.Open("postgres", "sslmode=disable user="+dbUser+" host="+dbHost+" port="+dbPort+" dbname="+database+" password="+dbPassword)
	if err != nil {
		log.Debug(err.Error())
		return false
	}

	var ts string
	var rows *sql.Rows

	rows, err = conn.Query("select now()::text")
	if err != nil {
		log.Debug(err.Error())
		return false
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
		if err = rows.Scan(&ts); err != nil {
			log.Debug(err.Error())
			return false
		}
		log.Debug("returned " + ts)
		return true
	}
	return false

}
