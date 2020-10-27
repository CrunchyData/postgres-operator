package pgadmin

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// N.B. Changing this name will cause a new group to be created and redirect
// connection updates to that new group without any cleanup of the old
// group name
const sgLabel = "Crunchy PostgreSQL Operator"

// DeleteUser deletes the specified user, their servergroups, and servers
func DeleteUser(qr *queryRunner, username string) error {
	uid, err := qr.Query(fmt.Sprintf("SELECT id FROM user WHERE email='%s'", sqlLiteral(username)))
	if err != nil {
		return err
	}

	if uid != "" {
		rm := fmt.Sprintf(
			`DELETE FROM server WHERE user_id='%[1]s';
			DELETE FROM servergroup WHERE user_id='%[1]s';
			DELETE FROM user where id='%[1]s';`, uid)
		err = qr.Exec(rm)
		if err != nil {
			return err
		}
	} // Otherwise treat delete as no-op
	return nil
}

// Sets the login password for the given username in the pgadmin database
// Adds the user to the pgadmin database if it does not exist
func SetLoginPassword(qr *queryRunner, username, pass string) error {
	hp, err := HashPassword(qr, pass)
	if err != nil {
		return err
	}

	// Idempotent user insertion and update, this implies that setting a
	// password (e.g. update) will establish a user entry
	//
	// role_id(2) == User role (vs 1:Administrator)
	//
	// Bulk query to reduce loss potential from exec errors
	query := fmt.Sprintf(
		`INSERT OR IGNORE INTO user(email,password,active) VALUES ('%[1]s','%[2]s',1);
		INSERT OR IGNORE INTO roles_users(user_id, role_id) VALUES 
		    ((SELECT id FROM user WHERE email='%[1]s'), 2);
		UPDATE user SET password='%[2]s' WHERE email='%[1]s';`, sqlLiteral(username), hp,
	)

	if err := qr.Exec(query); err != nil {
		return err
	}
	return nil
}

// Configures a PG connection for the given username in the pgadmin database
func SetClusterConnection(qr *queryRunner, username string, dbInfo ServerEntry) error {
	// Encryption key for db connections is the user's login password hash
	//
	result, err := qr.Query(fmt.Sprintf("SELECT id, password FROM user WHERE email='%s';", sqlLiteral(username)))
	if err != nil {
		return err
	}
	if result == "" {
		return fmt.Errorf("error: no user found for [%s]", username)
	}

	fields := strings.SplitN(result, qr.Separator(), 2)
	uid, encKey := fields[0], fields[1]

	encPassword := encrypt(dbInfo.Password, encKey)
	// Insert entries into servergroups and servers for the dbInfo provided
	addSG := fmt.Sprintf(`INSERT OR IGNORE INTO servergroup(user_id,name)
		VALUES('%s','%s');`, uid, sgLabel)
	hasSvc := fmt.Sprintf(`SELECT name FROM server WHERE user_id = '%s';`, uid)
	addSvc := fmt.Sprintf(`INSERT INTO server(user_id, servergroup_id,
		name, host, port, maintenance_db, username, password, ssl_mode,
		comment) VALUES ('%[1]s',
			(SELECT id FROM servergroup WHERE user_id='%[1]s' AND name='%s'),
			'%s', '%s', %d, '%s', '%s', '%s', '%s', '%s');`,
		uid,     // user_id && servergroup_id %s (user_id)
		sgLabel, // servergroup_id %s (name)
		dbInfo.Name,
		dbInfo.Host,
		dbInfo.Port,
		dbInfo.MaintenanceDB,
		sqlLiteral(username),
		encPassword,
		dbInfo.SSLMode,
		dbInfo.Comment,
	)
	updSvcPass := fmt.Sprintf("UPDATE server SET password='%s' WHERE user_id = '%s';", encPassword, uid)
	if err := qr.Exec(addSG); err != nil {
		return err
	}
	serverName, err := qr.Query(hasSvc)
	if err != nil {
		return err
	}
	if serverName == "" {
		if err := qr.Exec(addSvc); err != nil {
			return err
		}
	} else {
		// Currently, ignoring overwriting existing entry as the user may have
		// modified through app, but ensure password updates make it through
		// to avoid the user inconvenience of entering their password
		if err := qr.Exec(updSvcPass); err != nil {
			return err
		}
	}
	return nil
}

// GetUsernames provides a list of the provisioned pgadmin login users
func GetUsernames(qr *queryRunner) ([]string, error) {
	q := "SELECT email FROM user WHERE active=1 AND id>1"
	results, err := qr.Query(q)

	return strings.Split(results, "\n"), err
}

// GetPgAdminQueryRunner takes cluster information, identifies whether
// it has a pgAdmin deployment and provides a query runner for executing
// queries against the pgAdmin database
//
// The pointer will be nil if there is no pgAdmin deployed for the cluster
func GetPgAdminQueryRunner(clientset kubernetes.Interface, restconfig *rest.Config, cluster *crv1.Pgcluster) (*queryRunner, error) {
	ctx := context.TODO()

	if active, ok := cluster.Labels[config.LABEL_PGADMIN]; !ok || active != "true" {
		return nil, nil
	}

	selector := fmt.Sprintf("%s=true,%s=%s", config.LABEL_PGADMIN, config.LABEL_PG_CLUSTER, cluster.Name)

	pods, err := clientset.CoreV1().Pods(cluster.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Errorf("failed to find pgadmin pod [%v]", err)
		return nil, err
	}

	// pgAdmin deployment is single-replica, not HA, should only be one pod
	if l := len(pods.Items); l > 1 {
		log.Warnf("Unexpected number of pods for pgadmin [%d], defaulting to first", l)
	} else if l == 0 {
		err := fmt.Errorf("Unable to find pgadmin pod for cluster %s, deleting instance", cluster.Name)
		return nil, err
	}

	return NewQueryRunner(clientset, restconfig, pods.Items[0]), nil
}

// ServerEntryFromPgService populates the ServerEntry struct based on
// details of the kubernetes service, it is up to the caller to provide
// the assumed PgCluster service
func ServerEntryFromPgService(service *v1.Service, clustername string) ServerEntry {
	dbService := ServerEntry{
		Name:          clustername,
		Host:          service.Spec.ClusterIP,
		Port:          5432,
		SSLMode:       "prefer",
		MaintenanceDB: clustername,
	}

	// Set Port info
	for _, portInfo := range service.Spec.Ports {
		if portInfo.Name == "postgres" {
			dbService.Port = int(portInfo.Port)
		}
	}
	return dbService
}

// sqlLiteral escapes single quotes in strings
func sqlLiteral(s string) string {
	return strings.ReplaceAll(s, `'`, `''`)
}
