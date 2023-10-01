CREATE EXTENSION pgcrypto;
CREATE EXTENSION intarray;
CREATE EXTENSION "uuid-ossp";
CREATE EXTENSION pg_trgm;

-- These settings provide ingest speed boost at cost
-- of disabling replication and possible loss of any uncommited data
ALTER SYSTEM SET synchronous_commit = off;
ALTER SYSTEM SET wal_level = minimal;
ALTER SYSTEM SET max_wal_senders = 0;

-- These settings provide ingest speed boost at cost
-- of data consistency
-- Be prepared to loose ALL data on crash when using these
--ALTER SYSTEM SET fsync = off;
--ALTER SYSTEM SET full_page_writes = off;
