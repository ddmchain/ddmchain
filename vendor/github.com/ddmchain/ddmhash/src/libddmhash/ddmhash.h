/*
  This file is part of ddmhash.

  ddmhash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  ddmhash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with ddmhash.  If not, see <http://www.gnu.org/licenses/>.
*/

/** @file ddmhash.h
*/
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define DDMHASH_REVISION 23
#define DDMHASH_DATASET_BYTES_INIT 1073741824U // 2**30
#define DDMHASH_DATASET_BYTES_GROWTH 8388608U  // 2**23
#define DDMHASH_CACHE_BYTES_INIT 1073741824U // 2**24
#define DDMHASH_CACHE_BYTES_GROWTH 131072U  // 2**17
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

/// Type of a seedhash/blockhash e.t.c.
typedef struct ddmhash_h256 { uint8_t b[32]; } ddmhash_h256_t;

// convenience macro to statically initialize an h256_t
// usage:
// ddmhash_h256_t a = ddmhash_h256_static_init(1, 2, 3, ... )
// have to provide all 32 values. If you don't provide all the rest
// will simply be unitialized (not guranteed to be 0)
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

/**
 * Allocate and initialize a new ddmhash_light handler
 *
 * @param block_number   The block number for which to create the handler
 * @return               Newly allocated ddmhash_light handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref ddmhash_compute_cache_nodes()
 */
ddmhash_light_t ddmhash_light_new(uint64_t block_number);
/**
 * Frees a previously allocated ddmhash_light handler
 * @param light        The light handler to free
 */
void ddmhash_light_delete(ddmhash_light_t light);
/**
 * Calculate the light client data
 *
 * @param light          The light client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               an object of ddmhash_return_value_t holding the return values
 */
ddmhash_return_value_t ddmhash_light_compute(
	ddmhash_light_t light,
	ddmhash_h256_t const header_hash,
	uint64_t nonce
);

/**
 * Allocate and initialize a new ddmhash_full handler
 *
 * @param light         The light handler containing the cache.
 * @param callback      A callback function with signature of @ref ddmhash_callback_t
 *                      It accepts an unsigned with which a progress of DAG calculation
 *                      can be displayed. If all goes well the callback should return 0.
 *                      If a non-zero value is returned then DAG generation will stop.
 *                      Be advised. A progress value of 100 means that DAG creation is
 *                      almost complete and that this function will soon return succesfully.
 *                      It does not mean that the function has already had a succesfull return.
 * @return              Newly allocated ddmhash_full handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref ddmhash_compute_full_data()
 */
ddmhash_full_t ddmhash_full_new(ddmhash_light_t light, ddmhash_callback_t callback);

/**
 * Frees a previously allocated ddmhash_full handler
 * @param full    The light handler to free
 */
void ddmhash_full_delete(ddmhash_full_t full);
/**
 * Calculate the full client data
 *
 * @param full           The full client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               An object of ddmhash_return_value to hold the return value
 */
ddmhash_return_value_t ddmhash_full_compute(
	ddmhash_full_t full,
	ddmhash_h256_t const header_hash,
	uint64_t nonce
);
/**
 * Get a pointer to the full DAG data
 */
void const* ddmhash_full_dag(ddmhash_full_t full);
/**
 * Get the size of the DAG data
 */
uint64_t ddmhash_full_dag_size(ddmhash_full_t full);

/**
 * Calculate the seedhash for a given block number
 */
ddmhash_h256_t ddmhash_get_seedhash(uint64_t block_number);

#ifdef __cplusplus
}
#endif
