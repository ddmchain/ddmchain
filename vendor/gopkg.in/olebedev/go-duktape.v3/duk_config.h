
#if !defined(DUK_CONFIG_H_INCLUDED)
#define DUK_CONFIG_H_INCLUDED

#undef DUK_F_DLL_BUILD

#if defined(__APPLE__)
#define DUK_F_APPLE
#endif

#if defined(__FreeBSD__) || defined(__FreeBSD)
#define DUK_F_FREEBSD
#endif

#if defined(DUK_F_FREEBSD) && defined(__ORBIS__)
#define DUK_F_ORBIS
#endif

#if defined(__OpenBSD__) || defined(__OpenBSD)
#define DUK_F_OPENBSD
#endif

#if defined(__NetBSD__) || defined(__NetBSD)
#define DUK_F_NETBSD
#endif

#if defined(DUK_F_FREEBSD) || defined(DUK_F_NETBSD) || defined(DUK_F_OPENBSD) || \
    defined(__bsdi__) || defined(__DragonFly__)
#define DUK_F_BSD
#endif

#if defined(__TOS__)
#define DUK_F_TOS
#endif

#if defined(__m68k__) || defined(M68000) || defined(__MC68K__)
#define DUK_F_M68K
#endif

#if defined(AMIGA) || defined(__amigaos__)
#define DUK_F_AMIGAOS
#endif

#if defined(__powerpc) || defined(__powerpc__) || defined(__PPC__)
#define DUK_F_PPC
#if defined(__PPC64__) || defined(__LP64__) || defined(_LP64)
#define DUK_F_PPC64
#else
#define DUK_F_PPC32
#endif
#endif

#if defined(_DURANGO) || defined(_XBOX_ONE)
#define DUK_F_DURANGO
#endif

#if defined(_WIN32) || defined(WIN32) || defined(_WIN64) || defined(WIN64) || \
    defined(__WIN32__) || defined(__TOS_WIN__) || defined(__WINDOWS__)
#define DUK_F_WINDOWS
#if defined(_WIN64) || defined(WIN64)
#define DUK_F_WIN64
#else
#define DUK_F_WIN32
#endif
#endif

#if defined(__FLASHPLAYER__)
#define DUK_F_FLASHPLAYER
#endif

#if defined(__QNX__)
#define DUK_F_QNX
#endif

#if defined(_TINSPIRE)
#define DUK_F_TINSPIRE
#endif

#if defined(EMSCRIPTEN)
#define DUK_F_EMSCRIPTEN
#endif

#if defined(__BCC__) || defined(__BCC_VERSION__)
#define DUK_F_BCC
#endif

#if defined(__linux) || defined(__linux__) || defined(linux)
#define DUK_F_LINUX
#endif

#if defined(__sun) && defined(__SVR4)
#define DUK_F_SUN
#if defined(__SUNPRO_C) && (__SUNPRO_C < 0x550)
#define DUK_F_OLD_SOLARIS

#include <sys/isa_defs.h>
#endif
#endif

#if defined(_AIX)

#define DUK_F_AIX
#endif

#if defined(__hpux)
#define DUK_F_HPUX
#if defined(__ia64)
#define DUK_F_HPUX_ITANIUM
#endif
#endif

#if defined(__posix)
#define DUK_F_POSIX
#endif

#if defined(__CYGWIN__)
#define DUK_F_CYGWIN
#endif

#if defined(__unix) || defined(__unix__) || defined(unix) || \
    defined(DUK_F_LINUX) || defined(DUK_F_BSD)
#define DUK_F_UNIX
#endif

#undef DUK_F_CPP
#if defined(__cplusplus)
#define DUK_F_CPP
#endif

#if defined(__amd64__) || defined(__amd64) || \
    defined(__x86_64__) || defined(__x86_64) || \
    defined(_M_X64) || defined(_M_AMD64)
#if defined(__ILP32__) || defined(_ILP32)
#define DUK_F_X32
#else
#define DUK_F_X64
#endif
#elif defined(i386) || defined(__i386) || defined(__i386__) || \
      defined(__i486__) || defined(__i586__) || defined(__i686__) || \
      defined(__IA32__) || defined(_M_IX86) || defined(__X86__) || \
      defined(_X86_) || defined(__THW_INTEL__) || defined(__I86__)
#if defined(__LP64__) || defined(_LP64)

#define DUK_F_X64
#else
#define DUK_F_X86
#endif
#endif

#if defined(__arm__) || defined(__thumb__) || defined(_ARM) || defined(_M_ARM) || defined(__aarch64__)
#define DUK_F_ARM
#if defined(__LP64__) || defined(_LP64) || defined(__arm64) || defined(__arm64__) || defined(__aarch64__)
#define DUK_F_ARM64
#else
#define DUK_F_ARM32
#endif
#endif

#if defined(__mips__) || defined(mips) || defined(_MIPS_ISA) || \
    defined(_R3000) || defined(_R4000) || defined(_R5900) || \
    defined(_MIPS_ISA_MIPS1) || defined(_MIPS_ISA_MIPS2) || \
    defined(_MIPS_ISA_MIPS3) || defined(_MIPS_ISA_MIPS4) || \
    defined(__mips) || defined(__MIPS__)
#define DUK_F_MIPS
#if defined(__LP64__) || defined(_LP64) || defined(__mips64) || \
    defined(__mips64__) || defined(__mips_n64)
#define DUK_F_MIPS64
#else
#define DUK_F_MIPS32
#endif
#endif

#if defined(sparc) || defined(__sparc) || defined(__sparc__)
#define DUK_F_SPARC
#if defined(__LP64__) || defined(_LP64)
#define DUK_F_SPARC64
#else
#define DUK_F_SPARC32
#endif
#endif

#if defined(__sh__) || \
    defined(__sh1__) || defined(__SH1__) || \
    defined(__sh2__) || defined(__SH2__) || \
    defined(__sh3__) || defined(__SH3__) || \
    defined(__sh4__) || defined(__SH4__) || \
    defined(__sh5__) || defined(__SH5__)
#define DUK_F_SUPERH
#endif

#if defined(__clang__)
#define DUK_F_CLANG
#endif

#undef DUK_F_C99
#if defined(__STDC_VERSION__) && (__STDC_VERSION__ >= 199901L)
#define DUK_F_C99
#endif

#undef DUK_F_CPP11
#if defined(__cplusplus) && (__cplusplus >= 201103L)
#define DUK_F_CPP11
#endif

#if defined(__GNUC__) && !defined(__clang__) && !defined(DUK_F_CLANG)
#define DUK_F_GCC
#if defined(__GNUC__) && defined(__GNUC_MINOR__) && defined(__GNUC_PATCHLEVEL__)

#define DUK_F_GCC_VERSION  (__GNUC__ * 10000L + __GNUC_MINOR__ * 100L + __GNUC_PATCHLEVEL__)
#else
#error cannot figure out gcc version
#endif
#endif

#if defined(__MINGW32__) || defined(__MINGW64__)
#define DUK_F_MINGW
#endif

#if defined(_MSC_VER)

#define DUK_F_MSVC
#if defined(_MSC_FULL_VER)
#if (_MSC_FULL_VER > 100000000)
#define DUK_F_MSVC_FULL_VER _MSC_FULL_VER
#else
#define DUK_F_MSCV_FULL_VER (_MSC_FULL_VER * 10)
#endif
#endif
#endif  

#if defined(__TINYC__)

#define DUK_F_TINYC
#endif

#if defined(__VBCC__)
#define DUK_F_VBCC
#endif

#if defined(ANDROID) || defined(__ANDROID__)
#define DUK_F_ANDROID
#endif

#if defined(__MINT__)
#define DUK_F_MINT
#endif

#if defined(__cplusplus) && !defined(__STDC_LIMIT_MACROS)
#define __STDC_LIMIT_MACROS
#endif
#if defined(__cplusplus) && !defined(__STDC_CONSTANT_MACROS)
#define __STDC_CONSTANT_MACROS
#endif

#if defined(DUK_F_APPLE)

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <TargetConditionals.h>
#include <architecture/byte_order.h>
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#if TARGET_IPHONE_SIMULATOR
#define DUK_USE_OS_STRING "iphone-sim"
#elif TARGET_OS_IPHONE
#define DUK_USE_OS_STRING "iphone"
#elif TARGET_OS_MAC
#define DUK_USE_OS_STRING "osx"
#else
#define DUK_USE_OS_STRING "osx-unknown"
#endif

#define DUK_JMPBUF_TYPE       jmp_buf
#define DUK_SETJMP(jb)        _setjmp((jb))
#define DUK_LONGJMP(jb)       _longjmp((jb), 1)
#elif defined(DUK_F_ORBIS)

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_S

#define DUK_USE_DATE_FMT_STRFTIME
#include <sys/types.h>
#include <machine/endian.h>
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING  "orbis"
#elif defined(DUK_F_OPENBSD)

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <sys/types.h>
#include <sys/endian.h>
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING  "openbsd"
#elif defined(DUK_F_BSD)

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <sys/types.h>
#include <sys/endian.h>
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING  "bsd"
#elif defined(DUK_F_TOS)

#define DUK_USE_DATE_NOW_TIME
#define DUK_USE_DATE_TZO_GMTIME

#define DUK_USE_DATE_FMT_STRFTIME
#include <time.h>

#define DUK_USE_OS_STRING  "tos"

#if !defined(DUK_USE_BYTEORDER) && defined(DUK_F_M68K)
#define DUK_USE_BYTEORDER 3
#endif
#elif defined(DUK_F_AMIGAOS)

#if defined(DUK_F_M68K)

#define DUK_USE_DATE_NOW_TIME
#define DUK_USE_DATE_TZO_GMTIME

#define DUK_USE_DATE_FMT_STRFTIME
#include <time.h>
#elif defined(DUK_F_PPC)
#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <time.h>
#if !defined(UINTPTR_MAX)
#define UINTPTR_MAX UINT_MAX
#endif
#else
#error AmigaOS but not M68K/PPC, not supported now
#endif

#define DUK_USE_OS_STRING "amigaos"

#if !defined(DUK_USE_BYTEORDER) && (defined(DUK_F_M68K) || defined(DUK_F_PPC))
#define DUK_USE_BYTEORDER 3
#endif
#elif defined(DUK_F_DURANGO)

#if defined(DUK_COMPILING_DUKTAPE) && !defined(_CRT_SECURE_NO_WARNINGS)
#define _CRT_SECURE_NO_WARNINGS
#endif

#define DUK_USE_DATE_NOW_WINDOWS
#define DUK_USE_DATE_TZO_WINDOWS_NO_DST

#if defined(DUK_COMPILING_DUKTAPE)

#include <windows.h>
#endif

#define DUK_USE_OS_STRING "durango"

#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 1
#endif
#elif defined(DUK_F_WINDOWS)

#if defined(DUK_COMPILING_DUKTAPE) && !defined(_CRT_SECURE_NO_WARNINGS)
#define _CRT_SECURE_NO_WARNINGS
#endif

#if defined(DUK_COMPILING_DUKTAPE)

#include <windows.h>
#endif

#if defined(DUK_USE_DATE_NOW_WINDOWS_SUBMS) || defined(DUK_USE_DATE_NOW_WINDOWS)

#else
#if defined(_WIN32_WINNT) && (_WIN32_WINNT >= 0x0602)
#define DUK_USE_DATE_NOW_WINDOWS_SUBMS
#else
#define DUK_USE_DATE_NOW_WINDOWS
#endif
#endif

#define DUK_USE_DATE_TZO_WINDOWS

#if !defined(DUK_USE_GET_MONOTONIC_TIME_WINDOWS_QPC) && \
    defined(_WIN32_WINNT) && (_WIN32_WINNT >= 0x0600)
#define DUK_USE_GET_MONOTONIC_TIME_WINDOWS_QPC
#endif

#define DUK_USE_OS_STRING "windows"

#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 1
#endif
#elif defined(DUK_F_FLASHPLAYER)

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <endian.h>
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING "flashplayer"

#if !defined(DUK_USE_BYTEORDER) && defined(DUK_F_FLASHPLAYER)
#define DUK_USE_BYTEORDER 1
#endif
#elif defined(DUK_F_QNX)

#if defined(DUK_F_QNX) && defined(DUK_COMPILING_DUKTAPE)

#define _XOPEN_SOURCE    600
#define _POSIX_C_SOURCE  200112L
#endif

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <sys/types.h>
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING "qnx"
#elif defined(DUK_F_TINSPIRE)

#if defined(DUK_COMPILING_DUKTAPE) && !defined(_XOPEN_SOURCE)
#define _XOPEN_SOURCE    
#endif

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <sys/types.h>
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING "tinspire"
#elif defined(DUK_F_EMSCRIPTEN)

#if defined(DUK_COMPILING_DUKTAPE)
#if !defined(_POSIX_C_SOURCE)
#define _POSIX_C_SOURCE  200809L
#endif
#if !defined(_GNU_SOURCE)
#define _GNU_SOURCE      
#endif
#if !defined(_XOPEN_SOURCE)
#define _XOPEN_SOURCE    
#endif
#endif  

#include <sys/types.h>
#if defined(DUK_F_BCC)

#else
#include <endian.h>
#endif  
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>
#include <stdint.h>

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME

#define DUK_USE_OS_STRING "emscripten"
#elif defined(DUK_F_LINUX)

#if defined(DUK_COMPILING_DUKTAPE)
#if !defined(_POSIX_C_SOURCE)
#define _POSIX_C_SOURCE  200809L
#endif
#if !defined(_GNU_SOURCE)
#define _GNU_SOURCE      
#endif
#if !defined(_XOPEN_SOURCE)
#define _XOPEN_SOURCE    
#endif
#endif  

#include <sys/types.h>
#if defined(DUK_F_BCC)

#else
#include <endian.h>
#include <stdint.h>
#endif  
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME

#if 0  
#define DUK_USE_GET_MONOTONIC_TIME_CLOCK_GETTIME
#endif

#define DUK_USE_OS_STRING "linux"
#elif defined(DUK_F_SUN)

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME

#include <sys/types.h>
#if defined(DUK_F_OLD_SOLARIS)

#define DUK_F_NO_STDINT_H
#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 3
#endif
#else  
#include <ast/endian.h>
#endif  

#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING "solaris"
#elif defined(DUK_F_AIX)

#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 3
#endif
#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING "aix"
#elif defined(DUK_F_HPUX)

#define DUK_F_NO_STDINT_H
#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 3
#endif
#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING "hpux"
#elif defined(DUK_F_POSIX)

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <sys/types.h>
#include <endian.h>
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_USE_OS_STRING "posix"
#elif defined(DUK_F_CYGWIN)

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_FMT_STRFTIME
#include <sys/types.h>
#include <endian.h>
#include <sys/param.h>
#include <sys/time.h>
#include <time.h>

#define DUK_JMPBUF_TYPE       jmp_buf
#define DUK_SETJMP(jb)        _setjmp((jb))
#define DUK_LONGJMP(jb)       _longjmp((jb), 1)

#define DUK_USE_OS_STRING "windows"
#elif defined(DUK_F_UNIX)

#define DUK_USE_DATE_NOW_GETTIMEOFDAY
#define DUK_USE_DATE_TZO_GMTIME_R
#define DUK_USE_DATE_PRS_STRPTIME
#define DUK_USE_DATE_FMT_STRFTIME
#include <time.h>
#include <sys/time.h>
#define DUK_USE_OS_STRING "unknown"
#else

#define DUK_USE_DATE_NOW_TIME

#define DUK_USE_DATE_TZO_GMTIME

#undef DUK_USE_DATE_PRS_STRPTIME
#undef DUK_USE_DATE_FMT_STRFTIME

#include <time.h>

#define DUK_USE_OS_STRING "unknown"
#endif  

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdarg.h>  
#include <setjmp.h>
#include <stddef.h>  
#include <math.h>
#include <limits.h>

#if defined(DUK_F_NO_STDINT_H)

#else

#include <stdint.h>
#endif

#if defined(DUK_F_CPP)
#include <exception>  
#endif

#if defined(DUK_F_X86)

#define DUK_USE_ARCH_STRING "x86"
#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 1
#endif

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 1
#endif
#define DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_X64)

#define DUK_USE_ARCH_STRING "x64"
#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 1
#endif

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 1
#endif
#undef DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_X32)

#define DUK_USE_ARCH_STRING "x32"
#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 1
#endif

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 1
#endif
#define DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_ARM32)

#define DUK_USE_ARCH_STRING "arm32"

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 4
#endif
#define DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_ARM64)

#define DUK_USE_ARCH_STRING "arm64"

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif
#undef DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_MIPS32)

#define DUK_USE_ARCH_STRING "mips32"

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif
#define DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_MIPS64)

#define DUK_USE_ARCH_STRING "mips64"

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif
#undef DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_PPC32)

#define DUK_USE_ARCH_STRING "ppc32"
#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 3
#endif
#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif
#define DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_PPC64)

#define DUK_USE_ARCH_STRING "ppc64"
#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 3
#endif
#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif
#undef DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_SPARC32)

#define DUK_USE_ARCH_STRING "sparc32"

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif
#define DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_SPARC64)

#define DUK_USE_ARCH_STRING "sparc64"

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif
#undef DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_SUPERH)

#define DUK_USE_ARCH_STRING "sh"

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 4
#endif
#define DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_M68K)

#define DUK_USE_ARCH_STRING "m68k"
#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 3
#endif
#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif
#define DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#elif defined(DUK_F_EMSCRIPTEN)

#define DUK_USE_ARCH_STRING "emscripten"
#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 1
#endif
#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif
#undef DUK_USE_PACKED_TVAL
#define DUK_F_PACKED_TVAL_PROVIDED
#else

#define DUK_USE_ARCH_STRING "generic"

#endif  

#if defined(DUK_F_CLANG)

#if defined(DUK_F_C99) || defined(DUK_F_CPP11)

#define DUK_VA_COPY(dest,src) va_copy(dest,src)
#else

#define DUK_VA_COPY(dest,src) __va_copy(dest,src)
#endif

#define DUK_NORETURN(decl)  decl __attribute__((noreturn))

#if defined(__clang__) && defined(__has_builtin)
#if __has_builtin(__builtin_unreachable)
#define DUK_UNREACHABLE()  do { __builtin_unreachable(); } while (0)
#endif
#endif

#define DUK_USE_BRANCH_HINTS
#define DUK_LIKELY(x)    __builtin_expect((x), 1)
#define DUK_UNLIKELY(x)  __builtin_expect((x), 0)
#if defined(__clang__) && defined(__has_builtin)
#if __has_builtin(__builtin_unpredictable)
#define DUK_UNPREDICTABLE(x)  __builtin_unpredictable((x))
#endif
#endif

#if defined(DUK_F_C99) || defined(DUK_F_CPP11)
#define DUK_NOINLINE        __attribute__((noinline))
#define DUK_INLINE          inline
#define DUK_ALWAYS_INLINE   inline __attribute__((always_inline))
#endif

#if defined(DUK_F_DLL_BUILD) && defined(DUK_F_WINDOWS)

#if defined(DUK_COMPILING_DUKTAPE)
#define DUK_EXTERNAL_DECL  extern __declspec(dllexport)
#define DUK_EXTERNAL       __declspec(dllexport)
#else
#define DUK_EXTERNAL_DECL  extern __declspec(dllimport)
#define DUK_EXTERNAL       should_not_happen
#endif
#if defined(DUK_SINGLE_FILE)
#define DUK_INTERNAL_DECL  static
#define DUK_INTERNAL       static
#else
#define DUK_INTERNAL_DECL  extern
#define DUK_INTERNAL       
#endif
#define DUK_LOCAL_DECL     static
#define DUK_LOCAL          static
#else
#define DUK_EXTERNAL_DECL  __attribute__ ((visibility("default"))) extern
#define DUK_EXTERNAL       __attribute__ ((visibility("default")))
#if defined(DUK_SINGLE_FILE)
#if (defined(DUK_F_GCC_VERSION) && DUK_F_GCC_VERSION >= 30101) || defined(DUK_F_CLANG)

#define DUK_INTERNAL_DECL  static __attribute__ ((unused))
#define DUK_INTERNAL       static __attribute__ ((unused))
#else
#define DUK_INTERNAL_DECL  static
#define DUK_INTERNAL       static
#endif
#else
#if (defined(DUK_F_GCC_VERSION) && DUK_F_GCC_VERSION >= 30101) || defined(DUK_F_CLANG)
#define DUK_INTERNAL_DECL  __attribute__ ((visibility("hidden"))) __attribute__ ((unused)) extern
#define DUK_INTERNAL       __attribute__ ((visibility("hidden"))) __attribute__ ((unused))
#else
#define DUK_INTERNAL_DECL  __attribute__ ((visibility("hidden"))) extern
#define DUK_INTERNAL       __attribute__ ((visibility("hidden")))
#endif
#endif
#define DUK_LOCAL_DECL     static
#define DUK_LOCAL          static
#endif

#if defined(DUK_F_CPP)
#define DUK_USE_COMPILER_STRING "clang"
#else
#define DUK_USE_COMPILER_STRING "clang"
#endif

#undef DUK_USE_VARIADIC_MACROS
#if defined(DUK_F_C99) || defined(DUK_F_CPP11)
#define DUK_USE_VARIADIC_MACROS
#endif

#define DUK_USE_UNION_INITIALIZERS

#undef DUK_USE_FLEX_C99
#undef DUK_USE_FLEX_ZEROSIZE
#undef DUK_USE_FLEX_ONESIZE
#if defined(DUK_F_C99)
#define DUK_USE_FLEX_C99
#else
#define DUK_USE_FLEX_ZEROSIZE
#endif

#undef DUK_USE_GCC_PRAGMAS
#define DUK_USE_PACK_CLANG_ATTR
#elif defined(DUK_F_GCC)

#if defined(DUK_F_C99) || defined(DUK_F_CPP11)

#define DUK_VA_COPY(dest,src) va_copy(dest,src)
#else

#define DUK_VA_COPY(dest,src) __va_copy(dest,src)
#endif

#if defined(DUK_F_GCC_VERSION) && (DUK_F_GCC_VERSION >= 20500L)

#define DUK_NORETURN(decl)  decl __attribute__((noreturn))
#endif

#if defined(DUK_F_GCC_VERSION) && (DUK_F_GCC_VERSION >= 40500L)

#define DUK_UNREACHABLE()  do { __builtin_unreachable(); } while (0)
#endif

#define DUK_USE_BRANCH_HINTS
#if defined(DUK_F_GCC_VERSION) && (DUK_F_GCC_VERSION >= 40500L)

#define DUK_LIKELY(x)    __builtin_expect((x), 1)
#define DUK_UNLIKELY(x)  __builtin_expect((x), 0)
#endif

#if (defined(DUK_F_C99) || defined(DUK_F_CPP11)) && \
    defined(DUK_F_GCC_VERSION) && (DUK_F_GCC_VERSION >= 30101)
#define DUK_NOINLINE        __attribute__((noinline))
#define DUK_INLINE          inline
#define DUK_ALWAYS_INLINE   inline __attribute__((always_inline))
#endif

#if (defined(DUK_F_C99) || defined(DUK_F_CPP11)) && \
    defined(DUK_F_GCC_VERSION) && (DUK_F_GCC_VERSION >= 40300)
#define DUK_HOT             __attribute__((hot))
#define DUK_COLD            __attribute__((cold))
#endif

#if defined(DUK_F_DLL_BUILD) && defined(DUK_F_WINDOWS)

#if defined(DUK_COMPILING_DUKTAPE)
#define DUK_EXTERNAL_DECL  extern __declspec(dllexport)
#define DUK_EXTERNAL       __declspec(dllexport)
#else
#define DUK_EXTERNAL_DECL  extern __declspec(dllimport)
#define DUK_EXTERNAL       should_not_happen
#endif
#if defined(DUK_SINGLE_FILE)
#define DUK_INTERNAL_DECL  static
#define DUK_INTERNAL       static
#else
#define DUK_INTERNAL_DECL  extern
#define DUK_INTERNAL       
#endif
#define DUK_LOCAL_DECL     static
#define DUK_LOCAL          static
#elif defined(DUK_F_GCC_VERSION) && (DUK_F_GCC_VERSION >= 40000)
#define DUK_EXTERNAL_DECL  __attribute__ ((visibility("default"))) extern
#define DUK_EXTERNAL       __attribute__ ((visibility("default")))
#if defined(DUK_SINGLE_FILE)
#if (defined(DUK_F_GCC_VERSION) && DUK_F_GCC_VERSION >= 30101) || defined(DUK_F_CLANG)

#define DUK_INTERNAL_DECL  static __attribute__ ((unused))
#define DUK_INTERNAL       static __attribute__ ((unused))
#else
#define DUK_INTERNAL_DECL  static
#define DUK_INTERNAL       static
#endif
#else
#if (defined(DUK_F_GCC_VERSION) && DUK_F_GCC_VERSION >= 30101) || defined(DUK_F_CLANG)
#define DUK_INTERNAL_DECL  __attribute__ ((visibility("hidden"))) __attribute__ ((unused)) extern
#define DUK_INTERNAL       __attribute__ ((visibility("hidden"))) __attribute__ ((unused))
#else
#define DUK_INTERNAL_DECL  __attribute__ ((visibility("hidden"))) extern
#define DUK_INTERNAL       __attribute__ ((visibility("hidden")))
#endif
#endif
#define DUK_LOCAL_DECL     static
#define DUK_LOCAL          static
#endif

#if defined(DUK_F_MINGW)
#if defined(DUK_F_CPP)
#define DUK_USE_COMPILER_STRING "mingw++"
#else
#define DUK_USE_COMPILER_STRING "mingw"
#endif
#else
#if defined(DUK_F_CPP)
#define DUK_USE_COMPILER_STRING "g++"
#else
#define DUK_USE_COMPILER_STRING "gcc"
#endif
#endif

#undef DUK_USE_VARIADIC_MACROS
#if defined(DUK_F_C99) || (defined(DUK_F_CPP11) && defined(__GNUC__))
#define DUK_USE_VARIADIC_MACROS
#endif

#define DUK_USE_UNION_INITIALIZERS

#undef DUK_USE_FLEX_C99
#undef DUK_USE_FLEX_ZEROSIZE
#undef DUK_USE_FLEX_ONESIZE
#if defined(DUK_F_C99)
#define DUK_USE_FLEX_C99
#else
#define DUK_USE_FLEX_ZEROSIZE
#endif

#if defined(DUK_F_GCC_VERSION) && (DUK_F_GCC_VERSION >= 40600)
#define DUK_USE_GCC_PRAGMAS
#else
#undef DUK_USE_GCC_PRAGMAS
#endif

#define DUK_USE_PACK_GCC_ATTR
#elif defined(DUK_F_MSVC)

#define DUK_NORETURN(decl)  __declspec(noreturn) decl

#undef DUK_USE_BRANCH_HINTS

#if defined(DUK_F_DLL_BUILD) && defined(DUK_F_WINDOWS)

#if defined(DUK_COMPILING_DUKTAPE)
#define DUK_EXTERNAL_DECL  extern __declspec(dllexport)
#define DUK_EXTERNAL       __declspec(dllexport)
#else
#define DUK_EXTERNAL_DECL  extern __declspec(dllimport)
#define DUK_EXTERNAL       should_not_happen
#endif
#if defined(DUK_SINGLE_FILE)
#define DUK_INTERNAL_DECL  static
#define DUK_INTERNAL       static
#else
#define DUK_INTERNAL_DECL  extern
#define DUK_INTERNAL       
#endif
#define DUK_LOCAL_DECL     static
#define DUK_LOCAL          static
#endif

#if defined(DUK_F_CPP)
#define DUK_USE_COMPILER_STRING "msvc++"
#else
#define DUK_USE_COMPILER_STRING "msvc"
#endif

#undef DUK_USE_VARIADIC_MACROS
#if defined(DUK_F_C99)
#define DUK_USE_VARIADIC_MACROS
#elif defined(_MSC_VER) && (_MSC_VER >= 1400)

#define DUK_USE_VARIADIC_MACROS
#endif

#undef DUK_USE_UNION_INITIALIZERS
#if defined(_MSC_VER) && (_MSC_VER >= 1800)

#define DUK_USE_UNION_INITIALIZERS
#endif

#undef DUK_USE_FLEX_C99
#undef DUK_USE_FLEX_ZEROSIZE
#undef DUK_USE_FLEX_ONESIZE
#if defined(DUK_F_C99)
#define DUK_USE_FLEX_C99
#else
#define DUK_USE_FLEX_ZEROSIZE
#endif

#undef DUK_USE_GCC_PRAGMAS

#define DUK_USE_PACK_MSVC_PRAGMA

#if defined(_MSC_VER) && (_MSC_VER >= 1500)
#define DUK_NOINLINE        __declspec(noinline)
#define DUK_INLINE          __inline
#define DUK_ALWAYS_INLINE   __forceinline
#endif

#if defined(_MSC_VER) && (_MSC_VER >= 1900)
#define DUK_SNPRINTF     snprintf
#define DUK_VSNPRINTF    vsnprintf
#else

#define DUK_SNPRINTF     _snprintf
#define DUK_VSNPRINTF    _vsnprintf
#endif

#if defined(_MSC_VER) && (_MSC_VER < 1500)
#pragma warning(disable: 4100 4101 4550 4551)
#define DUK_UNREF(x)
#else
#define DUK_UNREF(x)  do { __pragma(warning(suppress:4100 4101 4550 4551)) (x); } while (0)
#endif

#define DUK_U64_CONSTANT(x) x##ui64
#define DUK_I64_CONSTANT(x) x##i64
#elif defined(DUK_F_EMSCRIPTEN)

#define DUK_NORETURN(decl)  decl __attribute__((noreturn))

#if defined(__clang__) && defined(__has_builtin)
#if __has_builtin(__builtin_unreachable)
#define DUK_UNREACHABLE()  do { __builtin_unreachable(); } while (0)
#endif
#endif

#define DUK_USE_BRANCH_HINTS
#define DUK_LIKELY(x)    __builtin_expect((x), 1)
#define DUK_UNLIKELY(x)  __builtin_expect((x), 0)
#if defined(__clang__) && defined(__has_builtin)
#if __has_builtin(__builtin_unpredictable)
#define DUK_UNPREDICTABLE(x)  __builtin_unpredictable((x))
#endif
#endif

#if defined(DUK_F_C99) || defined(DUK_F_CPP11)
#define DUK_NOINLINE        __attribute__((noinline))
#define DUK_INLINE          inline
#define DUK_ALWAYS_INLINE   inline __attribute__((always_inline))
#endif

#define DUK_EXTERNAL_DECL  __attribute__ ((visibility("default"))) extern
#define DUK_EXTERNAL       __attribute__ ((visibility("default")))
#if defined(DUK_SINGLE_FILE)
#if (defined(DUK_F_GCC_VERSION) && DUK_F_GCC_VERSION >= 30101) || defined(DUK_F_CLANG)

#define DUK_INTERNAL_DECL  static __attribute__ ((unused))
#define DUK_INTERNAL       static __attribute__ ((unused))
#else
#define DUK_INTERNAL_DECL  static
#define DUK_INTERNAL       static
#endif
#else
#if (defined(DUK_F_GCC_VERSION) && DUK_F_GCC_VERSION >= 30101) || defined(DUK_F_CLANG)
#define DUK_INTERNAL_DECL  __attribute__ ((visibility("hidden"))) __attribute__ ((unused)) extern
#define DUK_INTERNAL       __attribute__ ((visibility("hidden"))) __attribute__ ((unused))
#else
#define DUK_INTERNAL_DECL  __attribute__ ((visibility("hidden"))) extern
#define DUK_INTERNAL       __attribute__ ((visibility("hidden")))
#endif
#endif
#define DUK_LOCAL_DECL     static
#define DUK_LOCAL          static

#define DUK_USE_COMPILER_STRING "emscripten"

#undef DUK_USE_VARIADIC_MACROS
#if defined(DUK_F_C99) || defined(DUK_F_CPP11)
#define DUK_USE_VARIADIC_MACROS
#endif

#define DUK_USE_UNION_INITIALIZERS

#undef DUK_USE_FLEX_C99
#undef DUK_USE_FLEX_ZEROSIZE
#undef DUK_USE_FLEX_ONESIZE
#if defined(DUK_F_C99)
#define DUK_USE_FLEX_C99
#else
#define DUK_USE_FLEX_ZEROSIZE
#endif

#undef DUK_USE_GCC_PRAGMAS
#define DUK_USE_PACK_CLANG_ATTR
#elif defined(DUK_F_TINYC)

#undef DUK_USE_BRANCH_HINTS

#if defined(DUK_F_CPP)
#define DUK_USE_COMPILER_STRING "tinyc++"
#else
#define DUK_USE_COMPILER_STRING "tinyc"
#endif

#define DUK_USE_VARIADIC_MACROS

#define DUK_USE_UNION_INITIALIZERS

#define DUK_USE_FLEX_ONESIZE

#define DUK_USE_PACK_DUMMY_MEMBER
#elif defined(DUK_F_VBCC)

#undef DUK_USE_BRANCH_HINTS

#if defined(DUK_F_CPP)
#define DUK_USE_COMPILER_STRING "vbcc-c++"
#else
#define DUK_USE_COMPILER_STRING "vbcc"
#endif

#undef DUK_USE_VARIADIC_MACROS
#if defined(DUK_F_C99) || defined(DUK_F_CPP11)
#define DUK_USE_VARIADIC_MACROS
#endif

#undef DUK_USE_UNION_INITIALIZERS
#if defined(DUK_F_C99)
#define DUK_USE_UNION_INITIALIZERS
#endif

#define DUK_USE_FLEX_ZEROSIZE
#define DUK_USE_PACK_DUMMY_MEMBER
#elif defined(DUK_F_BCC)

#undef DUK_USE_BRANCH_HINTS

#if defined(DUK_F_CPP)
#define DUK_USE_COMPILER_STRING "bcc++"
#else
#define DUK_USE_COMPILER_STRING "bcc"
#endif

#undef DUK_USE_VARIADIC_MACROS

#undef DUK_USE_UNION_INITIALIZERS

#define DUK_USE_FLEX_ONESIZE

#define DUK_USE_PACK_DUMMY_MEMBER

#if !defined(DUK_USE_BYTEORDER)
#define DUK_USE_BYTEORDER 1
#endif
#else

#undef DUK_USE_BRANCH_HINTS

#if defined(DUK_F_CPP)
#define DUK_USE_COMPILER_STRING "generic-c++"
#else
#define DUK_USE_COMPILER_STRING "generic"
#endif

#undef DUK_USE_VARIADIC_MACROS
#if defined(DUK_F_C99) || defined(DUK_F_CPP11)
#define DUK_USE_VARIADIC_MACROS
#endif

#undef DUK_USE_UNION_INITIALIZERS
#if defined(DUK_F_C99)
#define DUK_USE_UNION_INITIALIZERS
#endif

#define DUK_USE_FLEX_ONESIZE

#define DUK_USE_PACK_DUMMY_MEMBER
#endif  

#if defined(__UCLIBC__)
#define DUK_F_UCLIBC
#endif

#if !defined(INT_MAX)
#error INT_MAX not defined
#endif

#if defined(INT_MAX) && defined(INT_MIN)
#if INT_MAX != -(INT_MIN + 1)
#error platform does not seem complement of two
#endif
#else
#error cannot check complement of two
#endif

#if defined(DUK_F_X86) || defined(DUK_F_X32) || \
    defined(DUK_F_M68K) || defined(DUK_F_PPC32) || \
    defined(DUK_F_BCC) || \
    (defined(__WORDSIZE) && (__WORDSIZE == 32)) || \
    ((defined(DUK_F_OLD_SOLARIS) || defined(DUK_F_AIX) || \
      defined(DUK_F_HPUX)) && defined(_ILP32)) || \
    defined(DUK_F_ARM32)
#define DUK_F_32BIT_PTRS
#elif defined(DUK_F_X64) || \
      (defined(__WORDSIZE) && (__WORDSIZE == 64)) || \
   ((defined(DUK_F_OLD_SOLARIS) || defined(DUK_F_AIX) || \
     defined(DUK_F_HPUX)) && defined(_LP64)) || \
    defined(DUK_F_ARM64)
#define DUK_F_64BIT_PTRS
#else

#endif

#undef DUK_F_HAVE_INTTYPES
#if defined(__STDC_VERSION__) && (__STDC_VERSION__ >= 199901L) && \
    !(defined(DUK_F_AMIGAOS) && defined(DUK_F_VBCC))

#define DUK_F_HAVE_INTTYPES
#elif defined(__cplusplus) && (__cplusplus >= 201103L)

#define DUK_F_HAVE_INTTYPES
#endif

#if defined(DUK_F_HAVE_INTTYPES)

#define DUK_F_HAVE_64BIT
#include <inttypes.h>

typedef uint8_t duk_uint8_t;
typedef int8_t duk_int8_t;
typedef uint16_t duk_uint16_t;
typedef int16_t duk_int16_t;
typedef uint32_t duk_uint32_t;
typedef int32_t duk_int32_t;
typedef uint64_t duk_uint64_t;
typedef int64_t duk_int64_t;
typedef uint_least8_t duk_uint_least8_t;
typedef int_least8_t duk_int_least8_t;
typedef uint_least16_t duk_uint_least16_t;
typedef int_least16_t duk_int_least16_t;
typedef uint_least32_t duk_uint_least32_t;
typedef int_least32_t duk_int_least32_t;
typedef uint_least64_t duk_uint_least64_t;
typedef int_least64_t duk_int_least64_t;
typedef uint_fast8_t duk_uint_fast8_t;
typedef int_fast8_t duk_int_fast8_t;
typedef uint_fast16_t duk_uint_fast16_t;
typedef int_fast16_t duk_int_fast16_t;
typedef uint_fast32_t duk_uint_fast32_t;
typedef int_fast32_t duk_int_fast32_t;
typedef uint_fast64_t duk_uint_fast64_t;
typedef int_fast64_t duk_int_fast64_t;
typedef uintptr_t duk_uintptr_t;
typedef intptr_t duk_intptr_t;
typedef uintmax_t duk_uintmax_t;
typedef intmax_t duk_intmax_t;

#define DUK_UINT8_MIN         0
#define DUK_UINT8_MAX         UINT8_MAX
#define DUK_INT8_MIN          INT8_MIN
#define DUK_INT8_MAX          INT8_MAX
#define DUK_UINT_LEAST8_MIN   0
#define DUK_UINT_LEAST8_MAX   UINT_LEAST8_MAX
#define DUK_INT_LEAST8_MIN    INT_LEAST8_MIN
#define DUK_INT_LEAST8_MAX    INT_LEAST8_MAX
#define DUK_UINT_FAST8_MIN    0
#define DUK_UINT_FAST8_MAX    UINT_FAST8_MAX
#define DUK_INT_FAST8_MIN     INT_FAST8_MIN
#define DUK_INT_FAST8_MAX     INT_FAST8_MAX
#define DUK_UINT16_MIN        0
#define DUK_UINT16_MAX        UINT16_MAX
#define DUK_INT16_MIN         INT16_MIN
#define DUK_INT16_MAX         INT16_MAX
#define DUK_UINT_LEAST16_MIN  0
#define DUK_UINT_LEAST16_MAX  UINT_LEAST16_MAX
#define DUK_INT_LEAST16_MIN   INT_LEAST16_MIN
#define DUK_INT_LEAST16_MAX   INT_LEAST16_MAX
#define DUK_UINT_FAST16_MIN   0
#define DUK_UINT_FAST16_MAX   UINT_FAST16_MAX
#define DUK_INT_FAST16_MIN    INT_FAST16_MIN
#define DUK_INT_FAST16_MAX    INT_FAST16_MAX
#define DUK_UINT32_MIN        0
#define DUK_UINT32_MAX        UINT32_MAX
#define DUK_INT32_MIN         INT32_MIN
#define DUK_INT32_MAX         INT32_MAX
#define DUK_UINT_LEAST32_MIN  0
#define DUK_UINT_LEAST32_MAX  UINT_LEAST32_MAX
#define DUK_INT_LEAST32_MIN   INT_LEAST32_MIN
#define DUK_INT_LEAST32_MAX   INT_LEAST32_MAX
#define DUK_UINT_FAST32_MIN   0
#define DUK_UINT_FAST32_MAX   UINT_FAST32_MAX
#define DUK_INT_FAST32_MIN    INT_FAST32_MIN
#define DUK_INT_FAST32_MAX    INT_FAST32_MAX
#define DUK_UINT64_MIN        0
#define DUK_UINT64_MAX        UINT64_MAX
#define DUK_INT64_MIN         INT64_MIN
#define DUK_INT64_MAX         INT64_MAX
#define DUK_UINT_LEAST64_MIN  0
#define DUK_UINT_LEAST64_MAX  UINT_LEAST64_MAX
#define DUK_INT_LEAST64_MIN   INT_LEAST64_MIN
#define DUK_INT_LEAST64_MAX   INT_LEAST64_MAX
#define DUK_UINT_FAST64_MIN   0
#define DUK_UINT_FAST64_MAX   UINT_FAST64_MAX
#define DUK_INT_FAST64_MIN    INT_FAST64_MIN
#define DUK_INT_FAST64_MAX    INT_FAST64_MAX

#define DUK_UINTPTR_MIN       0
#define DUK_UINTPTR_MAX       UINTPTR_MAX
#define DUK_INTPTR_MIN        INTPTR_MIN
#define DUK_INTPTR_MAX        INTPTR_MAX

#define DUK_UINTMAX_MIN       0
#define DUK_UINTMAX_MAX       UINTMAX_MAX
#define DUK_INTMAX_MIN        INTMAX_MIN
#define DUK_INTMAX_MAX        INTMAX_MAX

#define DUK_SIZE_MIN          0
#define DUK_SIZE_MAX          SIZE_MAX
#undef DUK_SIZE_MAX_COMPUTED

#else  

#if (defined(CHAR_BIT) && (CHAR_BIT == 8)) || \
    (defined(UCHAR_MAX) && (UCHAR_MAX == 255))
typedef unsigned char duk_uint8_t;
typedef signed char duk_int8_t;
#else
#error cannot detect 8-bit type
#endif

#if defined(USHRT_MAX) && (USHRT_MAX == 65535UL)
typedef unsigned short duk_uint16_t;
typedef signed short duk_int16_t;
#elif defined(UINT_MAX) && (UINT_MAX == 65535UL)

typedef unsigned int duk_uint16_t;
typedef signed int duk_int16_t;
#else
#error cannot detect 16-bit type
#endif

#if defined(UINT_MAX) && (UINT_MAX == 4294967295UL)
typedef unsigned int duk_uint32_t;
typedef signed int duk_int32_t;
#elif defined(ULONG_MAX) && (ULONG_MAX == 4294967295UL)

typedef unsigned long duk_uint32_t;
typedef signed long duk_int32_t;
#else
#error cannot detect 32-bit type
#endif

#undef DUK_F_HAVE_64BIT
#if !defined(DUK_F_HAVE_64BIT) && defined(ULONG_MAX)
#if (ULONG_MAX > 4294967295UL)
#define DUK_F_HAVE_64BIT
typedef unsigned long duk_uint64_t;
typedef signed long duk_int64_t;
#endif
#endif
#if !defined(DUK_F_HAVE_64BIT) && defined(ULLONG_MAX)
#if (ULLONG_MAX > 4294967295UL)
#define DUK_F_HAVE_64BIT
typedef unsigned long long duk_uint64_t;
typedef signed long long duk_int64_t;
#endif
#endif
#if !defined(DUK_F_HAVE_64BIT) && defined(__ULONG_LONG_MAX__)
#if (__ULONG_LONG_MAX__ > 4294967295UL)
#define DUK_F_HAVE_64BIT
typedef unsigned long long duk_uint64_t;
typedef signed long long duk_int64_t;
#endif
#endif
#if !defined(DUK_F_HAVE_64BIT) && defined(__LONG_LONG_MAX__)
#if (__LONG_LONG_MAX__ > 2147483647L)
#define DUK_F_HAVE_64BIT
typedef unsigned long long duk_uint64_t;
typedef signed long long duk_int64_t;
#endif
#endif
#if !defined(DUK_F_HAVE_64BIT) && defined(DUK_F_MINGW)
#define DUK_F_HAVE_64BIT
typedef unsigned long duk_uint64_t;
typedef signed long duk_int64_t;
#endif
#if !defined(DUK_F_HAVE_64BIT) && defined(DUK_F_MSVC)
#define DUK_F_HAVE_64BIT
typedef unsigned __int64 duk_uint64_t;
typedef signed __int64 duk_int64_t;
#endif
#if !defined(DUK_F_HAVE_64BIT)

#endif

typedef duk_uint8_t duk_uint_least8_t;
typedef duk_int8_t duk_int_least8_t;
typedef duk_uint16_t duk_uint_least16_t;
typedef duk_int16_t duk_int_least16_t;
typedef duk_uint32_t duk_uint_least32_t;
typedef duk_int32_t duk_int_least32_t;
typedef duk_uint8_t duk_uint_fast8_t;
typedef duk_int8_t duk_int_fast8_t;
typedef duk_uint16_t duk_uint_fast16_t;
typedef duk_int16_t duk_int_fast16_t;
typedef duk_uint32_t duk_uint_fast32_t;
typedef duk_int32_t duk_int_fast32_t;
#if defined(DUK_F_HAVE_64BIT)
typedef duk_uint64_t duk_uint_least64_t;
typedef duk_int64_t duk_int_least64_t;
typedef duk_uint64_t duk_uint_fast64_t;
typedef duk_int64_t duk_int_fast64_t;
#endif
#if defined(DUK_F_HAVE_64BIT)
typedef duk_uint64_t duk_uintmax_t;
typedef duk_int64_t duk_intmax_t;
#else
typedef duk_uint32_t duk_uintmax_t;
typedef duk_int32_t duk_intmax_t;
#endif

#define DUK_UINT8_MIN         0UL
#define DUK_UINT8_MAX         0xffUL
#define DUK_INT8_MIN          (-0x80L)
#define DUK_INT8_MAX          0x7fL
#define DUK_UINT_LEAST8_MIN   0UL
#define DUK_UINT_LEAST8_MAX   0xffUL
#define DUK_INT_LEAST8_MIN    (-0x80L)
#define DUK_INT_LEAST8_MAX    0x7fL
#define DUK_UINT_FAST8_MIN    0UL
#define DUK_UINT_FAST8_MAX    0xffUL
#define DUK_INT_FAST8_MIN     (-0x80L)
#define DUK_INT_FAST8_MAX     0x7fL
#define DUK_UINT16_MIN        0UL
#define DUK_UINT16_MAX        0xffffUL
#define DUK_INT16_MIN         (-0x7fffL - 1L)
#define DUK_INT16_MAX         0x7fffL
#define DUK_UINT_LEAST16_MIN  0UL
#define DUK_UINT_LEAST16_MAX  0xffffUL
#define DUK_INT_LEAST16_MIN   (-0x7fffL - 1L)
#define DUK_INT_LEAST16_MAX   0x7fffL
#define DUK_UINT_FAST16_MIN   0UL
#define DUK_UINT_FAST16_MAX   0xffffUL
#define DUK_INT_FAST16_MIN    (-0x7fffL - 1L)
#define DUK_INT_FAST16_MAX    0x7fffL
#define DUK_UINT32_MIN        0UL
#define DUK_UINT32_MAX        0xffffffffUL
#define DUK_INT32_MIN         (-0x7fffffffL - 1L)
#define DUK_INT32_MAX         0x7fffffffL
#define DUK_UINT_LEAST32_MIN  0UL
#define DUK_UINT_LEAST32_MAX  0xffffffffUL
#define DUK_INT_LEAST32_MIN   (-0x7fffffffL - 1L)
#define DUK_INT_LEAST32_MAX   0x7fffffffL
#define DUK_UINT_FAST32_MIN   0UL
#define DUK_UINT_FAST32_MAX   0xffffffffUL
#define DUK_INT_FAST32_MIN    (-0x7fffffffL - 1L)
#define DUK_INT_FAST32_MAX    0x7fffffffL

#if defined(DUK_F_HAVE_64BIT)
#define DUK_UINT64_MIN        ((duk_uint64_t) 0)
#define DUK_UINT64_MAX        ((duk_uint64_t) -1)
#define DUK_INT64_MIN         ((duk_int64_t) (~(DUK_UINT64_MAX >> 1)))
#define DUK_INT64_MAX         ((duk_int64_t) (DUK_UINT64_MAX >> 1))
#define DUK_UINT_LEAST64_MIN  DUK_UINT64_MIN
#define DUK_UINT_LEAST64_MAX  DUK_UINT64_MAX
#define DUK_INT_LEAST64_MIN   DUK_INT64_MIN
#define DUK_INT_LEAST64_MAX   DUK_INT64_MAX
#define DUK_UINT_FAST64_MIN   DUK_UINT64_MIN
#define DUK_UINT_FAST64_MAX   DUK_UINT64_MAX
#define DUK_INT_FAST64_MIN    DUK_INT64_MIN
#define DUK_INT_FAST64_MAX    DUK_INT64_MAX
#define DUK_UINT64_MIN_COMPUTED
#define DUK_UINT64_MAX_COMPUTED
#define DUK_INT64_MIN_COMPUTED
#define DUK_INT64_MAX_COMPUTED
#define DUK_UINT_LEAST64_MIN_COMPUTED
#define DUK_UINT_LEAST64_MAX_COMPUTED
#define DUK_INT_LEAST64_MIN_COMPUTED
#define DUK_INT_LEAST64_MAX_COMPUTED
#define DUK_UINT_FAST64_MIN_COMPUTED
#define DUK_UINT_FAST64_MAX_COMPUTED
#define DUK_INT_FAST64_MIN_COMPUTED
#define DUK_INT_FAST64_MAX_COMPUTED
#endif

#if defined(DUK_F_HAVE_64BIT)
#define DUK_UINTMAX_MIN       DUK_UINT64_MIN
#define DUK_UINTMAX_MAX       DUK_UINT64_MAX
#define DUK_INTMAX_MIN        DUK_INT64_MIN
#define DUK_INTMAX_MAX        DUK_INT64_MAX
#define DUK_UINTMAX_MIN_COMPUTED
#define DUK_UINTMAX_MAX_COMPUTED
#define DUK_INTMAX_MIN_COMPUTED
#define DUK_INTMAX_MAX_COMPUTED
#else
#define DUK_UINTMAX_MIN       0UL
#define DUK_UINTMAX_MAX       0xffffffffUL
#define DUK_INTMAX_MIN        (-0x7fffffffL - 1L)
#define DUK_INTMAX_MAX        0x7fffffffL
#endif

#if defined(DUK_F_32BIT_PTRS)
typedef duk_int32_t duk_intptr_t;
typedef duk_uint32_t duk_uintptr_t;
#define DUK_UINTPTR_MIN       DUK_UINT32_MIN
#define DUK_UINTPTR_MAX       DUK_UINT32_MAX
#define DUK_INTPTR_MIN        DUK_INT32_MIN
#define DUK_INTPTR_MAX        DUK_INT32_MAX
#elif defined(DUK_F_64BIT_PTRS) && defined(DUK_F_HAVE_64BIT)
typedef duk_int64_t duk_intptr_t;
typedef duk_uint64_t duk_uintptr_t;
#define DUK_UINTPTR_MIN       DUK_UINT64_MIN
#define DUK_UINTPTR_MAX       DUK_UINT64_MAX
#define DUK_INTPTR_MIN        DUK_INT64_MIN
#define DUK_INTPTR_MAX        DUK_INT64_MAX
#define DUK_UINTPTR_MIN_COMPUTED
#define DUK_UINTPTR_MAX_COMPUTED
#define DUK_INTPTR_MIN_COMPUTED
#define DUK_INTPTR_MAX_COMPUTED
#else
#error cannot determine intptr type
#endif

#undef DUK_SIZE_MAX_COMPUTED
#if !defined(SIZE_MAX)
#define DUK_SIZE_MAX_COMPUTED
#define SIZE_MAX              ((size_t) (-1))
#endif
#define DUK_SIZE_MIN          0
#define DUK_SIZE_MAX          SIZE_MAX

#endif  

typedef size_t duk_size_t;
typedef ptrdiff_t duk_ptrdiff_t;

#if defined(UINT_MAX) && (UINT_MAX >= 0xffffffffUL)
typedef int duk_int_t;
typedef unsigned int duk_uint_t;
#define DUK_INT_MIN           INT_MIN
#define DUK_INT_MAX           INT_MAX
#define DUK_UINT_MIN          0
#define DUK_UINT_MAX          UINT_MAX
#else
typedef duk_int_fast32_t duk_int_t;
typedef duk_uint_fast32_t duk_uint_t;
#define DUK_INT_MIN           DUK_INT_FAST32_MIN
#define DUK_INT_MAX           DUK_INT_FAST32_MAX
#define DUK_UINT_MIN          DUK_UINT_FAST32_MIN
#define DUK_UINT_MAX          DUK_UINT_FAST32_MAX
#endif

typedef duk_int_fast32_t duk_int_fast_t;
typedef duk_uint_fast32_t duk_uint_fast_t;
#define DUK_INT_FAST_MIN      DUK_INT_FAST32_MIN
#define DUK_INT_FAST_MAX      DUK_INT_FAST32_MAX
#define DUK_UINT_FAST_MIN     DUK_UINT_FAST32_MIN
#define DUK_UINT_FAST_MAX     DUK_UINT_FAST32_MAX

typedef int duk_small_int_t;
typedef unsigned int duk_small_uint_t;
#define DUK_SMALL_INT_MIN     INT_MIN
#define DUK_SMALL_INT_MAX     INT_MAX
#define DUK_SMALL_UINT_MIN    0
#define DUK_SMALL_UINT_MAX    UINT_MAX

typedef duk_int_fast16_t duk_small_int_fast_t;
typedef duk_uint_fast16_t duk_small_uint_fast_t;
#define DUK_SMALL_INT_FAST_MIN    DUK_INT_FAST16_MIN
#define DUK_SMALL_INT_FAST_MAX    DUK_INT_FAST16_MAX
#define DUK_SMALL_UINT_FAST_MIN   DUK_UINT_FAST16_MIN
#define DUK_SMALL_UINT_FAST_MAX   DUK_UINT_FAST16_MAX

typedef duk_small_uint_t duk_bool_t;
#define DUK_BOOL_MIN              DUK_SMALL_INT_MIN
#define DUK_BOOL_MAX              DUK_SMALL_INT_MAX

typedef duk_int_t duk_idx_t;
#define DUK_IDX_MIN               DUK_INT_MIN
#define DUK_IDX_MAX               DUK_INT_MAX

typedef duk_uint_t duk_uidx_t;
#define DUK_UIDX_MIN              DUK_UINT_MIN
#define DUK_UIDX_MAX              DUK_UINT_MAX

typedef duk_uint_t duk_uarridx_t;
#define DUK_UARRIDX_MIN           DUK_UINT_MIN
#define DUK_UARRIDX_MAX           DUK_UINT_MAX

typedef duk_small_int_t duk_ret_t;
#define DUK_RET_MIN               DUK_SMALL_INT_MIN
#define DUK_RET_MAX               DUK_SMALL_INT_MAX

typedef duk_int_t duk_errcode_t;
#define DUK_ERRCODE_MIN           DUK_INT_MIN
#define DUK_ERRCODE_MAX           DUK_INT_MAX

typedef duk_int_t duk_codepoint_t;
typedef duk_uint_t duk_ucodepoint_t;
#define DUK_CODEPOINT_MIN         DUK_INT_MIN
#define DUK_CODEPOINT_MAX         DUK_INT_MAX
#define DUK_UCODEPOINT_MIN        DUK_UINT_MIN
#define DUK_UCODEPOINT_MAX        DUK_UINT_MAX

typedef float duk_float_t;
typedef double duk_double_t;

#if !defined(DUK_SIZE_MAX)
#error DUK_SIZE_MAX is undefined, probably missing SIZE_MAX
#elif !defined(DUK_SIZE_MAX_COMPUTED)
#if DUK_SIZE_MAX < 0xffffffffUL

#endif
#endif

typedef struct duk_hthread duk_context;

#if defined(DUK_F_HAVE_64BIT) && !defined(DUK_F_VBCC)
#define DUK_USE_64BIT_OPS
#else
#undef DUK_USE_64BIT_OPS
#endif

#if !defined(DUK_ABORT)
#define DUK_ABORT             abort
#endif

#if !defined(DUK_SETJMP)
#define DUK_JMPBUF_TYPE       jmp_buf
#define DUK_SETJMP(jb)        setjmp((jb))
#define DUK_LONGJMP(jb)       longjmp((jb), 1)
#endif

#if 0

#define DUK_JMPBUF_TYPE       sigjmp_buf
#define DUK_SETJMP(jb)        sigsetjmp((jb))
#define DUK_LONGJMP(jb)       siglongjmp((jb), 1)
#endif

#if !defined(DUK_ANSI_MALLOC)
#define DUK_ANSI_MALLOC      malloc
#endif
#if !defined(DUK_ANSI_REALLOC)
#define DUK_ANSI_REALLOC     realloc
#endif
#if !defined(DUK_ANSI_CALLOC)
#define DUK_ANSI_CALLOC      calloc
#endif
#if !defined(DUK_ANSI_FREE)
#define DUK_ANSI_FREE        free
#endif

#if !defined(DUK_MEMCPY)
#if defined(DUK_F_UCLIBC)

#define DUK_MEMCPY       memmove
#else
#define DUK_MEMCPY       memcpy
#endif
#endif
#if !defined(DUK_MEMMOVE)
#define DUK_MEMMOVE      memmove
#endif
#if !defined(DUK_MEMCMP)
#define DUK_MEMCMP       memcmp
#endif
#if !defined(DUK_MEMSET)
#define DUK_MEMSET       memset
#endif
#if !defined(DUK_STRLEN)
#define DUK_STRLEN       strlen
#endif
#if !defined(DUK_STRCMP)
#define DUK_STRCMP       strcmp
#endif
#if !defined(DUK_STRNCMP)
#define DUK_STRNCMP      strncmp
#endif
#if !defined(DUK_SPRINTF)
#define DUK_SPRINTF      sprintf
#endif
#if !defined(DUK_SNPRINTF)

#define DUK_SNPRINTF     snprintf
#endif
#if !defined(DUK_VSPRINTF)
#define DUK_VSPRINTF     vsprintf
#endif
#if !defined(DUK_VSNPRINTF)

#define DUK_VSNPRINTF    vsnprintf
#endif
#if !defined(DUK_SSCANF)
#define DUK_SSCANF       sscanf
#endif
#if !defined(DUK_VSSCANF)
#define DUK_VSSCANF      vsscanf
#endif
#if !defined(DUK_MEMZERO)
#define DUK_MEMZERO(p,n) DUK_MEMSET((p), 0, (n))
#endif

#if !defined(DUK_DOUBLE_INFINITY)
#undef DUK_USE_COMPUTED_INFINITY
#if defined(DUK_F_GCC_VERSION) && (DUK_F_GCC_VERSION < 40600)

#define DUK_DOUBLE_INFINITY  (__builtin_inf())
#elif defined(INFINITY)
#define DUK_DOUBLE_INFINITY  ((double) INFINITY)
#elif !defined(DUK_F_VBCC) && !defined(DUK_F_MSVC) && !defined(DUK_F_BCC) && \
      !defined(DUK_F_OLD_SOLARIS) && !defined(DUK_F_AIX)
#define DUK_DOUBLE_INFINITY  (1.0 / 0.0)
#else

#define DUK_USE_COMPUTED_INFINITY
#define DUK_DOUBLE_INFINITY  duk_computed_infinity
#endif
#endif

#if !defined(DUK_DOUBLE_NAN)
#undef DUK_USE_COMPUTED_NAN
#if defined(NAN)
#define DUK_DOUBLE_NAN       NAN
#elif !defined(DUK_F_VBCC) && !defined(DUK_F_MSVC) && !defined(DUK_F_BCC) && \
      !defined(DUK_F_OLD_SOLARIS) && !defined(DUK_F_AIX)
#define DUK_DOUBLE_NAN       (0.0 / 0.0)
#else

#define DUK_USE_COMPUTED_NAN
#define DUK_DOUBLE_NAN       duk_computed_nan
#endif
#endif

#undef DUK_USE_REPL_FPCLASSIFY
#undef DUK_USE_REPL_SIGNBIT
#undef DUK_USE_REPL_ISFINITE
#undef DUK_USE_REPL_ISNAN
#undef DUK_USE_REPL_ISINF

#undef DUK_F_USE_REPL_ALL
#if !(defined(FP_NAN) && defined(FP_INFINITE) && defined(FP_ZERO) && \
      defined(FP_SUBNORMAL) && defined(FP_NORMAL))

#define DUK_F_USE_REPL_ALL
#elif defined(DUK_F_AMIGAOS) && defined(DUK_F_VBCC)

#define DUK_F_USE_REPL_ALL
#elif defined(DUK_F_AMIGAOS) && defined(DUK_F_M68K)

#define DUK_F_USE_REPL_ALL
#elif defined(DUK_F_FREEBSD) && defined(DUK_F_CLANG)

#define DUK_F_USE_REPL_ALL
#elif defined(DUK_F_UCLIBC)

#define DUK_F_USE_REPL_ALL
#elif defined(DUK_F_AIX)

#define DUK_F_USE_REPL_ALL
#endif

#if defined(DUK_F_USE_REPL_ALL)
#define DUK_USE_REPL_FPCLASSIFY
#define DUK_USE_REPL_SIGNBIT
#define DUK_USE_REPL_ISFINITE
#define DUK_USE_REPL_ISNAN
#define DUK_USE_REPL_ISINF
#define DUK_FPCLASSIFY       duk_repl_fpclassify
#define DUK_SIGNBIT          duk_repl_signbit
#define DUK_ISFINITE         duk_repl_isfinite
#define DUK_ISNAN            duk_repl_isnan
#define DUK_ISINF            duk_repl_isinf
#define DUK_FP_NAN           0
#define DUK_FP_INFINITE      1
#define DUK_FP_ZERO          2
#define DUK_FP_SUBNORMAL     3
#define DUK_FP_NORMAL        4
#else
#define DUK_FPCLASSIFY       fpclassify
#define DUK_SIGNBIT          signbit
#define DUK_ISFINITE         isfinite
#define DUK_ISNAN            isnan
#define DUK_ISINF            isinf
#define DUK_FP_NAN           FP_NAN
#define DUK_FP_INFINITE      FP_INFINITE
#define DUK_FP_ZERO          FP_ZERO
#define DUK_FP_SUBNORMAL     FP_SUBNORMAL
#define DUK_FP_NORMAL        FP_NORMAL
#endif

#if defined(DUK_F_USE_REPL_ALL)
#undef DUK_F_USE_REPL_ALL
#endif

#if !defined(DUK_FABS)
#define DUK_FABS             fabs
#endif
#if !defined(DUK_FLOOR)
#define DUK_FLOOR            floor
#endif
#if !defined(DUK_CEIL)
#define DUK_CEIL             ceil
#endif
#if !defined(DUK_FMOD)
#define DUK_FMOD             fmod
#endif
#if !defined(DUK_POW)
#define DUK_POW              pow
#endif
#if !defined(DUK_ACOS)
#define DUK_ACOS             acos
#endif
#if !defined(DUK_ASIN)
#define DUK_ASIN             asin
#endif
#if !defined(DUK_ATAN)
#define DUK_ATAN             atan
#endif
#if !defined(DUK_ATAN2)
#define DUK_ATAN2            atan2
#endif
#if !defined(DUK_SIN)
#define DUK_SIN              sin
#endif
#if !defined(DUK_COS)
#define DUK_COS              cos
#endif
#if !defined(DUK_TAN)
#define DUK_TAN              tan
#endif
#if !defined(DUK_EXP)
#define DUK_EXP              exp
#endif
#if !defined(DUK_LOG)
#define DUK_LOG              log
#endif
#if !defined(DUK_SQRT)
#define DUK_SQRT             sqrt
#endif

#if (defined(DUK_F_C99) || defined(DUK_F_CPP11) || (defined(_MSC_VER) && (_MSC_VER >= 1800))) && \
    !defined(DUK_F_ANDROID) && !defined(DUK_F_MINT)
#if !defined(DUK_CBRT)
#define DUK_CBRT             cbrt
#endif
#if !defined(DUK_LOG2)
#define DUK_LOG2             log2
#endif
#if !defined(DUK_LOG10)
#define DUK_LOG10            log10
#endif
#if !defined(DUK_TRUNC)
#define DUK_TRUNC            trunc
#endif
#endif  

#undef DUK_USE_POW_WORKAROUNDS
#if defined(DUK_F_NETBSD) || defined(DUK_F_MINGW)
#define DUK_USE_POW_WORKAROUNDS
#endif

#undef DUK_USE_ATAN2_WORKAROUNDS
#if defined(DUK_F_MINGW)
#define DUK_USE_ATAN2_WORKAROUNDS
#endif

#undef DUK_USE_PARANOID_MATH

#undef DUK_USE_PARANOID_DATE_COMPUTATION
#if !defined(DUK_F_C99)
#define DUK_USE_PARANOID_DATE_COMPUTATION
#endif

#if !defined(DUK_USE_BYTEORDER) && defined(__BYTE_ORDER__)
#if defined(__ORDER_LITTLE_ENDIAN__) && (__BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__)
#if defined(__FLOAT_WORD_ORDER__) && defined(__ORDER_LITTLE_ENDIAN__) && (__FLOAT_WORD_ORDER__ == __ORDER_LITTLE_ENDIAN__)
#define DUK_USE_BYTEORDER 1
#elif defined(__FLOAT_WORD_ORDER__) && defined(__ORDER_BIG_ENDIAN__) && (__FLOAT_WORD_ORDER__ == __ORDER_BIG_ENDIAN__)
#define DUK_USE_BYTEORDER 2
#elif !defined(__FLOAT_WORD_ORDER__)

#define DUK_USE_BYTEORDER 1
#else

#endif  
#elif defined(__ORDER_BIG_ENDIAN__) && (__BYTE_ORDER__ == __ORDER_BIG_ENDIAN__)
#if defined(__FLOAT_WORD_ORDER__) && defined(__ORDER_BIG_ENDIAN__) && (__FLOAT_WORD_ORDER__ == __ORDER_BIG_ENDIAN__)
#define DUK_USE_BYTEORDER 3
#elif !defined(__FLOAT_WORD_ORDER__)

#define DUK_USE_BYTEORDER 3
#else

#endif  
#else

#endif  
#endif  

#if !defined(DUK_USE_BYTEORDER)
#if defined(__BYTE_ORDER) && defined(__LITTLE_ENDIAN) && (__BYTE_ORDER == __LITTLE_ENDIAN) || \
    defined(_BYTE_ORDER) && defined(_LITTLE_ENDIAN) && (_BYTE_ORDER == _LITTLE_ENDIAN) || \
    defined(__LITTLE_ENDIAN__)
#if defined(__FLOAT_WORD_ORDER) && defined(__LITTLE_ENDIAN) && (__FLOAT_WORD_ORDER == __LITTLE_ENDIAN) || \
    defined(_FLOAT_WORD_ORDER) && defined(_LITTLE_ENDIAN) && (_FLOAT_WORD_ORDER == _LITTLE_ENDIAN)
#define DUK_USE_BYTEORDER 1
#elif defined(__FLOAT_WORD_ORDER) && defined(__BIG_ENDIAN) && (__FLOAT_WORD_ORDER == __BIG_ENDIAN) || \
      defined(_FLOAT_WORD_ORDER) && defined(_BIG_ENDIAN) && (_FLOAT_WORD_ORDER == _BIG_ENDIAN)
#define DUK_USE_BYTEORDER 2
#elif !defined(__FLOAT_WORD_ORDER) && !defined(_FLOAT_WORD_ORDER)

#define DUK_USE_BYTEORDER 1
#else

#endif  
#elif defined(__BYTE_ORDER) && defined(__BIG_ENDIAN) && (__BYTE_ORDER == __BIG_ENDIAN) || \
      defined(_BYTE_ORDER) && defined(_BIG_ENDIAN) && (_BYTE_ORDER == _BIG_ENDIAN) || \
      defined(__BIG_ENDIAN__)
#if defined(__FLOAT_WORD_ORDER) && defined(__BIG_ENDIAN) && (__FLOAT_WORD_ORDER == __BIG_ENDIAN) || \
    defined(_FLOAT_WORD_ORDER) && defined(_BIG_ENDIAN) && (_FLOAT_WORD_ORDER == _BIG_ENDIAN)
#define DUK_USE_BYTEORDER 3
#elif !defined(__FLOAT_WORD_ORDER) && !defined(_FLOAT_WORD_ORDER)

#define DUK_USE_BYTEORDER 3
#else

#endif  
#else

#endif  
#endif  

#if !defined(DUK_USE_BYTEORDER)
#if defined(__LITTLEENDIAN__)
#define DUK_USE_BYTEORDER 1
#elif defined(__BIGENDIAN__)
#define DUK_USE_BYTEORDER 3
#endif
#endif

#if !defined(DUK_USE_ALIGN_BY)
#define DUK_USE_ALIGN_BY 8
#endif

#if !(defined(DUK_USE_PACK_MSVC_PRAGMA) || defined(DUK_USE_PACK_GCC_ATTR) || \
      defined(DUK_USE_PACK_CLANG_ATTR) || defined(DUK_USE_PACK_DUMMY_MEMBER))
#define DUK_USE_PACK_DUMMY_MEMBER
#endif

#if !defined(DUK_VA_COPY)

#if defined(DUK_F_C99) || defined(DUK_F_CPP11)

#define DUK_VA_COPY(dest,src) va_copy(dest,src)
#else

#define DUK_VA_COPY(dest,src) do { (dest) = (src); } while (0)
#endif
#endif

#if !defined(DUK_MACRO_STRINGIFY)

#define DUK_MACRO_STRINGIFY_HELPER(x)  #x
#define DUK_MACRO_STRINGIFY(x)  DUK_MACRO_STRINGIFY_HELPER(x)
#endif

#if !defined(DUK_CAUSE_SEGFAULT)

#define DUK_CAUSE_SEGFAULT()  do { *((volatile duk_uint32_t *) NULL) = (duk_uint32_t) 0xdeadbeefUL; } while (0)
#endif
#if !defined(DUK_UNREF)

#define DUK_UNREF(x)  do { (void) (x); } while (0)
#endif
#if !defined(DUK_NORETURN)
#define DUK_NORETURN(decl)  decl
#endif
#if !defined(DUK_UNREACHABLE)

#define DUK_UNREACHABLE()  do { } while (0)
#endif
#if !defined(DUK_LOSE_CONST)

#define DUK_LOSE_CONST(src) ((void *) (duk_uintptr_t) (src))
#endif

#if !defined(DUK_LIKELY)
#define DUK_LIKELY(x)    (x)
#endif
#if !defined(DUK_UNLIKELY)
#define DUK_UNLIKELY(x)  (x)
#endif
#if !defined(DUK_UNPREDICTABLE)
#define DUK_UNPREDICTABLE(x)  (x)
#endif

#if !defined(DUK_NOINLINE)
#define DUK_NOINLINE       
#endif
#if !defined(DUK_INLINE)
#define DUK_INLINE         
#endif
#if !defined(DUK_ALWAYS_INLINE)
#define DUK_ALWAYS_INLINE  
#endif

#if !defined(DUK_HOT)
#define DUK_HOT            
#endif
#if !defined(DUK_COLD)
#define DUK_COLD           
#endif

#if !defined(DUK_EXTERNAL_DECL)
#define DUK_EXTERNAL_DECL  extern
#endif
#if !defined(DUK_EXTERNAL)
#define DUK_EXTERNAL       
#endif
#if !defined(DUK_INTERNAL_DECL)
#if defined(DUK_SINGLE_FILE)
#define DUK_INTERNAL_DECL  static
#else
#define DUK_INTERNAL_DECL  extern
#endif
#endif
#if !defined(DUK_INTERNAL)
#if defined(DUK_SINGLE_FILE)
#define DUK_INTERNAL       static
#else
#define DUK_INTERNAL       
#endif
#endif
#if !defined(DUK_LOCAL_DECL)
#define DUK_LOCAL_DECL     static
#endif
#if !defined(DUK_LOCAL)
#define DUK_LOCAL          static
#endif

#if !defined(DUK_FILE_MACRO)
#define DUK_FILE_MACRO  __FILE__
#endif
#if !defined(DUK_LINE_MACRO)
#define DUK_LINE_MACRO  __LINE__
#endif
#if !defined(DUK_FUNC_MACRO)
#if defined(DUK_F_C99) || defined(DUK_F_CPP11)
#define DUK_FUNC_MACRO  __func__
#elif defined(__FUNCTION__)
#define DUK_FUNC_MACRO  __FUNCTION__
#else
#define DUK_FUNC_MACRO  "unknown"
#endif
#endif

#if !defined(DUK_BSWAP32)
#define DUK_BSWAP32(x) \
	((((duk_uint32_t) (x)) >> 24) | \
	 ((((duk_uint32_t) (x)) >> 8) & 0xff00UL) | \
	 ((((duk_uint32_t) (x)) << 8) & 0xff0000UL) | \
	 (((duk_uint32_t) (x)) << 24))
#endif
#if !defined(DUK_BSWAP16)
#define DUK_BSWAP16(x) \
	((duk_uint16_t) (x) >> 8) | \
	((duk_uint16_t) (x) << 8)
#endif

#if !(defined(DUK_USE_FLEX_C99) || defined(DUK_USE_FLEX_ZEROSIZE) || defined(DUK_USE_FLEX_ONESIZE))
#if defined(DUK_F_C99)
#define DUK_USE_FLEX_C99
#else
#define DUK_USE_FLEX_ZEROSIZE  
#endif
#endif

#if !(defined(DUK_USE_PACK_GCC_ATTR) || defined(DUK_USE_PACK_CLANG_ATTR) || \
      defined(DUK_USE_PACK_MSVC_PRAGMA) || defined(DUK_USE_PACK_DUMMY_MEMBER))
#define DUK_USE_PACK_DUMMY_MEMBER
#endif

#if 0  
#undef DUK_USE_GCC_PRAGMAS
#endif

#if !defined(DUK_U64_CONSTANT)
#define DUK_U64_CONSTANT(x) x##ULL
#endif
#if !defined(DUK_I64_CONSTANT)
#define DUK_I64_CONSTANT(x) x##LL
#endif

#if !defined(DUK_SINGLE_FILE)
#undef DUK_NOINLINE
#undef DUK_INLINE
#undef DUK_ALWAYS_INLINE
#define DUK_NOINLINE       
#define DUK_INLINE         
#define DUK_ALWAYS_INLINE  
#endif

#if !defined(DUK_F_PACKED_TVAL_PROVIDED)
#undef DUK_F_PACKED_TVAL_POSSIBLE

#if !defined(DUK_F_PACKED_TVAL_POSSIBLE) && defined(DUK_UINTPTR_MAX)
#if (DUK_UINTPTR_MAX <= 0xffffffffUL)
#define DUK_F_PACKED_TVAL_POSSIBLE
#endif
#endif

#if !defined(DUK_F_PACKED_TVAL_POSSIBLE) && defined(DUK_UINTPTR_MAX) && !defined(DUK_UINTPTR_MAX_COMPUTED)
#if (DUK_UINTPTR_MAX <= 0xffffffffUL)
#define DUK_F_PACKED_TVAL_POSSIBLE
#endif
#endif

#if !defined(DUK_F_PACKED_TVAL_POSSIBLE) && defined(DUK_SIZE_MAX) && !defined(DUK_SIZE_MAX_COMPUTED)
#if (DUK_SIZE_MAX <= 0xffffffffUL)
#define DUK_F_PACKED_TVAL_POSSIBLE
#endif
#endif

#undef DUK_USE_PACKED_TVAL
#if defined(DUK_F_PACKED_TVAL_POSSIBLE)
#define DUK_USE_PACKED_TVAL
#endif

#undef DUK_F_PACKED_TVAL_POSSIBLE
#endif  

#undef DUK_USE_HOBJECT_LAYOUT_1
#undef DUK_USE_HOBJECT_LAYOUT_2
#undef DUK_USE_HOBJECT_LAYOUT_3
#if (DUK_USE_ALIGN_BY == 1)

#define DUK_USE_HOBJECT_LAYOUT_1
#else

#define DUK_USE_HOBJECT_LAYOUT_2
#endif

#if defined(__FAST_MATH__)
#error __FAST_MATH__ defined, refusing to compile
#endif

#define DUK_USE_ARRAY_BUILTIN
#define DUK_USE_ARRAY_FASTPATH
#define DUK_USE_ARRAY_PROP_FASTPATH
#undef DUK_USE_ASSERTIONS
#define DUK_USE_AUGMENT_ERROR_CREATE
#define DUK_USE_AUGMENT_ERROR_THROW
#define DUK_USE_AVOID_PLATFORM_FUNCPTRS
#define DUK_USE_BASE64_FASTPATH
#define DUK_USE_BOOLEAN_BUILTIN
#define DUK_USE_BUFFEROBJECT_SUPPORT
#undef DUK_USE_BUFLEN16
#define DUK_USE_BYTECODE_DUMP_SUPPORT
#define DUK_USE_CACHE_ACTIVATION
#define DUK_USE_CACHE_CATCHER
#define DUK_USE_CALLSTACK_LIMIT 10000
#define DUK_USE_COMMONJS_MODULES
#define DUK_USE_COMPILER_RECLIMIT 2500
#define DUK_USE_COROUTINE_SUPPORT
#undef DUK_USE_CPP_EXCEPTIONS
#undef DUK_USE_DATAPTR16
#undef DUK_USE_DATAPTR_DEC16
#undef DUK_USE_DATAPTR_ENC16
#define DUK_USE_DATE_BUILTIN
#undef DUK_USE_DATE_FORMAT_STRING
#undef DUK_USE_DATE_GET_LOCAL_TZOFFSET
#undef DUK_USE_DATE_GET_NOW
#undef DUK_USE_DATE_PARSE_STRING
#undef DUK_USE_DATE_PRS_GETDATE
#undef DUK_USE_DEBUG
#undef DUK_USE_DEBUGGER_DUMPHEAP
#undef DUK_USE_DEBUGGER_INSPECT
#undef DUK_USE_DEBUGGER_PAUSE_UNCAUGHT
#undef DUK_USE_DEBUGGER_SUPPORT
#define DUK_USE_DEBUGGER_THROW_NOTIFY
#undef DUK_USE_DEBUGGER_TRANSPORT_TORTURE
#define DUK_USE_DEBUG_BUFSIZE 65536L
#define DUK_USE_DEBUG_LEVEL 0
#undef DUK_USE_DEBUG_WRITE
#define DUK_USE_DOUBLE_LINKED_HEAP
#define DUK_USE_DUKTAPE_BUILTIN
#define DUK_USE_ENCODING_BUILTINS
#define DUK_USE_ERRCREATE
#define DUK_USE_ERRTHROW
#define DUK_USE_ES6
#define DUK_USE_ES6_OBJECT_PROTO_PROPERTY
#define DUK_USE_ES6_OBJECT_SETPROTOTYPEOF
#define DUK_USE_ES6_PROXY
#define DUK_USE_ES6_REGEXP_SYNTAX
#define DUK_USE_ES6_UNICODE_ESCAPE
#define DUK_USE_ES7
#define DUK_USE_ES7_EXP_OPERATOR
#define DUK_USE_ES8
#define DUK_USE_ES9
#define DUK_USE_ESBC_LIMITS
#define DUK_USE_ESBC_MAX_BYTES 2147418112L
#define DUK_USE_ESBC_MAX_LINENUMBER 2147418112L
#undef DUK_USE_EXEC_FUN_LOCAL
#undef DUK_USE_EXEC_INDIRECT_BOUND_CHECK
#undef DUK_USE_EXEC_PREFER_SIZE
#define DUK_USE_EXEC_REGCONST_OPTIMIZE
#undef DUK_USE_EXEC_TIMEOUT_CHECK
#undef DUK_USE_EXPLICIT_NULL_INIT
#undef DUK_USE_EXTSTR_FREE
#undef DUK_USE_EXTSTR_INTERN_CHECK
#undef DUK_USE_FASTINT
#define DUK_USE_FAST_REFCOUNT_DEFAULT
#undef DUK_USE_FATAL_HANDLER
#define DUK_USE_FATAL_MAXLEN 128
#define DUK_USE_FINALIZER_SUPPORT
#undef DUK_USE_FINALIZER_TORTURE
#undef DUK_USE_FUNCPTR16
#undef DUK_USE_FUNCPTR_DEC16
#undef DUK_USE_FUNCPTR_ENC16
#define DUK_USE_FUNCTION_BUILTIN
#define DUK_USE_FUNC_FILENAME_PROPERTY
#define DUK_USE_FUNC_NAME_PROPERTY
#undef DUK_USE_GC_TORTURE
#undef DUK_USE_GET_MONOTONIC_TIME
#undef DUK_USE_GET_RANDOM_DOUBLE
#undef DUK_USE_GLOBAL_BINDING
#define DUK_USE_GLOBAL_BUILTIN
#undef DUK_USE_HEAPPTR16
#undef DUK_USE_HEAPPTR_DEC16
#undef DUK_USE_HEAPPTR_ENC16
#define DUK_USE_HEX_FASTPATH
#define DUK_USE_HOBJECT_ARRAY_ABANDON_LIMIT 2
#define DUK_USE_HOBJECT_ARRAY_FAST_RESIZE_LIMIT 9
#define DUK_USE_HOBJECT_ARRAY_MINGROW_ADD 16
#define DUK_USE_HOBJECT_ARRAY_MINGROW_DIVISOR 8
#define DUK_USE_HOBJECT_ENTRY_MINGROW_ADD 16
#define DUK_USE_HOBJECT_ENTRY_MINGROW_DIVISOR 8
#define DUK_USE_HOBJECT_HASH_PART
#define DUK_USE_HOBJECT_HASH_PROP_LIMIT 8
#define DUK_USE_HSTRING_ARRIDX
#define DUK_USE_HSTRING_CLEN
#undef DUK_USE_HSTRING_EXTDATA
#define DUK_USE_HSTRING_LAZY_CLEN
#define DUK_USE_HTML_COMMENTS
#define DUK_USE_IDCHAR_FASTPATH
#undef DUK_USE_INJECT_HEAP_ALLOC_ERROR
#undef DUK_USE_INTERRUPT_COUNTER
#undef DUK_USE_INTERRUPT_DEBUG_FIXUP
#define DUK_USE_JC
#define DUK_USE_JSON_BUILTIN
#define DUK_USE_JSON_DECNUMBER_FASTPATH
#define DUK_USE_JSON_DECSTRING_FASTPATH
#define DUK_USE_JSON_DEC_RECLIMIT 1000
#define DUK_USE_JSON_EATWHITE_FASTPATH
#define DUK_USE_JSON_ENC_RECLIMIT 1000
#define DUK_USE_JSON_QUOTESTRING_FASTPATH
#undef DUK_USE_JSON_STRINGIFY_FASTPATH
#define DUK_USE_JSON_SUPPORT
#define DUK_USE_JX
#define DUK_USE_LEXER_SLIDING_WINDOW
#undef DUK_USE_LIGHTFUNC_BUILTINS
#define DUK_USE_MARK_AND_SWEEP_RECLIMIT 256
#define DUK_USE_MATH_BUILTIN
#define DUK_USE_NATIVE_CALL_RECLIMIT 1000
#define DUK_USE_NONSTD_ARRAY_CONCAT_TRAILER
#define DUK_USE_NONSTD_ARRAY_MAP_TRAILER
#define DUK_USE_NONSTD_ARRAY_SPLICE_DELCOUNT
#undef DUK_USE_NONSTD_FUNC_CALLER_PROPERTY
#undef DUK_USE_NONSTD_FUNC_SOURCE_PROPERTY
#define DUK_USE_NONSTD_FUNC_STMT
#define DUK_USE_NONSTD_GETTER_KEY_ARGUMENT
#define DUK_USE_NONSTD_JSON_ESC_U2028_U2029
#define DUK_USE_NONSTD_SETTER_KEY_ARGUMENT
#define DUK_USE_NONSTD_STRING_FROMCHARCODE_32BIT
#define DUK_USE_NUMBER_BUILTIN
#define DUK_USE_OBJECT_BUILTIN
#undef DUK_USE_OBJSIZES16
#undef DUK_USE_PARANOID_ERRORS
#define DUK_USE_PC2LINE
#define DUK_USE_PERFORMANCE_BUILTIN
#undef DUK_USE_PREFER_SIZE
#undef DUK_USE_PROMISE_BUILTIN
#define DUK_USE_PROVIDE_DEFAULT_ALLOC_FUNCTIONS
#undef DUK_USE_REFCOUNT16
#define DUK_USE_REFCOUNT32
#define DUK_USE_REFERENCE_COUNTING
#define DUK_USE_REFLECT_BUILTIN
#define DUK_USE_REGEXP_CANON_BITMAP
#undef DUK_USE_REGEXP_CANON_WORKAROUND
#define DUK_USE_REGEXP_COMPILER_RECLIMIT 10000
#define DUK_USE_REGEXP_EXECUTOR_RECLIMIT 10000
#define DUK_USE_REGEXP_SUPPORT
#undef DUK_USE_ROM_GLOBAL_CLONE
#undef DUK_USE_ROM_GLOBAL_INHERIT
#undef DUK_USE_ROM_OBJECTS
#define DUK_USE_ROM_PTRCOMP_FIRST 63488L
#undef DUK_USE_ROM_STRINGS
#define DUK_USE_SECTION_B
#undef DUK_USE_SELF_TESTS
#define DUK_USE_SHEBANG_COMMENTS
#undef DUK_USE_SHUFFLE_TORTURE
#define DUK_USE_SOURCE_NONBMP
#undef DUK_USE_STRHASH16
#undef DUK_USE_STRHASH_DENSE
#define DUK_USE_STRHASH_SKIP_SHIFT 5
#define DUK_USE_STRICT_DECL
#undef DUK_USE_STRICT_UTF8_SOURCE
#define DUK_USE_STRING_BUILTIN
#undef DUK_USE_STRLEN16
#define DUK_USE_STRTAB_GROW_LIMIT 17
#define DUK_USE_STRTAB_MAXSIZE 268435456L
#define DUK_USE_STRTAB_MINSIZE 1024
#undef DUK_USE_STRTAB_PTRCOMP
#define DUK_USE_STRTAB_RESIZE_CHECK_MASK 255
#define DUK_USE_STRTAB_SHRINK_LIMIT 6
#undef DUK_USE_STRTAB_TORTURE
#undef DUK_USE_SYMBOL_BUILTIN
#define DUK_USE_TAILCALL
#define DUK_USE_TARGET_INFO "unknown"
#define DUK_USE_TRACEBACKS
#define DUK_USE_TRACEBACK_DEPTH 10
#define DUK_USE_USER_DECLARE() 
#define DUK_USE_VALSTACK_GROW_SHIFT 2
#define DUK_USE_VALSTACK_LIMIT 1000000L
#define DUK_USE_VALSTACK_SHRINK_CHECK_SHIFT 2
#define DUK_USE_VALSTACK_SHRINK_SLACK_SHIFT 4
#undef DUK_USE_VALSTACK_UNSAFE
#define DUK_USE_VERBOSE_ERRORS
#define DUK_USE_VERBOSE_EXECUTOR_ERRORS
#define DUK_USE_VOLUNTARY_GC
#define DUK_USE_ZERO_BUFFER_DATA

#if defined(DUK_COMPILING_DUKTAPE)

#if defined(DUK_USE_DATE_GET_NOW)

#elif defined(DUK_USE_DATE_NOW_GETTIMEOFDAY)
#define DUK_USE_DATE_GET_NOW(ctx)            duk_bi_date_get_now_gettimeofday()
#elif defined(DUK_USE_DATE_NOW_TIME)
#define DUK_USE_DATE_GET_NOW(ctx)            duk_bi_date_get_now_time()
#elif defined(DUK_USE_DATE_NOW_WINDOWS)
#define DUK_USE_DATE_GET_NOW(ctx)            duk_bi_date_get_now_windows()
#elif defined(DUK_USE_DATE_NOW_WINDOWS_SUBMS)
#define DUK_USE_DATE_GET_NOW(ctx)            duk_bi_date_get_now_windows_subms()
#else
#error no provider for DUK_USE_DATE_GET_NOW()
#endif

#if defined(DUK_USE_DATE_GET_LOCAL_TZOFFSET)

#elif defined(DUK_USE_DATE_TZO_GMTIME_R) || defined(DUK_USE_DATE_TZO_GMTIME_S) || defined(DUK_USE_DATE_TZO_GMTIME)
#define DUK_USE_DATE_GET_LOCAL_TZOFFSET(d)   duk_bi_date_get_local_tzoffset_gmtime((d))
#elif defined(DUK_USE_DATE_TZO_WINDOWS)
#define DUK_USE_DATE_GET_LOCAL_TZOFFSET(d)   duk_bi_date_get_local_tzoffset_windows((d))
#elif defined(DUK_USE_DATE_TZO_WINDOWS_NO_DST)
#define DUK_USE_DATE_GET_LOCAL_TZOFFSET(d)   duk_bi_date_get_local_tzoffset_windows_no_dst((d))
#else
#error no provider for DUK_USE_DATE_GET_LOCAL_TZOFFSET()
#endif

#if defined(DUK_USE_DATE_PARSE_STRING)

#elif defined(DUK_USE_DATE_PRS_STRPTIME)
#define DUK_USE_DATE_PARSE_STRING(ctx,str)   duk_bi_date_parse_string_strptime((ctx), (str))
#elif defined(DUK_USE_DATE_PRS_GETDATE)
#define DUK_USE_DATE_PARSE_STRING(ctx,str)   duk_bi_date_parse_string_getdate((ctx), (str))
#else

#endif

#if defined(DUK_USE_DATE_FORMAT_STRING)

#elif defined(DUK_USE_DATE_FMT_STRFTIME)
#define DUK_USE_DATE_FORMAT_STRING(ctx,parts,tzoffset,flags) \
	duk_bi_date_format_parts_strftime((ctx), (parts), (tzoffset), (flags))
#else

#endif

#if defined(DUK_USE_GET_MONOTONIC_TIME)

#elif defined(DUK_USE_GET_MONOTONIC_TIME_CLOCK_GETTIME)
#define DUK_USE_GET_MONOTONIC_TIME(ctx)  duk_bi_date_get_monotonic_time_clock_gettime()
#elif defined(DUK_USE_GET_MONOTONIC_TIME_WINDOWS_QPC)
#define DUK_USE_GET_MONOTONIC_TIME(ctx)  duk_bi_date_get_monotonic_time_windows_qpc()
#else

#endif

#endif  

#if defined(DUK_OPT_ASSERTIONS)
#error unsupported legacy feature option DUK_OPT_ASSERTIONS used
#endif
#if defined(DUK_OPT_BUFFEROBJECT_SUPPORT)
#error unsupported legacy feature option DUK_OPT_BUFFEROBJECT_SUPPORT used
#endif
#if defined(DUK_OPT_BUFLEN16)
#error unsupported legacy feature option DUK_OPT_BUFLEN16 used
#endif
#if defined(DUK_OPT_DATAPTR16)
#error unsupported legacy feature option DUK_OPT_DATAPTR16 used
#endif
#if defined(DUK_OPT_DATAPTR_DEC16)
#error unsupported legacy feature option DUK_OPT_DATAPTR_DEC16 used
#endif
#if defined(DUK_OPT_DATAPTR_ENC16)
#error unsupported legacy feature option DUK_OPT_DATAPTR_ENC16 used
#endif
#if defined(DUK_OPT_DDDPRINT)
#error unsupported legacy feature option DUK_OPT_DDDPRINT used
#endif
#if defined(DUK_OPT_DDPRINT)
#error unsupported legacy feature option DUK_OPT_DDPRINT used
#endif
#if defined(DUK_OPT_DEBUG)
#error unsupported legacy feature option DUK_OPT_DEBUG used
#endif
#if defined(DUK_OPT_DEBUGGER_DUMPHEAP)
#error unsupported legacy feature option DUK_OPT_DEBUGGER_DUMPHEAP used
#endif
#if defined(DUK_OPT_DEBUGGER_FWD_LOGGING)
#error unsupported legacy feature option DUK_OPT_DEBUGGER_FWD_LOGGING used
#endif
#if defined(DUK_OPT_DEBUGGER_FWD_PRINTALERT)
#error unsupported legacy feature option DUK_OPT_DEBUGGER_FWD_PRINTALERT used
#endif
#if defined(DUK_OPT_DEBUGGER_SUPPORT)
#error unsupported legacy feature option DUK_OPT_DEBUGGER_SUPPORT used
#endif
#if defined(DUK_OPT_DEBUGGER_TRANSPORT_TORTURE)
#error unsupported legacy feature option DUK_OPT_DEBUGGER_TRANSPORT_TORTURE used
#endif
#if defined(DUK_OPT_DEBUG_BUFSIZE)
#error unsupported legacy feature option DUK_OPT_DEBUG_BUFSIZE used
#endif
#if defined(DUK_OPT_DECLARE)
#error unsupported legacy feature option DUK_OPT_DECLARE used
#endif
#if defined(DUK_OPT_DEEP_C_STACK)
#error unsupported legacy feature option DUK_OPT_DEEP_C_STACK used
#endif
#if defined(DUK_OPT_DLL_BUILD)
#error unsupported legacy feature option DUK_OPT_DLL_BUILD used
#endif
#if defined(DUK_OPT_DPRINT)
#error unsupported legacy feature option DUK_OPT_DPRINT used
#endif
#if defined(DUK_OPT_DPRINT_COLORS)
#error unsupported legacy feature option DUK_OPT_DPRINT_COLORS used
#endif
#if defined(DUK_OPT_DPRINT_RDTSC)
#error unsupported legacy feature option DUK_OPT_DPRINT_RDTSC used
#endif
#if defined(DUK_OPT_EXEC_TIMEOUT_CHECK)
#error unsupported legacy feature option DUK_OPT_EXEC_TIMEOUT_CHECK used
#endif
#if defined(DUK_OPT_EXTERNAL_STRINGS)
#error unsupported legacy feature option DUK_OPT_EXTERNAL_STRINGS used
#endif
#if defined(DUK_OPT_EXTSTR_FREE)
#error unsupported legacy feature option DUK_OPT_EXTSTR_FREE used
#endif
#if defined(DUK_OPT_EXTSTR_INTERN_CHECK)
#error unsupported legacy feature option DUK_OPT_EXTSTR_INTERN_CHECK used
#endif
#if defined(DUK_OPT_FASTINT)
#error unsupported legacy feature option DUK_OPT_FASTINT used
#endif
#if defined(DUK_OPT_FORCE_ALIGN)
#error unsupported legacy feature option DUK_OPT_FORCE_ALIGN used
#endif
#if defined(DUK_OPT_FORCE_BYTEORDER)
#error unsupported legacy feature option DUK_OPT_FORCE_BYTEORDER used
#endif
#if defined(DUK_OPT_FUNCPTR16)
#error unsupported legacy feature option DUK_OPT_FUNCPTR16 used
#endif
#if defined(DUK_OPT_FUNCPTR_DEC16)
#error unsupported legacy feature option DUK_OPT_FUNCPTR_DEC16 used
#endif
#if defined(DUK_OPT_FUNCPTR_ENC16)
#error unsupported legacy feature option DUK_OPT_FUNCPTR_ENC16 used
#endif
#if defined(DUK_OPT_FUNC_NONSTD_CALLER_PROPERTY)
#error unsupported legacy feature option DUK_OPT_FUNC_NONSTD_CALLER_PROPERTY used
#endif
#if defined(DUK_OPT_FUNC_NONSTD_SOURCE_PROPERTY)
#error unsupported legacy feature option DUK_OPT_FUNC_NONSTD_SOURCE_PROPERTY used
#endif
#if defined(DUK_OPT_GC_TORTURE)
#error unsupported legacy feature option DUK_OPT_GC_TORTURE used
#endif
#if defined(DUK_OPT_HAVE_CUSTOM_H)
#error unsupported legacy feature option DUK_OPT_HAVE_CUSTOM_H used
#endif
#if defined(DUK_OPT_HEAPPTR16)
#error unsupported legacy feature option DUK_OPT_HEAPPTR16 used
#endif
#if defined(DUK_OPT_HEAPPTR_DEC16)
#error unsupported legacy feature option DUK_OPT_HEAPPTR_DEC16 used
#endif
#if defined(DUK_OPT_HEAPPTR_ENC16)
#error unsupported legacy feature option DUK_OPT_HEAPPTR_ENC16 used
#endif
#if defined(DUK_OPT_INTERRUPT_COUNTER)
#error unsupported legacy feature option DUK_OPT_INTERRUPT_COUNTER used
#endif
#if defined(DUK_OPT_JSON_STRINGIFY_FASTPATH)
#error unsupported legacy feature option DUK_OPT_JSON_STRINGIFY_FASTPATH used
#endif
#if defined(DUK_OPT_LIGHTFUNC_BUILTINS)
#error unsupported legacy feature option DUK_OPT_LIGHTFUNC_BUILTINS used
#endif
#if defined(DUK_OPT_NONSTD_FUNC_CALLER_PROPERTY)
#error unsupported legacy feature option DUK_OPT_NONSTD_FUNC_CALLER_PROPERTY used
#endif
#if defined(DUK_OPT_NONSTD_FUNC_SOURCE_PROPERTY)
#error unsupported legacy feature option DUK_OPT_NONSTD_FUNC_SOURCE_PROPERTY used
#endif
#if defined(DUK_OPT_NO_ARRAY_SPLICE_NONSTD_DELCOUNT)
#error unsupported legacy feature option DUK_OPT_NO_ARRAY_SPLICE_NONSTD_DELCOUNT used
#endif
#if defined(DUK_OPT_NO_AUGMENT_ERRORS)
#error unsupported legacy feature option DUK_OPT_NO_AUGMENT_ERRORS used
#endif
#if defined(DUK_OPT_NO_BROWSER_LIKE)
#error unsupported legacy feature option DUK_OPT_NO_BROWSER_LIKE used
#endif
#if defined(DUK_OPT_NO_BUFFEROBJECT_SUPPORT)
#error unsupported legacy feature option DUK_OPT_NO_BUFFEROBJECT_SUPPORT used
#endif
#if defined(DUK_OPT_NO_BYTECODE_DUMP_SUPPORT)
#error unsupported legacy feature option DUK_OPT_NO_BYTECODE_DUMP_SUPPORT used
#endif
#if defined(DUK_OPT_NO_COMMONJS_MODULES)
#error unsupported legacy feature option DUK_OPT_NO_COMMONJS_MODULES used
#endif
#if defined(DUK_OPT_NO_ES6_OBJECT_PROTO_PROPERTY)
#error unsupported legacy feature option DUK_OPT_NO_ES6_OBJECT_PROTO_PROPERTY used
#endif
#if defined(DUK_OPT_NO_ES6_OBJECT_SETPROTOTYPEOF)
#error unsupported legacy feature option DUK_OPT_NO_ES6_OBJECT_SETPROTOTYPEOF used
#endif
#if defined(DUK_OPT_NO_ES6_PROXY)
#error unsupported legacy feature option DUK_OPT_NO_ES6_PROXY used
#endif
#if defined(DUK_OPT_NO_FILE_IO)
#error unsupported legacy feature option DUK_OPT_NO_FILE_IO used
#endif
#if defined(DUK_OPT_NO_FUNC_STMT)
#error unsupported legacy feature option DUK_OPT_NO_FUNC_STMT used
#endif
#if defined(DUK_OPT_NO_JC)
#error unsupported legacy feature option DUK_OPT_NO_JC used
#endif
#if defined(DUK_OPT_NO_JSONC)
#error unsupported legacy feature option DUK_OPT_NO_JSONC used
#endif
#if defined(DUK_OPT_NO_JSONX)
#error unsupported legacy feature option DUK_OPT_NO_JSONX used
#endif
#if defined(DUK_OPT_NO_JX)
#error unsupported legacy feature option DUK_OPT_NO_JX used
#endif
#if defined(DUK_OPT_NO_MARK_AND_SWEEP)
#error unsupported legacy feature option DUK_OPT_NO_MARK_AND_SWEEP used
#endif
#if defined(DUK_OPT_NO_MS_STRINGTABLE_RESIZE)
#error unsupported legacy feature option DUK_OPT_NO_MS_STRINGTABLE_RESIZE used
#endif
#if defined(DUK_OPT_NO_NONSTD_ACCESSOR_KEY_ARGUMENT)
#error unsupported legacy feature option DUK_OPT_NO_NONSTD_ACCESSOR_KEY_ARGUMENT used
#endif
#if defined(DUK_OPT_NO_NONSTD_ARRAY_CONCAT_TRAILER)
#error unsupported legacy feature option DUK_OPT_NO_NONSTD_ARRAY_CONCAT_TRAILER used
#endif
#if defined(DUK_OPT_NO_NONSTD_ARRAY_MAP_TRAILER)
#error unsupported legacy feature option DUK_OPT_NO_NONSTD_ARRAY_MAP_TRAILER used
#endif
#if defined(DUK_OPT_NO_NONSTD_ARRAY_SPLICE_DELCOUNT)
#error unsupported legacy feature option DUK_OPT_NO_NONSTD_ARRAY_SPLICE_DELCOUNT used
#endif
#if defined(DUK_OPT_NO_NONSTD_FUNC_STMT)
#error unsupported legacy feature option DUK_OPT_NO_NONSTD_FUNC_STMT used
#endif
#if defined(DUK_OPT_NO_NONSTD_JSON_ESC_U2028_U2029)
#error unsupported legacy feature option DUK_OPT_NO_NONSTD_JSON_ESC_U2028_U2029 used
#endif
#if defined(DUK_OPT_NO_NONSTD_STRING_FROMCHARCODE_32BIT)
#error unsupported legacy feature option DUK_OPT_NO_NONSTD_STRING_FROMCHARCODE_32BIT used
#endif
#if defined(DUK_OPT_NO_OBJECT_ES6_PROTO_PROPERTY)
#error unsupported legacy feature option DUK_OPT_NO_OBJECT_ES6_PROTO_PROPERTY used
#endif
#if defined(DUK_OPT_NO_OBJECT_ES6_SETPROTOTYPEOF)
#error unsupported legacy feature option DUK_OPT_NO_OBJECT_ES6_SETPROTOTYPEOF used
#endif
#if defined(DUK_OPT_NO_OCTAL_SUPPORT)
#error unsupported legacy feature option DUK_OPT_NO_OCTAL_SUPPORT used
#endif
#if defined(DUK_OPT_NO_PACKED_TVAL)
#error unsupported legacy feature option DUK_OPT_NO_PACKED_TVAL used
#endif
#if defined(DUK_OPT_NO_PC2LINE)
#error unsupported legacy feature option DUK_OPT_NO_PC2LINE used
#endif
#if defined(DUK_OPT_NO_REFERENCE_COUNTING)
#error unsupported legacy feature option DUK_OPT_NO_REFERENCE_COUNTING used
#endif
#if defined(DUK_OPT_NO_REGEXP_SUPPORT)
#error unsupported legacy feature option DUK_OPT_NO_REGEXP_SUPPORT used
#endif
#if defined(DUK_OPT_NO_SECTION_B)
#error unsupported legacy feature option DUK_OPT_NO_SECTION_B used
#endif
#if defined(DUK_OPT_NO_SOURCE_NONBMP)
#error unsupported legacy feature option DUK_OPT_NO_SOURCE_NONBMP used
#endif
#if defined(DUK_OPT_NO_STRICT_DECL)
#error unsupported legacy feature option DUK_OPT_NO_STRICT_DECL used
#endif
#if defined(DUK_OPT_NO_TRACEBACKS)
#error unsupported legacy feature option DUK_OPT_NO_TRACEBACKS used
#endif
#if defined(DUK_OPT_NO_VERBOSE_ERRORS)
#error unsupported legacy feature option DUK_OPT_NO_VERBOSE_ERRORS used
#endif
#if defined(DUK_OPT_NO_VOLUNTARY_GC)
#error unsupported legacy feature option DUK_OPT_NO_VOLUNTARY_GC used
#endif
#if defined(DUK_OPT_NO_ZERO_BUFFER_DATA)
#error unsupported legacy feature option DUK_OPT_NO_ZERO_BUFFER_DATA used
#endif
#if defined(DUK_OPT_OBJSIZES16)
#error unsupported legacy feature option DUK_OPT_OBJSIZES16 used
#endif
#if defined(DUK_OPT_PANIC_HANDLER)
#error unsupported legacy feature option DUK_OPT_PANIC_HANDLER used
#endif
#if defined(DUK_OPT_REFCOUNT16)
#error unsupported legacy feature option DUK_OPT_REFCOUNT16 used
#endif
#if defined(DUK_OPT_SEGFAULT_ON_PANIC)
#error unsupported legacy feature option DUK_OPT_SEGFAULT_ON_PANIC used
#endif
#if defined(DUK_OPT_SELF_TESTS)
#error unsupported legacy feature option DUK_OPT_SELF_TESTS used
#endif
#if defined(DUK_OPT_SETJMP)
#error unsupported legacy feature option DUK_OPT_SETJMP used
#endif
#if defined(DUK_OPT_SHUFFLE_TORTURE)
#error unsupported legacy feature option DUK_OPT_SHUFFLE_TORTURE used
#endif
#if defined(DUK_OPT_SIGSETJMP)
#error unsupported legacy feature option DUK_OPT_SIGSETJMP used
#endif
#if defined(DUK_OPT_STRHASH16)
#error unsupported legacy feature option DUK_OPT_STRHASH16 used
#endif
#if defined(DUK_OPT_STRICT_UTF8_SOURCE)
#error unsupported legacy feature option DUK_OPT_STRICT_UTF8_SOURCE used
#endif
#if defined(DUK_OPT_STRLEN16)
#error unsupported legacy feature option DUK_OPT_STRLEN16 used
#endif
#if defined(DUK_OPT_STRTAB_CHAIN)
#error unsupported legacy feature option DUK_OPT_STRTAB_CHAIN used
#endif
#if defined(DUK_OPT_STRTAB_CHAIN_SIZE)
#error unsupported legacy feature option DUK_OPT_STRTAB_CHAIN_SIZE used
#endif
#if defined(DUK_OPT_TARGET_INFO)
#error unsupported legacy feature option DUK_OPT_TARGET_INFO used
#endif
#if defined(DUK_OPT_TRACEBACK_DEPTH)
#error unsupported legacy feature option DUK_OPT_TRACEBACK_DEPTH used
#endif
#if defined(DUK_OPT_UNDERSCORE_SETJMP)
#error unsupported legacy feature option DUK_OPT_UNDERSCORE_SETJMP used
#endif
#if defined(DUK_OPT_USER_INITJS)
#error unsupported legacy feature option DUK_OPT_USER_INITJS used
#endif

#if defined(DUK_USE_32BIT_PTRS)
#error unsupported config option used (option has been removed): DUK_USE_32BIT_PTRS
#endif
#if defined(DUK_USE_ALIGN_4)
#error unsupported config option used (option has been removed): DUK_USE_ALIGN_4
#endif
#if defined(DUK_USE_ALIGN_8)
#error unsupported config option used (option has been removed): DUK_USE_ALIGN_8
#endif
#if defined(DUK_USE_BROWSER_LIKE)
#error unsupported config option used (option has been removed): DUK_USE_BROWSER_LIKE
#endif
#if defined(DUK_USE_BUILTIN_INITJS)
#error unsupported config option used (option has been removed): DUK_USE_BUILTIN_INITJS
#endif
#if defined(DUK_USE_BYTEORDER_FORCED)
#error unsupported config option used (option has been removed): DUK_USE_BYTEORDER_FORCED
#endif
#if defined(DUK_USE_DATAPTR_DEC16) && !defined(DUK_USE_DATAPTR16)
#error config option DUK_USE_DATAPTR_DEC16 requires option DUK_USE_DATAPTR16 (which is missing)
#endif
#if defined(DUK_USE_DATAPTR_ENC16) && !defined(DUK_USE_DATAPTR16)
#error config option DUK_USE_DATAPTR_ENC16 requires option DUK_USE_DATAPTR16 (which is missing)
#endif
#if defined(DUK_USE_DDDPRINT)
#error unsupported config option used (option has been removed): DUK_USE_DDDPRINT
#endif
#if defined(DUK_USE_DDPRINT)
#error unsupported config option used (option has been removed): DUK_USE_DDPRINT
#endif
#if defined(DUK_USE_DEBUGGER_FWD_LOGGING)
#error unsupported config option used (option has been removed): DUK_USE_DEBUGGER_FWD_LOGGING
#endif
#if defined(DUK_USE_DEBUGGER_FWD_PRINTALERT)
#error unsupported config option used (option has been removed): DUK_USE_DEBUGGER_FWD_PRINTALERT
#endif
#if defined(DUK_USE_DEBUGGER_SUPPORT) && !defined(DUK_USE_INTERRUPT_COUNTER)
#error config option DUK_USE_DEBUGGER_SUPPORT requires option DUK_USE_INTERRUPT_COUNTER (which is missing)
#endif
#if defined(DUK_USE_DEEP_C_STACK)
#error unsupported config option used (option has been removed): DUK_USE_DEEP_C_STACK
#endif
#if defined(DUK_USE_DOUBLE_BE)
#error unsupported config option used (option has been removed): DUK_USE_DOUBLE_BE
#endif
#if defined(DUK_USE_DOUBLE_BE) && defined(DUK_USE_DOUBLE_LE)
#error config option DUK_USE_DOUBLE_BE conflicts with option DUK_USE_DOUBLE_LE (which is also defined)
#endif
#if defined(DUK_USE_DOUBLE_BE) && defined(DUK_USE_DOUBLE_ME)
#error config option DUK_USE_DOUBLE_BE conflicts with option DUK_USE_DOUBLE_ME (which is also defined)
#endif
#if defined(DUK_USE_DOUBLE_LE)
#error unsupported config option used (option has been removed): DUK_USE_DOUBLE_LE
#endif
#if defined(DUK_USE_DOUBLE_LE) && defined(DUK_USE_DOUBLE_BE)
#error config option DUK_USE_DOUBLE_LE conflicts with option DUK_USE_DOUBLE_BE (which is also defined)
#endif
#if defined(DUK_USE_DOUBLE_LE) && defined(DUK_USE_DOUBLE_ME)
#error config option DUK_USE_DOUBLE_LE conflicts with option DUK_USE_DOUBLE_ME (which is also defined)
#endif
#if defined(DUK_USE_DOUBLE_ME)
#error unsupported config option used (option has been removed): DUK_USE_DOUBLE_ME
#endif
#if defined(DUK_USE_DOUBLE_ME) && defined(DUK_USE_DOUBLE_LE)
#error config option DUK_USE_DOUBLE_ME conflicts with option DUK_USE_DOUBLE_LE (which is also defined)
#endif
#if defined(DUK_USE_DOUBLE_ME) && defined(DUK_USE_DOUBLE_BE)
#error config option DUK_USE_DOUBLE_ME conflicts with option DUK_USE_DOUBLE_BE (which is also defined)
#endif
#if defined(DUK_USE_DPRINT)
#error unsupported config option used (option has been removed): DUK_USE_DPRINT
#endif
#if defined(DUK_USE_DPRINT) && !defined(DUK_USE_DEBUG)
#error config option DUK_USE_DPRINT requires option DUK_USE_DEBUG (which is missing)
#endif
#if defined(DUK_USE_DPRINT_COLORS)
#error unsupported config option used (option has been removed): DUK_USE_DPRINT_COLORS
#endif
#if defined(DUK_USE_DPRINT_RDTSC)
#error unsupported config option used (option has been removed): DUK_USE_DPRINT_RDTSC
#endif
#if defined(DUK_USE_ES6_REGEXP_BRACES)
#error unsupported config option used (option has been removed): DUK_USE_ES6_REGEXP_BRACES
#endif
#if defined(DUK_USE_ESBC_MAX_BYTES) && !defined(DUK_USE_ESBC_LIMITS)
#error config option DUK_USE_ESBC_MAX_BYTES requires option DUK_USE_ESBC_LIMITS (which is missing)
#endif
#if defined(DUK_USE_ESBC_MAX_LINENUMBER) && !defined(DUK_USE_ESBC_LIMITS)
#error config option DUK_USE_ESBC_MAX_LINENUMBER requires option DUK_USE_ESBC_LIMITS (which is missing)
#endif
#if defined(DUK_USE_EXEC_TIMEOUT_CHECK) && !defined(DUK_USE_INTERRUPT_COUNTER)
#error config option DUK_USE_EXEC_TIMEOUT_CHECK requires option DUK_USE_INTERRUPT_COUNTER (which is missing)
#endif
#if defined(DUK_USE_EXTSTR_FREE) && !defined(DUK_USE_HSTRING_EXTDATA)
#error config option DUK_USE_EXTSTR_FREE requires option DUK_USE_HSTRING_EXTDATA (which is missing)
#endif
#if defined(DUK_USE_EXTSTR_INTERN_CHECK) && !defined(DUK_USE_HSTRING_EXTDATA)
#error config option DUK_USE_EXTSTR_INTERN_CHECK requires option DUK_USE_HSTRING_EXTDATA (which is missing)
#endif
#if defined(DUK_USE_FASTINT) && !defined(DUK_USE_64BIT_OPS)
#error config option DUK_USE_FASTINT requires option DUK_USE_64BIT_OPS (which is missing)
#endif
#if defined(DUK_USE_FILE_IO)
#error unsupported config option used (option has been removed): DUK_USE_FILE_IO
#endif
#if defined(DUK_USE_FULL_TVAL)
#error unsupported config option used (option has been removed): DUK_USE_FULL_TVAL
#endif
#if defined(DUK_USE_FUNCPTR_DEC16) && !defined(DUK_USE_FUNCPTR16)
#error config option DUK_USE_FUNCPTR_DEC16 requires option DUK_USE_FUNCPTR16 (which is missing)
#endif
#if defined(DUK_USE_FUNCPTR_ENC16) && !defined(DUK_USE_FUNCPTR16)
#error config option DUK_USE_FUNCPTR_ENC16 requires option DUK_USE_FUNCPTR16 (which is missing)
#endif
#if defined(DUK_USE_HASHBYTES_UNALIGNED_U32_ACCESS)
#error unsupported config option used (option has been removed): DUK_USE_HASHBYTES_UNALIGNED_U32_ACCESS
#endif
#if defined(DUK_USE_HEAPPTR16) && defined(DUK_USE_DEBUG)
#error config option DUK_USE_HEAPPTR16 conflicts with option DUK_USE_DEBUG (which is also defined)
#endif
#if defined(DUK_USE_HEAPPTR_DEC16) && !defined(DUK_USE_HEAPPTR16)
#error config option DUK_USE_HEAPPTR_DEC16 requires option DUK_USE_HEAPPTR16 (which is missing)
#endif
#if defined(DUK_USE_HEAPPTR_ENC16) && !defined(DUK_USE_HEAPPTR16)
#error config option DUK_USE_HEAPPTR_ENC16 requires option DUK_USE_HEAPPTR16 (which is missing)
#endif
#if defined(DUK_USE_INTEGER_BE)
#error unsupported config option used (option has been removed): DUK_USE_INTEGER_BE
#endif
#if defined(DUK_USE_INTEGER_BE) && defined(DUK_USE_INTEGER_LE)
#error config option DUK_USE_INTEGER_BE conflicts with option DUK_USE_INTEGER_LE (which is also defined)
#endif
#if defined(DUK_USE_INTEGER_BE) && defined(DUK_USE_INTEGER_ME)
#error config option DUK_USE_INTEGER_BE conflicts with option DUK_USE_INTEGER_ME (which is also defined)
#endif
#if defined(DUK_USE_INTEGER_LE)
#error unsupported config option used (option has been removed): DUK_USE_INTEGER_LE
#endif
#if defined(DUK_USE_INTEGER_LE) && defined(DUK_USE_INTEGER_BE)
#error config option DUK_USE_INTEGER_LE conflicts with option DUK_USE_INTEGER_BE (which is also defined)
#endif
#if defined(DUK_USE_INTEGER_LE) && defined(DUK_USE_INTEGER_ME)
#error config option DUK_USE_INTEGER_LE conflicts with option DUK_USE_INTEGER_ME (which is also defined)
#endif
#if defined(DUK_USE_INTEGER_ME)
#error unsupported config option used (option has been removed): DUK_USE_INTEGER_ME
#endif
#if defined(DUK_USE_INTEGER_ME) && defined(DUK_USE_INTEGER_LE)
#error config option DUK_USE_INTEGER_ME conflicts with option DUK_USE_INTEGER_LE (which is also defined)
#endif
#if defined(DUK_USE_INTEGER_ME) && defined(DUK_USE_INTEGER_BE)
#error config option DUK_USE_INTEGER_ME conflicts with option DUK_USE_INTEGER_BE (which is also defined)
#endif
#if defined(DUK_USE_MARKANDSWEEP_FINALIZER_TORTURE)
#error unsupported config option used (option has been removed): DUK_USE_MARKANDSWEEP_FINALIZER_TORTURE
#endif
#if defined(DUK_USE_MARK_AND_SWEEP)
#error unsupported config option used (option has been removed): DUK_USE_MARK_AND_SWEEP
#endif
#if defined(DUK_USE_MATH_FMAX)
#error unsupported config option used (option has been removed): DUK_USE_MATH_FMAX
#endif
#if defined(DUK_USE_MATH_FMIN)
#error unsupported config option used (option has been removed): DUK_USE_MATH_FMIN
#endif
#if defined(DUK_USE_MATH_ROUND)
#error unsupported config option used (option has been removed): DUK_USE_MATH_ROUND
#endif
#if defined(DUK_USE_MS_STRINGTABLE_RESIZE)
#error unsupported config option used (option has been removed): DUK_USE_MS_STRINGTABLE_RESIZE
#endif
#if defined(DUK_USE_NONSTD_REGEXP_DOLLAR_ESCAPE)
#error unsupported config option used (option has been removed): DUK_USE_NONSTD_REGEXP_DOLLAR_ESCAPE
#endif
#if defined(DUK_USE_NO_DOUBLE_ALIASING_SELFTEST)
#error unsupported config option used (option has been removed): DUK_USE_NO_DOUBLE_ALIASING_SELFTEST
#endif
#if defined(DUK_USE_OCTAL_SUPPORT)
#error unsupported config option used (option has been removed): DUK_USE_OCTAL_SUPPORT
#endif
#if defined(DUK_USE_PACKED_TVAL_POSSIBLE)
#error unsupported config option used (option has been removed): DUK_USE_PACKED_TVAL_POSSIBLE
#endif
#if defined(DUK_USE_PANIC_ABORT)
#error unsupported config option used (option has been removed): DUK_USE_PANIC_ABORT
#endif
#if defined(DUK_USE_PANIC_EXIT)
#error unsupported config option used (option has been removed): DUK_USE_PANIC_EXIT
#endif
#if defined(DUK_USE_PANIC_HANDLER)
#error unsupported config option used (option has been removed): DUK_USE_PANIC_HANDLER
#endif
#if defined(DUK_USE_PANIC_SEGFAULT)
#error unsupported config option used (option has been removed): DUK_USE_PANIC_SEGFAULT
#endif
#if defined(DUK_USE_POW_NETBSD_WORKAROUND)
#error unsupported config option used (option has been removed): DUK_USE_POW_NETBSD_WORKAROUND
#endif
#if defined(DUK_USE_RDTSC)
#error unsupported config option used (option has been removed): DUK_USE_RDTSC
#endif
#if defined(DUK_USE_REFZERO_FINALIZER_TORTURE)
#error unsupported config option used (option has been removed): DUK_USE_REFZERO_FINALIZER_TORTURE
#endif
#if defined(DUK_USE_ROM_GLOBAL_CLONE) && !defined(DUK_USE_ROM_STRINGS)
#error config option DUK_USE_ROM_GLOBAL_CLONE requires option DUK_USE_ROM_STRINGS (which is missing)
#endif
#if defined(DUK_USE_ROM_GLOBAL_CLONE) && !defined(DUK_USE_ROM_OBJECTS)
#error config option DUK_USE_ROM_GLOBAL_CLONE requires option DUK_USE_ROM_OBJECTS (which is missing)
#endif
#if defined(DUK_USE_ROM_GLOBAL_CLONE) && defined(DUK_USE_ROM_GLOBAL_INHERIT)
#error config option DUK_USE_ROM_GLOBAL_CLONE conflicts with option DUK_USE_ROM_GLOBAL_INHERIT (which is also defined)
#endif
#if defined(DUK_USE_ROM_GLOBAL_INHERIT) && !defined(DUK_USE_ROM_STRINGS)
#error config option DUK_USE_ROM_GLOBAL_INHERIT requires option DUK_USE_ROM_STRINGS (which is missing)
#endif
#if defined(DUK_USE_ROM_GLOBAL_INHERIT) && !defined(DUK_USE_ROM_OBJECTS)
#error config option DUK_USE_ROM_GLOBAL_INHERIT requires option DUK_USE_ROM_OBJECTS (which is missing)
#endif
#if defined(DUK_USE_ROM_GLOBAL_INHERIT) && defined(DUK_USE_ROM_GLOBAL_CLONE)
#error config option DUK_USE_ROM_GLOBAL_INHERIT conflicts with option DUK_USE_ROM_GLOBAL_CLONE (which is also defined)
#endif
#if defined(DUK_USE_ROM_OBJECTS) && !defined(DUK_USE_ROM_STRINGS)
#error config option DUK_USE_ROM_OBJECTS requires option DUK_USE_ROM_STRINGS (which is missing)
#endif
#if defined(DUK_USE_ROM_STRINGS) && !defined(DUK_USE_ROM_OBJECTS)
#error config option DUK_USE_ROM_STRINGS requires option DUK_USE_ROM_OBJECTS (which is missing)
#endif
#if defined(DUK_USE_SETJMP)
#error unsupported config option used (option has been removed): DUK_USE_SETJMP
#endif
#if defined(DUK_USE_SIGSETJMP)
#error unsupported config option used (option has been removed): DUK_USE_SIGSETJMP
#endif
#if defined(DUK_USE_STRTAB_CHAIN)
#error unsupported config option used (option has been removed): DUK_USE_STRTAB_CHAIN
#endif
#if defined(DUK_USE_STRTAB_CHAIN_SIZE)
#error unsupported config option used (option has been removed): DUK_USE_STRTAB_CHAIN_SIZE
#endif
#if defined(DUK_USE_STRTAB_CHAIN_SIZE) && !defined(DUK_USE_STRTAB_CHAIN)
#error config option DUK_USE_STRTAB_CHAIN_SIZE requires option DUK_USE_STRTAB_CHAIN (which is missing)
#endif
#if defined(DUK_USE_STRTAB_PROBE)
#error unsupported config option used (option has been removed): DUK_USE_STRTAB_PROBE
#endif
#if defined(DUK_USE_STRTAB_PTRCOMP) && !defined(DUK_USE_HEAPPTR16)
#error config option DUK_USE_STRTAB_PTRCOMP requires option DUK_USE_HEAPPTR16 (which is missing)
#endif
#if defined(DUK_USE_TAILCALL) && defined(DUK_USE_NONSTD_FUNC_CALLER_PROPERTY)
#error config option DUK_USE_TAILCALL conflicts with option DUK_USE_NONSTD_FUNC_CALLER_PROPERTY (which is also defined)
#endif
#if defined(DUK_USE_UNALIGNED_ACCESSES_POSSIBLE)
#error unsupported config option used (option has been removed): DUK_USE_UNALIGNED_ACCESSES_POSSIBLE
#endif
#if defined(DUK_USE_UNDERSCORE_SETJMP)
#error unsupported config option used (option has been removed): DUK_USE_UNDERSCORE_SETJMP
#endif
#if defined(DUK_USE_USER_INITJS)
#error unsupported config option used (option has been removed): DUK_USE_USER_INITJS
#endif

#if defined(DUK_USE_CPP_EXCEPTIONS) && !defined(__cplusplus)
#error DUK_USE_CPP_EXCEPTIONS enabled but not compiling with a C++ compiler
#endif

#if defined(DUK_USE_BYTEORDER)
#if (DUK_USE_BYTEORDER == 1)
#define DUK_USE_INTEGER_LE
#define DUK_USE_DOUBLE_LE
#elif (DUK_USE_BYTEORDER == 2)
#define DUK_USE_INTEGER_LE  
#define DUK_USE_DOUBLE_ME
#elif (DUK_USE_BYTEORDER == 3)
#define DUK_USE_INTEGER_BE
#define DUK_USE_DOUBLE_BE
#else
#error unsupported: byte order invalid
#endif  
#else
#error unsupported: byte order detection failed
#endif  

#endif  
