#pragma once
#include "compiler.h"
#include "endian.h"
#include "ddmhash.h"
#include <stdio.h>

#define ENABLE_SSE 0

#if defined(_M_X64) && ENABLE_SSE
#include <smmintrin.h>
#endif

#ifdef __cplusplus
extern "C" {
#endif

#define NODE_WORDS (64/4)
#define MIX_WORDS (DDMHASH_MIX_BYTES/4)
#define MIX_NODES (MIX_WORDS / NODE_WORDS)
#include <stdint.h>

typedef union node {
	uint8_t bytes[NODE_WORDS * 4];
	uint32_t words[NODE_WORDS];
	uint64_t double_words[NODE_WORDS / 2];

#if defined(_M_X64) && ENABLE_SSE
	__m128i xmm[NODE_WORDS/4];
#endif

} node;

static inline uint8_t ddmhash_h256_get(ddmhash_h256_t const* hash, unsigned int i)
{
	return hash->b[i];
}

static inline void ddmhash_h256_set(ddmhash_h256_t* hash, unsigned int i, uint8_t v)
{
	hash->b[i] = v;
}

static inline void ddmhash_h256_reset(ddmhash_h256_t* hash)
{
	memset(hash, 0, 32);
}

static inline bool ddmhash_check_difficulty(
	ddmhash_h256_t const* hash,
	ddmhash_h256_t const* boundary
)
{

	for (int i = 0; i < 32; i++) {
		if (ddmhash_h256_get(hash, i) == ddmhash_h256_get(boundary, i)) {
			continue;
		}
		return ddmhash_h256_get(hash, i) < ddmhash_h256_get(boundary, i);
	}
	return true;
}

bool ddmhash_quick_check_difficulty(
	ddmhash_h256_t const* header_hash,
	uint64_t const nonce,
	ddmhash_h256_t const* mix_hash,
	ddmhash_h256_t const* boundary
);

struct ddmhash_light {
	void* cache;
	uint64_t cache_size;
	uint64_t block_number;
};

ddmhash_light_t ddmhash_light_new_internal(uint64_t cache_size, ddmhash_h256_t const* seed);

ddmhash_return_value_t ddmhash_light_compute_internal(
	ddmhash_light_t light,
	uint64_t full_size,
	ddmhash_h256_t const header_hash,
	uint64_t nonce
);

struct ddmhash_full {
	FILE* file;
	uint64_t file_size;
	node* data;
};

ddmhash_full_t ddmhash_full_new_internal(
	char const* dirname,
	ddmhash_h256_t const seed_hash,
	uint64_t full_size,
	ddmhash_light_t const light,
	ddmhash_callback_t callback
);

void ddmhash_calculate_dag_item(
	node* const ret,
	uint32_t node_index,
	ddmhash_light_t const cache
);

void ddmhash_quick_hash(
	ddmhash_h256_t* return_hash,
	ddmhash_h256_t const* header_hash,
	const uint64_t nonce,
	ddmhash_h256_t const* mix_hash
);

uint64_t ddmhash_get_datasize(uint64_t const block_number);
uint64_t ddmhash_get_cachesize(uint64_t const block_number);

bool ddmhash_compute_full_data(
	void* mem,
	uint64_t full_size,
	ddmhash_light_t const light,
	ddmhash_callback_t callback
);

#ifdef __cplusplus
}
#endif
