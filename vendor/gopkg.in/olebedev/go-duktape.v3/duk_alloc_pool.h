#if !defined(DUK_ALLOC_POOL_H_INCLUDED)
#define DUK_ALLOC_POOL_H_INCLUDED

#include "duktape.h"

#if defined(DUK_ALLOC_POOL_TRACK_WASTE)
#define DUK_ALLOC_POOL_WASTE_MARKER  0xedcb2345UL
#endif

#if defined(DUK_USE_ROM_OBJECTS) && defined(DUK_USE_HEAPPTR16)
#define DUK_ALLOC_POOL_ROMPTR_COMPRESSION
#define DUK_ALLOC_POOL_ROMPTR_FIRST DUK_USE_ROM_PTRCOMP_FIRST

extern const void * const duk_rom_compressed_pointers[];
#endif

typedef struct {
	unsigned int size;  
	unsigned int a;     
	unsigned int b;
} duk_pool_config;

struct duk_pool_free;
typedef struct duk_pool_free duk_pool_free;
struct duk_pool_free {
	duk_pool_free *next;
};

typedef struct {
	duk_pool_free *first;
	char *alloc_end;
	unsigned int size;
	unsigned int count;
#if defined(DUK_ALLOC_POOL_TRACK_HIGHWATER)
	unsigned int hwm_used_count;
#endif
} duk_pool_state;

typedef struct {
	size_t used_count;
	size_t used_bytes;
	size_t free_count;
	size_t free_bytes;
	size_t waste_bytes;
	size_t hwm_used_count;
} duk_pool_stats;

typedef struct {
	int num_pools;
	duk_pool_state *states;
#if defined(DUK_ALLOC_POOL_TRACK_HIGHWATER)
	size_t hwm_used_bytes;
	size_t hwm_waste_bytes;
#endif
} duk_pool_global;

typedef struct {
	size_t used_bytes;
	size_t free_bytes;
	size_t waste_bytes;
	size_t hwm_used_bytes;
	size_t hwm_waste_bytes;
} duk_pool_global_stats;

void *duk_alloc_pool_init(char *buffer,
                          size_t size,
                          const duk_pool_config *configs,
                          duk_pool_state *states,
                          int num_pools,
                          duk_pool_global *global);

void *duk_alloc_pool(void *udata, duk_size_t size);
void *duk_realloc_pool(void *udata, void *ptr, duk_size_t size);
void duk_free_pool(void *udata, void *ptr);

void duk_alloc_pool_get_pool_stats(duk_pool_state *s, duk_pool_stats *res);
void duk_alloc_pool_get_global_stats(duk_pool_global *g, duk_pool_global_stats *res);

#if defined(DUK_USE_ROM_OBJECTS) && defined(DUK_USE_HEAPPTR16)
extern const void *duk_alloc_pool_romptr_low;
extern const void *duk_alloc_pool_romptr_high;
duk_uint16_t duk_alloc_pool_enc16_rom(void *ptr);
#endif
#if defined(DUK_USE_HEAPPTR16)
extern void *duk_alloc_pool_ptrcomp_base;
#endif

#if 0
duk_uint16_t duk_alloc_pool_enc16(void *ptr);
void *duk_alloc_pool_dec16(duk_uint16_t val);
#endif

#if defined(DUK_ALWAYS_INLINE)
#define DUK__ALLOC_POOL_ALWAYS_INLINE DUK_ALWAYS_INLINE
#else
#define DUK__ALLOC_POOL_ALWAYS_INLINE 
#endif

#if defined(DUK_USE_HEAPPTR16)
static DUK__ALLOC_POOL_ALWAYS_INLINE duk_uint16_t duk_alloc_pool_enc16(void *ptr) {
	if (ptr == NULL) {

		return 0;
	}
#if defined(DUK_ALLOC_POOL_ROMPTR_COMPRESSION)
	if (ptr >= duk_alloc_pool_romptr_low && ptr <= duk_alloc_pool_romptr_high) {

		return duk_alloc_pool_enc16_rom(ptr);
	}
#endif
	return (duk_uint16_t) (((size_t) ((char *) ptr - (char *) duk_alloc_pool_ptrcomp_base)) >> 2);
}

static DUK__ALLOC_POOL_ALWAYS_INLINE void *duk_alloc_pool_dec16(duk_uint16_t val) {
	if (val == 0) {

		return NULL;
	}
#if defined(DUK_ALLOC_POOL_ROMPTR_COMPRESSION)
	if (val >= DUK_ALLOC_POOL_ROMPTR_FIRST) {

		return (void *) (intptr_t) (duk_rom_compressed_pointers[val - DUK_ALLOC_POOL_ROMPTR_FIRST]);
	}
#endif
	return (void *) ((char *) duk_alloc_pool_ptrcomp_base + (((size_t) val) << 2));
}
#endif

#endif  
