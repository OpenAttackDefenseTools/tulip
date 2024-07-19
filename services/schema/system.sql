CREATE EXTENSION pgcrypto;
CREATE EXTENSION intarray;
CREATE EXTENSION "uuid-ossp";
CREATE EXTENSION pg_trgm;
CREATE EXTENSION btree_gin;
CREATE EXTENSION btree_gist;

ALTER SYSTEM SET shared_preload_libraries = 'timescaledb', 'pg_hint_plan', 'pg_prewarm', 'auto_explain', 'tulip';
ALTER SYSTEM SET timescaledb.telemetry_level = off;

LOAD 'auto_explain';
ALTER SYSTEM SET auto_explain.log_min_duration = 5000;
ALTER SYSTEM SET auto_explain.log_analyze = true;
ALTER SYSTEM SET auto_explain.log_timing = false;

-- These settings provide ingest speed boost at cost
-- of disabling replication and possible loss of any uncommited data
ALTER SYSTEM SET synchronous_commit = off;
ALTER SYSTEM SET wal_level = minimal;
ALTER SYSTEM SET max_wal_senders = 0;
ALTER SYSTEM SET max_wal_size = '10GB';

-- These settings provide ingest speed boost at cost
-- of data consistency
-- Be prepared to loose ALL data on crash when using these
--ALTER SYSTEM SET fsync = off;
--ALTER SYSTEM SET full_page_writes = off;
