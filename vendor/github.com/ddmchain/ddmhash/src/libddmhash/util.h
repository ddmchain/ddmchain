
#pragma once
#include <stdint.h>
#include "compiler.h"

#ifdef __cplusplus
extern "C" {
#endif

#ifdef _MSC_VER
void debugf(char const* str, ...);
#else
#define debugf printf
#endif

static inline uint32_t min_u32(uint32_t a, uint32_t b)
{
	return a < b ? a : b;
}

static inline uint32_t clamp_u32(uint32_t x, uint32_t min_, uint32_t max_)
{
	return x < min_ ? min_ : (x > max_ ? max_ : x);
}

#ifdef __cplusplus
}
#endif
