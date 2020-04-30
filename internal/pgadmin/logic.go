package pgadmin

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"
	"strconv"
	"strings"
)

// N.B. Changing this name will cause a new group to be created and redirect
// connection updates to that new group without any cleanup of the old
// group name
const sgLabel = "Crunchy On-Demand"

// DeleteUser deletes the specified user, their servergroups, and servers
func DeleteUser(qr *queryRunner, username string) error {
	idStr, err := qr.Query(fmt.Sprintf("SELECT id FROM user WHERE email='%s'", username))
	if err != nil {
		return err
	}

	if idStr != "" {
		uid, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("Failed to convert user id [%s] to int: %v", idStr, err)
		}

		rm := fmt.Sprintf(
			`DELETE FROM server WHERE user_id=%d;
			DELETE FROM servergroup WHERE user_id=%d;
			DELETE FROM user where id=%d;`, uid, uid, uid)
		err = qr.Exec(rm)
		if err != nil {
			return err
		}
	} // Otherwise treat delete as no-op
	return nil
}

// Sets the login password for the given username in the pgadmin database
// Adds the user to the pgadmin database if it does not exist
//
// N.B. No particular sanitization is done on the username for preventing
// shenanigans, it is the caller's responsibility as it knows the origin
func SetLoginPassword(qr *queryRunner, username, pass string) error {
	hp, err := HashPassword(qr, pass)
	if err != nil {
		return err
	}

	// Idempotent user insertion and update, this implies that setting a
	// password (e.g. update) will establish a user entry
	insQ := fmt.Sprintf("INSERT OR IGNORE INTO user(email,password,active) VALUES ('%s','%s',1);", username, hp)
	updQ := fmt.Sprintf("UPDATE user SET password='%s' WHERE email='%s';", hp, username)

	if err := qr.Exec(insQ); err != nil {
		return err
	}
	if err := qr.Exec(updQ); err != nil {
		return err
	}

	return nil
}

// Configures a PG connection for the given username in the pgadmin database
//
// N.B. No particular sanitization is done on the username for preventing
// shenanigans, it is the caller's responsibility as it knows the origin
func SetClusterConnection(qr *queryRunner, username string, dbInfo ServerEntry) error {
	// Encryption key for db connections is the user's login password hash
	//
	result, err := qr.Query(fmt.Sprintf("SELECT id, password FROM user WHERE email='%s';", username))
	if err != nil {
		return err
	}
	if result == "" {
		return fmt.Errorf("error: no user found for [%s]", username)
	}

	fields := strings.SplitN(result, qr.Separator(), 2)
	idStr, encKey := fields[0], fields[1]

	uid, err := strconv.Atoi(idStr)
	if err != nil {
		return fmt.Errorf("Failed to convert user id [%s] to int: %v", idStr, err)
	}

	// Insert entries into servergroups and servers for the dbInfo provided
	addSG := fmt.Sprintf(`INSERT OR IGNORE INTO servergroup(user_id,name) 
		VALUES(%d,'%s');`, uid, sgLabel)
	hasSvc := fmt.Sprintf(`SELECT name FROM server WHERE user_id = %d;`, uid)
	addSvc := fmt.Sprintf(`INSERT INTO server(user_id, servergroup_id, 
		name, host, port, maintenance_db, username, password, ssl_mode,
		comment) VALUES (%d, 
			(SELECT id FROM servergroup WHERE user_id=%d AND name='%s'),
			'%s', '%s', %d, '%s', '%s', '%s', '%s', '%s');`,
		uid,     // user_id
		uid,     // servergroup_id %d (user_id)
		sgLabel, // servergroup_id %s (name)
		dbInfo.Name,
		dbInfo.Host,
		dbInfo.Port,
		dbInfo.MaintenanceDB,
		username,
		encrypt(dbInfo.Password, encKey),
		dbInfo.SSLMode,
		dbInfo.Comment,
	)
	updSvcPass := fmt.Sprintf("UPDATE server SET password='%s';", encrypt(dbInfo.Password, encKey))

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
	if err != nil {
		return []string{}, err
	}

	return strings.Split(results, "\n"), nil
}
