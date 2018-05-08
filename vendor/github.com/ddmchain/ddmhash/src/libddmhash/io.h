
#pragma once
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdio.h>
#ifdef __cplusplus
#define __STDC_FORMAT_MACROS 1
#endif
#include <inttypes.h>
#include "endian.h"
#include "ddmhash.h"

#ifdef __cplusplus
extern "C" {
#endif

#define DAG_MUTABLE_NAME_MAX_SIZE (6 + 10 + 1 + 16 + 1)

enum ddmhash_io_rc {
	DDMHASH_IO_FAIL = 0,           
	DDMHASH_IO_MEMO_SIZE_MISMATCH, 
	DDMHASH_IO_MEMO_MISMATCH,      
	DDMHASH_IO_MEMO_MATCH,         
};

#if defined(_WIN32) && !defined(__GNUC__)
#define snprintf(...) sprintf_s(__VA_ARGS__)
#endif

#ifdef DDMHASH_PRINT_CRITICAL_OUTPUT
#define DDMHASH_CRITICAL(...)							\
	do													\
	{													\
		printf("DDMHASH CRITICAL ERROR: "__VA_ARGS__);	\
		printf("\n");									\
		fflush(stdout);									\
	} while (0)
#else
#define DDMHASH_CRITICAL(...)
#endif

enum ddmhash_io_rc ddmhash_io_prepare(
	char const* dirname,
	ddmhash_h256_t const seedhash,
	FILE** output_file,
	uint64_t file_size,
	bool force_create
);

FILE* ddmhash_fopen(char const* file_name, char const* mode);

char* ddmhash_strncat(char* dest, size_t dest_size, char const* src, size_t count);

bool ddmhash_mkdir(char const* dirname);

bool ddmhash_file_size(FILE* f, size_t* ret_size);

int ddmhash_fileno(FILE* f);

char* ddmhash_io_create_filename(
	char const* dirname,
	char const* filename,
	size_t filename_length
);

bool ddmhash_get_default_dirname(char* strbuf, size_t buffsize);

static inline bool ddmhash_io_mutable_name(
	uint32_t revision,
	ddmhash_h256_t const* seed_hash,
	char* output
)
{
    uint64_t hash = *((uint64_t*)seed_hash);
#if LITTLE_ENDIAN == BYTE_ORDER
    hash = ddmhash_swap_u64(hash);
#endif
    return snprintf(output, DAG_MUTABLE_NAME_MAX_SIZE, "full-R%u-%016" PRIx64, revision, hash) >= 0;
}

#ifdef __cplusplus
}
#endif
