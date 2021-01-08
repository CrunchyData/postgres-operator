/*
 * Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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

--- System Setup
SET application_name="container_setup";

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
CREATE EXTENSION IF NOT EXISTS pgaudit;

CREATE USER "${PGHA_USER}" LOGIN;
ALTER USER "${PGHA_USER}" PASSWORD $$${PGHA_USER_PASSWORD}$$;

CREATE USER testuser2 LOGIN;
ALTER USER testuser2 PASSWORD 'customconfpass';

CREATE DATABASE "${PGHA_DATABASE}";
GRANT ALL PRIVILEGES ON DATABASE "${PGHA_DATABASE}" TO testuser2;

--- PGHA_DATABASE Setup

\c "${PGHA_DATABASE}"

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
CREATE EXTENSION IF NOT EXISTS pgaudit;

CREATE SCHEMA IF NOT EXISTS AUTHORIZATION "${PGHA_USER}";

/* The following has been customized for the custom-config example */

SET SESSION AUTHORIZATION testuser2;

CREATE TABLE custom_config_table (
	KEY VARCHAR(30) PRIMARY KEY,
	VALUE VARCHAR(50) NOT NULL,
	UPDATEDT TIMESTAMP NOT NULL
);

INSERT INTO custom_config_table (KEY, VALUE, UPDATEDT) VALUES ('CPU', '256', now());

GRANT ALL ON custom_config_table TO testuser2;
