-- Helper function to convert number to bytes (big-endian)
CREATE FUNCTION int_big_endian(num bigint, len int)
RETURNS bytea LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
	result bytea = '';
	index int = 0;
BEGIN
	WHILE index < len LOOP
		result := set_byte('\x00', 0, (num % 256)::int) || result;
		num := (num - num % 256) / 256;
		index := index + 1;
	END LOOP;

	return result;
END; $$;

-- Helper function to convert bytes to number (big-endian)
CREATE FUNCTION big_endian_int(bytes bytea)
RETURNS bigint LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
	result bigint = 0;
	index int = 0;
BEGIN
	WHILE index < octet_length(bytes) LOOP
		result := result * 256 + get_byte(bytes, index);
		index := index + 1;
	END LOOP;

	return result;
END; $$;

-- Function to create UUID based on timestamp and random bytes
CREATE FUNCTION fid_pack("time" timestamptz, random bytea)
RETURNS uuid LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
	time_int bigint = extract(epoch FROM "time" AT TIME ZONE 'UTC') * 1000000;
	time_hex text = encode(int_big_endian(time_int, 8), 'hex');
	random_hex text = substring(encode(random, 'hex') FOR 14);
BEGIN
	RETURN substring(time_hex FOR 12)
		|| '8' || substring(time_hex FROM 13 FOR 3)
		|| '8' || substring(time_hex FROM 16 FOR 1)
		|| random_hex;
END; $$;

CREATE FUNCTION fid_pack_low("time" timestamptz)
RETURNS uuid LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
	RETURN fid_pack("time", '\x00000000000000');
END; $$;

CREATE FUNCTION fid_pack_high("time" timestamptz)
RETURNS uuid LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
	RETURN fid_pack("time", '\xffffffffffffff');
END; $$;

-- Functions to extract UUID values
CREATE FUNCTION fid_unpack_time(id uuid)
RETURNS timestamptz LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
	hex text := replace(id::text, '-', '');
	time_hex text := substring(hex FOR 12) || substring(hex FROM 14 FOR 3) || substring(hex FROM 18 FOR 1);
BEGIN
	RETURN to_timestamp(big_endian_int(decode(time_hex, 'hex'))::float / 1000000)::timestamptz;
END; $$;

CREATE FUNCTION fid_unpack_random(id uuid)
RETURNS bytea LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
	hex text := replace(id::text, '-', '');
	random_hex text := substring(hex FROM 19 FOR 14);
BEGIN
	RETURN decode(random_hex, 'hex');
END; $$;

CREATE FUNCTION fid_create("time" timestamptz)
RETURNS uuid LANGUAGE plpgsql VOLATILE PARALLEL SAFE AS $$
BEGIN
	RETURN fid_pack("time", gen_random_bytes(7));
END; $$;

-- Min / Max functions
CREATE FUNCTION fid_max()
RETURNS uuid LANGUAGE SQL IMMUTABLE PARALLEL SAFE STRICT
AS 'SELECT ''7fffffff-ffff-ffff-ffff-ffffffffffff''::uuid';

CREATE FUNCTION fid_min()
RETURNS uuid LANGUAGE SQL IMMUTABLE PARALLEL SAFE STRICT
AS 'SELECT ''00000000-0000-0000-0000-000000000000''::uuid';

-- Distance and index functions
CREATE FUNCTION fid_distance(uuid, uuid)
RETURNS int8 LANGUAGE SQL IMMUTABLE PARALLEL SAFE STRICT
AS 'SELECT (extract(epoch FROM fid_unpack_time($1) <-> fid_unpack_time($2)) * 1000000)::int8';

CREATE OPERATOR <-> (
	LEFTARG = uuid,
	RIGHTARG = uuid,
	PROCEDURE = fid_distance,
	COMMUTATOR = '<->'
);

CREATE FUNCTION fid_distance_op(internal, uuid, int2, oid, internal)
RETURNS float8 LANGUAGE C IMMUTABLE PARALLEL SAFE STRICT
AS 'tulip';

CREATE OPERATOR CLASS gist_fid_ops
FOR TYPE uuid USING gist AS
	OPERATOR 1  <,
	OPERATOR 2  <=,
	OPERATOR 3  =,
	OPERATOR 4  >=,
	OPERATOR 5  >,
	OPERATOR 6  <>,
	OPERATOR 15 <-> FOR ORDER BY pg_catalog.integer_ops,
	FUNCTION 1  gbt_uuid_consistent (internal, uuid, int2, oid, internal),
	FUNCTION 2  gbt_uuid_union (internal, internal),
	FUNCTION 3  gbt_uuid_compress (internal),
	FUNCTION 4  gbt_decompress (internal),
	FUNCTION 5  gbt_uuid_penalty (internal, internal, internal),
	FUNCTION 6  gbt_uuid_picksplit (internal, internal),
	FUNCTION 7  gbt_uuid_same (gbtreekey32, gbtreekey32, internal),
	FUNCTION 9  gbt_uuid_fetch (internal),
	FUNCTION 8  fid_distance_op(internal, uuid, int2, oid, internal),
	STORAGE gbtreekey32;

-- Ranking functions
-- This is used for GiST index sorting
CREATE FUNCTION fid_rank_desc(uuid)
RETURNS int8 LANGUAGE SQL IMMUTABLE PARALLEL SAFE STRICT
AS 'SELECT $1 <-> fid_max()';

CREATE FUNCTION fid_rank_asc(uuid)
RETURNS int8 LANGUAGE SQL IMMUTABLE PARALLEL SAFE STRICT
AS 'SELECT $1 <-> fid_min()';

-- Json helper functions
CREATE FUNCTION jsonb_unique(data jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE PARALLEL SAFE AS $$
BEGIN
	RETURN (SELECT jsonb_agg(DISTINCT value) FROM jsonb_array_elements(data));
END; $$;

-- Tick helper functions
CREATE FUNCTION tick_time_bucket(tick_first timestamptz, tick_length interval, "time" timestamptz)
RETURNS timestamptz LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
	RETURN time_bucket(tick_length, "time", origin => tick_first);
END; $$;

CREATE FUNCTION tick_number_bucket(tick_first timestamptz, tick_length interval, "time" timestamptz)
RETURNS int LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
	RETURN extract(epoch from (tick_time_bucket(tick_first, tick_length, "time") - tick_first)) / extract(epoch from tick_length);
END; $$;
