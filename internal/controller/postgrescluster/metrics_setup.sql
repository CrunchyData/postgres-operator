--
-- Copyright © 2017-2025 Crunchy Data Solutions, Inc. All Rights Reserved.
--

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ccp_monitoring') THEN
        CREATE ROLE ccp_monitoring WITH LOGIN;
    END IF;

    -- The pgmonitor role is required by the pgnodemx extension in PostgreSQL versions 9.5 and 9.6
    -- and should be removed when upgrading to PostgreSQL 10 and above.
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pgmonitor') THEN
        DROP ROLE pgmonitor;
    END IF;
END
$$;

GRANT pg_monitor to ccp_monitoring;
GRANT pg_execute_server_program TO ccp_monitoring;

ALTER ROLE ccp_monitoring SET lock_timeout TO '2min';
ALTER ROLE ccp_monitoring SET jit TO 'off';

CREATE SCHEMA IF NOT EXISTS monitor AUTHORIZATION ccp_monitoring;

DROP TABLE IF EXISTS monitor.pg_stat_statements_reset_info;
-- Table to store last reset time for pg_stat_statements
CREATE TABLE monitor.pg_stat_statements_reset_info(
   reset_time timestamptz
);

DROP FUNCTION IF EXISTS monitor.pg_stat_statements_reset_info(int);
-- Function to reset pg_stat_statements periodically
CREATE FUNCTION monitor.pg_stat_statements_reset_info(p_throttle_minutes integer DEFAULT 1440)
  RETURNS bigint
  LANGUAGE plpgsql
  SECURITY DEFINER
  SET search_path TO pg_catalog, pg_temp
AS $function$
DECLARE

  v_reset_timestamp      timestamptz;
  v_throttle             interval;

BEGIN

  IF p_throttle_minutes < 0 THEN
      RETURN 0;
  END IF;

  v_throttle := make_interval(mins := p_throttle_minutes);

  SELECT COALESCE(max(reset_time), '1970-01-01'::timestamptz) INTO v_reset_timestamp FROM monitor.pg_stat_statements_reset_info;

  IF ((CURRENT_TIMESTAMP - v_reset_timestamp) > v_throttle) THEN
      -- Ensure table is empty
      DELETE FROM monitor.pg_stat_statements_reset_info;
      PERFORM pg_stat_statements_reset();
      INSERT INTO monitor.pg_stat_statements_reset_info(reset_time) values (now());
  END IF;

  RETURN (SELECT extract(epoch from reset_time) FROM monitor.pg_stat_statements_reset_info);

EXCEPTION
   WHEN others then
       RETURN 0;
END
$function$;

GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA monitor TO ccp_monitoring;
GRANT ALL ON ALL TABLES IN SCHEMA monitor TO ccp_monitoring;

--- get_pgbackrest_info is used by the OTel collector.
--- get_replication_lag is created as function, so that we can query without warning on a replica.
CREATE OR REPLACE FUNCTION get_replication_lag() RETURNS TABLE(bytes NUMERIC) AS $$
BEGIN
    IF pg_is_in_recovery() THEN
        RETURN QUERY SELECT 0::NUMERIC AS bytes;
    ELSE
        RETURN QUERY SELECT pg_wal_lsn_diff(sent_lsn, replay_lsn) AS bytes
                     FROM pg_catalog.pg_stat_replication;
    END IF;
END;
$$ LANGUAGE plpgsql;

--- get_pgbackrest_info is used by the OTel collector.
--- get_pgbackrest_info is created as a function so that no ddl runs on a replica.
--- In the query, the --stanza argument matches DefaultStanzaName, defined in internal/pgbackrest/config.go.
CREATE OR REPLACE FUNCTION get_pgbackrest_info()
RETURNS TABLE (
    last_diff_backup BIGINT,
    last_full_backup BIGINT,
    last_incr_backup BIGINT,
    last_info_backrest_repo_version TEXT,
    last_info_backup_error INT,
    backup_type TEXT,
    backup_runtime_seconds BIGINT,
    repo_backup_size_bytes TEXT,
    repo_total_size_bytes TEXT,
    oldest_full_backup BIGINT,
    repo TEXT
) AS $$
BEGIN
    IF pg_is_in_recovery() THEN
        RETURN QUERY
        SELECT
            0::bigint AS last_diff_backup,
            0::bigint AS last_full_backup,
            0::bigint AS last_incr_backup,
            '0' AS last_info_backrest_repo_version,
            0::int AS last_info_backup_error,
            'n/a'::text AS backup_type,
            0::bigint AS backup_runtime_seconds,
            '0'::text AS repo_backup_size_bytes,
            '0'::text AS repo_total_size_bytes,
            0::bigint AS oldest_full_backup,
            'n/a' AS repo;
    ELSE
        DROP TABLE IF EXISTS pgbackrest_info;
        CREATE TEMPORARY TABLE pgbackrest_info (data json);
        COPY pgbackrest_info (data)
        FROM PROGRAM 'export LC_ALL=C && printf "\f" && pgbackrest info --log-level-console=info --log-level-stderr=warn --output=json --stanza=db && printf "\f"'
        WITH (FORMAT csv, HEADER false, QUOTE E'\f');

        RETURN QUERY
        WITH
        all_backups (data) AS (
            SELECT jsonb_array_elements(to_jsonb(data)) FROM pgbackrest_info
        ),
        stanza_backups (stanza, backup) AS (
            SELECT data->>'name', jsonb_array_elements(data->'backup') FROM all_backups
        ),
        ordered_backups (stanza, backup, seq_oldest, seq_newest) AS (
            SELECT stanza, backup,
                ROW_NUMBER() OVER (
                    PARTITION BY stanza, backup->'database'->>'repo-key', backup->>'type'
                    ORDER BY backup->'timestamp'->>'start' ASC, backup->'timestamp'->>'stop' ASC
                ),
                ROW_NUMBER() OVER (
                    PARTITION BY stanza, backup->'database'->>'repo-key', backup->>'type'
                    ORDER BY backup->'timestamp'->>'start' DESC, backup->'timestamp'->>'stop' DESC
                )
            FROM stanza_backups
        ),

        ccp_backrest_last_info AS (
            SELECT
                stanza,
                split_part(backup->'backrest'->>'version', '.', 1) || lpad(split_part(backup->'backrest'->>'version', '.', 2), 2, '0') || lpad(coalesce(nullif(split_part(backup->'backrest'->>'version', '.', 3), ''), '00'), 2, '0') AS backrest_repo_version,
                backup->'database'->>'repo-key' AS repo,
                backup->>'type' AS backup_type,
                backup->'info'->'repository'->>'delta' AS repo_backup_size_bytes,
                backup->'info'->'repository'->>'size' AS repo_total_size_bytes,
                (backup->'timestamp'->>'stop')::bigint - (backup->'timestamp'->>'start')::bigint AS backup_runtime_seconds,
                CASE WHEN backup->>'error' = 'true' THEN 1 ELSE 0 END AS backup_error
            FROM ordered_backups
            WHERE seq_newest = 1
        ),

        ccp_backrest_oldest_full_backup AS (
            SELECT
                stanza,
                backup->'database'->>'repo-key' AS repo,
                min((backup->'timestamp'->>'stop')::bigint) AS time_seconds
            FROM ordered_backups
            WHERE seq_oldest = 1 AND backup->>'type' IN ('full')
            GROUP BY 1,2
        ),

        ccp_backrest_last_full_backup AS (
            SELECT
                stanza,
                backup->'database'->>'repo-key' AS repo,
                EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint - max((backup->'timestamp'->>'stop')::bigint) AS time_since_completion_seconds
            FROM ordered_backups
            WHERE seq_newest = 1 AND backup->>'type' IN ('full')
            GROUP BY 1,2
        ),

        ccp_backrest_last_diff_backup AS (
            SELECT
                stanza,
                backup->'database'->>'repo-key' AS repo,
                EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint - max((backup->'timestamp'->>'stop')::bigint) AS time_since_completion_seconds
            FROM ordered_backups
            WHERE seq_newest = 1 AND backup->>'type' IN ('full','diff')
            GROUP BY 1,2
        ),

        ccp_backrest_last_incr_backup AS (
            SELECT
                stanza,
                backup->'database'->>'repo-key' AS repo,
                EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint - max((backup->'timestamp'->>'stop')::bigint) AS time_since_completion_seconds
            FROM ordered_backups
            WHERE seq_newest = 1 AND backup->>'type' IN ('full','diff','incr')
            GROUP BY 1,2
        )

        SELECT
            ccp_backrest_last_diff_backup.time_since_completion_seconds,
            ccp_backrest_last_full_backup.time_since_completion_seconds,
            ccp_backrest_last_incr_backup.time_since_completion_seconds,
            ccp_backrest_last_info.backrest_repo_version,
            ccp_backrest_last_info.backup_error,
            ccp_backrest_last_info.backup_type,
            ccp_backrest_last_info.backup_runtime_seconds,
            ccp_backrest_last_info.repo_backup_size_bytes,
            ccp_backrest_last_info.repo_total_size_bytes,
            ccp_backrest_oldest_full_backup.time_seconds,
            ccp_backrest_last_incr_backup.repo
        FROM
            ccp_backrest_last_diff_backup
            JOIN ccp_backrest_last_full_backup ON ccp_backrest_last_diff_backup.stanza = ccp_backrest_last_full_backup.stanza AND ccp_backrest_last_diff_backup.repo = ccp_backrest_last_full_backup.repo
            JOIN ccp_backrest_last_incr_backup ON ccp_backrest_last_diff_backup.stanza = ccp_backrest_last_incr_backup.stanza AND ccp_backrest_last_diff_backup.repo = ccp_backrest_last_incr_backup.repo
            JOIN ccp_backrest_last_info ON ccp_backrest_last_diff_backup.stanza = ccp_backrest_last_info.stanza AND ccp_backrest_last_diff_backup.repo = ccp_backrest_last_info.repo
            JOIN ccp_backrest_oldest_full_backup ON ccp_backrest_last_diff_backup.stanza = ccp_backrest_oldest_full_backup.stanza AND ccp_backrest_last_diff_backup.repo = ccp_backrest_oldest_full_backup.repo;
    END IF;
END;
$$ LANGUAGE plpgsql;

