#include "postgres.h"
#include "access/gist.h"
#include "executor/executor.h"
#include "utils/uuid.h"

PG_MODULE_MAGIC;

PG_FUNCTION_INFO_V1(fid_distance_op);

typedef struct {
	pg_uuid_t lower, upper;
} uuidKEY;

uint64_t fid_unpack_uint(pg_uuid_t* uuid);
Datum fid_distance_op(PG_FUNCTION_ARGS);

uint64_t fid_unpack_uint(pg_uuid_t* uuid) {
	return
		((uint64_t)uuid->data[0] << 56) |
		((uint64_t)uuid->data[1] << 48) |
		((uint64_t)uuid->data[2] << 40) |
		((uint64_t)uuid->data[3] << 32) |
		((uint64_t)uuid->data[4] << 24) |
		((uint64_t)uuid->data[5] << 16) |
		((uint64_t)uuid->data[6] << 8) |
		(uint64_t)uuid->data[7];
}

Datum fid_distance_op(PG_FUNCTION_ARGS) {
	GISTENTRY* entry = (GISTENTRY *) PG_GETARG_POINTER(0);
	pg_uuid_t* uuid = PG_GETARG_UUID_P(1);
	uuidKEY* key = (uuidKEY *)DatumGetPointer(entry->key);

	uint64_t query = fid_unpack_uint(uuid);
	uint64_t lower = fid_unpack_uint(&key->lower);
	uint64_t upper = fid_unpack_uint(&key->upper);

	if(query <= lower)
		PG_RETURN_FLOAT8((float8)(lower - query));
	if(query >= upper)
		PG_RETURN_FLOAT8((float8)(query - upper));
	PG_RETURN_FLOAT8(0);
}
