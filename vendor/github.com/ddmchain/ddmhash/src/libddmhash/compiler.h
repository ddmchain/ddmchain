
#pragma once

#if defined(_MSC_VER) && !defined(__cplusplus)
#define inline __inline
#endif

#if defined(_MSC_VER)
#define restrict __restrict
#else
#define restrict __restrict__
#endif

