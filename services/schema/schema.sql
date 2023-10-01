CREATE TABLE tag (
	name text PRIMARY KEY
);

INSERT INTO tag (name) VALUES
	('flag-in'),
	('flag-out'),
	('blocked'),
	('suricata'),
	('starred'),
	('tcp'),
	('udp'),
	('http');

CREATE TABLE pcap (
	id uuid PRIMARY KEY,
	name text NOT NULL UNIQUE,
	position bigint NOT NULL DEFAULT 0
);

CREATE TABLE flow (
	id uuid NOT NULL PRIMARY KEY,
	time timestamptz GENERATED ALWAYS AS (uuid_unpack_time(id)) STORED,
	port_src int NOT NULL,
	port_dst int NOT NULL,
	ip_src inet NOT NULL,
	ip_dst inet NOT NULL,
	duration interval NOT NULL,
	blocked boolean NOT NULL DEFAULT false,
	pcap_id uuid NOT NULL,
	connected_parent_id uuid,
	connected_child_id uuid,
	tags jsonb NOT NULL DEFAULT '[]',
	fingerprints jsonb NOT NULL DEFAULT '[]',
	signatures jsonb NOT NULL DEFAULT '[]',
	packets_count int NOT NULL DEFAULT 0,
	packets_size int NOT NULL DEFAULT 0,
	flags_in int NOT NULL DEFAULT 0,
	flags_out int NOT NULL DEFAULT 0
);

-- For suricata id lookup, see Database::SuricataIdFindFlow
CREATE INDEX ON flow (id, port_src, port_dst, ip_src, ip_dst);

-- For tag search
CREATE INDEX ON flow USING gin (tags);

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

-- For regex search, this one is chonky
CREATE INDEX ON flow_item USING gin (text gin_trgm_ops);

SELECT create_hypertable(
	'flow_item',
	'id',
	chunk_time_interval => INTERVAL '1 hour',
	time_partitioning_func => 'uuid_unpack_time'
);
