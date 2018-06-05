/*
* Copyright 2016-2018 Crunchy Data Solutions, Inc.
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package util

import (
	"database/sql"
	"fmt"

	log "github.com/Sirupsen/logrus"
	_ "github.com/lib/pq"
)

const (
	replInfoQueryFormat = "SELECT %s(%s(), '0/0')::bigint, %s(%s(), '0/0')::bigint"

	recvV9         = "pg_last_xlog_receive_location"
	replayV9       = "pg_last_xlog_replay_location"
	locationDiffV9 = "pg_xlog_location_diff"

	recvV10         = "pg_last_wal_receive_lsn"
	replayV10       = "pg_last_wal_replay_lsn"
	locationDiffV10 = "pg_wal_lsn_diff"
)

type Replica struct {
	Name   string
	IP     string
	Status *ReplicationInfo
}

type ReplicationInfo struct {
	ReceiveLocation uint64
	ReplayLocation  uint64
}

func GetReplicationInfo(target string) (*ReplicationInfo, error) {
	conn, err := sql.Open("postgres", target)

	if err != nil {
		log.Errorf("Could not connect to: %s", target)
		return nil, err
	}

	defer conn.Close()

	// Get PG version
	var version int

	rows, err := conn.Query("SELECT current_setting('server_version_num')")

	if err != nil {
		log.Errorf("Could not perform query for version: %s", target)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
	}

	// Get replication info
	var replicationInfoQuery string
	var recvLocation uint64
	var replayLocation uint64

	if version < 100000 {
		replicationInfoQuery = fmt.Sprintf(
			replInfoQueryFormat,
			locationDiffV9, recvV9,
			locationDiffV9, replayV9,
		)
	} else {
		replicationInfoQuery = fmt.Sprintf(
			replInfoQueryFormat,
			locationDiffV10, recvV10,
			locationDiffV10, replayV10,
		)
	}

	rows, err = conn.Query(replicationInfoQuery)

	if err != nil {
		log.Errorf("Could not perform replication info query: %s", target)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&recvLocation, &replayLocation); err != nil {
			return nil, err
		}
	}

	return &ReplicationInfo{recvLocation, replayLocation}, nil
}
