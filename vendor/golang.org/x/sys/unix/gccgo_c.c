
// +build gccgo

#include <errno.h>
#include <stdint.h>
#include <unistd.h>

#define _STRINGIFY2_(x) #x
#define _STRINGIFY_(x) _STRINGIFY2_(x)
#define GOSYM_PREFIX _STRINGIFY_(__USER_LABEL_PREFIX__)

struct ret {
	uintptr_t r;
	uintptr_t err;
};

struct ret
gccgoRealSyscall(uintptr_t trap, uintptr_t a1, uintptr_t a2, uintptr_t a3, uintptr_t a4, uintptr_t a5, uintptr_t a6, uintptr_t a7, uintptr_t a8, uintptr_t a9)
{
	struct ret r;

	errno = 0;
	r.r = syscall(trap, a1, a2, a3, a4, a5, a6, a7, a8, a9);
	r.err = errno;
	return r;
}

extern void use(void *) __asm__ (GOSYM_PREFIX GOPKGPATH ".use") __attribute__((noinline));

void
use(void *p __attribute__ ((unused)))
{
}
