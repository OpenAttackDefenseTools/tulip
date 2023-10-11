CREATE TABLE tag (
	name text PRIMARY KEY,
	sort serial NOT NULL
);

-- This is the order in which tags should be displayed
-- Change it hare if it should be different
-- Newly encountered tags will be automaticcaly
-- added to the end of this list
INSERT INTO tag (name) VALUES
	('tcp'),
	('udp'),
	('http'),
	('flag-in'),
	('flag-out'),
	('blocked'),
	('suricata'),
	('starred');

CREATE TABLE pcap (
	id uuid PRIMARY KEY,
	name text NOT NULL UNIQUE,
	position bigint NOT NULL DEFAULT 0
);

CREATE TABLE fingerprint (
	id int PRIMARY KEY,
	grp int NOT NULL
);

-- Fingerprint matching during assembly
CREATE INDEX ON fingerprint (grp);

CREATE TABLE flow (
	id uuid NOT NULL PRIMARY KEY,
	time timestamptz GENERATED ALWAYS AS (uuid_unpack_time(id)) STORED,
	port_src int NOT NULL,
	port_dst int NOT NULL,
	ip_src inet NOT NULL,
	ip_dst inet NOT NULL,
	duration interval NOT NULL,
	pcap_id uuid NOT NULL,
	link_parent_id uuid,
	link_child_id uuid,
	tags jsonb NOT NULL DEFAULT '[]',
	fingerprints int[] NOT NULL DEFAULT '{}',
	signatures jsonb NOT NULL DEFAULT '[]',
	packets_count int NOT NULL DEFAULT 0,
	packets_size int NOT NULL DEFAULT 0,
	flags_in int NOT NULL DEFAULT 0,
	flags_out int NOT NULL DEFAULT 0
);

-- Suricata id lookup, see Database::SuricataIdFindFlow
CREATE INDEX ON flow (id, port_src, port_dst, ip_src, ip_dst);
-- Tag search
CREATE INDEX ON flow USING gin (tags);
-- Fingerprint matching during assembly
CREATE INDEX ON flow USING gin (fingerprints);

SELECT create_hypertable(
	'flow',
	'id',
	chunk_time_interval => INTERVAL '1 hour',
	time_partitioning_func => 'uuid_unpack_time'
);

CREATE TABLE flow_item (
	id uuid NOT NULL PRIMARY KEY,
	flow_id uuid NOT NULL,
	kind text NOT NULL,
	time timestamptz GENERATED ALWAYS AS (uuid_unpack_time(id)) STORED,
	direction text NOT NULL,
	data bytea NOT NULL,
	text text NOT NULL
);

-- Regex search, this one is chonky
CREATE INDEX ON flow_item USING gin (text gin_trgm_ops);

SELECT create_hypertable(
	'flow_item',
	'id',
	chunk_time_interval => INTERVAL '1 hour',
	time_partitioning_func => 'uuid_unpack_time'
);
