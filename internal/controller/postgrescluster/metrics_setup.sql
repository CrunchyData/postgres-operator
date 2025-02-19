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
