#if !defined(DUK_MINIMAL_PRINTF_H_INCLUDED)
#define DUK_MINIMAL_PRINTF_H_INCLUDED

#include <stdarg.h>  
#include <stddef.h>  

extern int duk_minimal_sprintf(char *str, const char *format, ...);
extern int duk_minimal_snprintf(char *str, size_t size, const char *format, ...);
extern int duk_minimal_vsnprintf(char *str, size_t size, const char *format, va_list ap);
extern int duk_minimal_sscanf(const char *str, const char *format, ...);

#endif  
