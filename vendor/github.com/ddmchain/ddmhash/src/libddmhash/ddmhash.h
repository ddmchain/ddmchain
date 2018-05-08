
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define DDMHASH_REVISION 23
#define DDMHASH_DATASET_BYTES_INIT 1073741824U 
#define DDMHASH_DATASET_BYTES_GROWTH 8388608U  
#define DDMHASH_CACHE_BYTES_INIT 1073741824U 
#define DDMHASH_CACHE_BYTES_GROWTH 131072U  
#define DDMHASH_EPOCH_LENGTH 30000U
#define DDMHASH_MIX_BYTES 128
#define DDMHASH_HASH_BYTES 64
#define DDMHASH_DATASET_PARENTS 256
#define DDMHASH_CACHE_ROUNDS 3
#define DDMHASH_ACCESSES 64
#define DDMHASH_DAG_MAGIC_NUM_SIZE 8
#define DDMHASH_DAG_MAGIC_NUM 0xFEE1DEADBADDCAFE

#ifdef __cplusplus
extern "C" {
#endif

typedef struct ddmhash_h256 { uint8_t b[32]; } ddmhash_h256_t;

#define ddmhash_h256_static_init(...)			\
	{ {__VA_ARGS__} }

struct ddmhash_light;
typedef struct ddmhash_light* ddmhash_light_t;
struct ddmhash_full;
typedef struct ddmhash_full* ddmhash_full_t;
typedef int(*ddmhash_callback_t)(unsigned);

typedef struct ddmhash_return_value {
	ddmhash_h256_t result;
	ddmhash_h256_t mix_hash;
	bool success;
} ddmhash_return_value_t;

ddmhash_light_t ddmhash_light_new(uint64_t block_number);

void ddmhash_light_delete(ddmhash_light_t light);

ddmhash_return_value_t ddmhash_light_compute(
	ddmhash_light_t light,
	ddmhash_h256_t const header_hash,
	uint64_t nonce
);

ddmhash_full_t ddmhash_full_new(ddmhash_light_t light, ddmhash_callback_t callback);

void ddmhash_full_delete(ddmhash_full_t full);

ddmhash_return_value_t ddmhash_full_compute(
	ddmhash_full_t full,
	ddmhash_h256_t const header_hash,
	uint64_t nonce
);

void const* ddmhash_full_dag(ddmhash_full_t full);

uint64_t ddmhash_full_dag_size(ddmhash_full_t full);

ddmhash_h256_t ddmhash_get_seedhash(uint64_t block_number);

#ifdef __cplusplus
}
#endif
