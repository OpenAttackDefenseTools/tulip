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
	('flagid-in'),
	('flagid-out'),
	('blocked'),
	('suricata'),
	('starred');

-- Flag ids
CREATE TABLE flag_id (
	id serial NOT NULL PRIMARY KEY,
	content text NOT NULL,
	time timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON flag_id USING btree (content);

-- Pcaps
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
	time timestamptz GENERATED ALWAYS AS (fid_unpack_time(id)) STORED,
	port_src int NOT NULL,
	port_dst int NOT NULL,
	ip_src inet NOT NULL,
	ip_dst inet NOT NULL,
	duration interval NOT NULL,
	pcap_id uuid NOT NULL,
	link_parent_id uuid,
	link_child_id uuid,
	tags jsonb NOT NULL DEFAULT '[]',
	flags jsonb NOT NULL DEFAULT '[]',
	flagids jsonb NOT NULL DEFAULT '[]',
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
	time_partitioning_func => 'fid_unpack_time'
);

CREATE TABLE flow_item (
	id uuid NOT NULL PRIMARY KEY,
	flow_id uuid NOT NULL,
	kind text NOT NULL,
	time timestamptz GENERATED ALWAYS AS (fid_unpack_time(id)) STORED,
	direction text NOT NULL,
	data bytea NOT NULL
);

SELECT create_hypertable(
	'flow_item',
	'id',
	chunk_time_interval => INTERVAL '1 hour',
	time_partitioning_func => 'fid_unpack_time'
);

-- Table for quick indexed regex search on utf8 data
-- The data in this table is chunked to 1028 characters
-- with 64 character overlap
CREATE TABLE flow_index (
	flow_id uuid NOT NULL,
	text text NOT NULL
);

-- Regex search, this one is chonky
-- This is a GiST rather than GIN since GIN does not support sorting
-- In the future we could use something like RUM (https://github.com/postgrespro/rum)
-- Sadly, RUM does not support fast update like GIN does, so ingestion takes ages
CREATE INDEX ON flow_index USING gist (text gist_trgm_ops(siglen=2024), flow_id gist_fid_ops);
