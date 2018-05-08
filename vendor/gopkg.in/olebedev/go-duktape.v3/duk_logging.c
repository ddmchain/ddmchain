
#include <stdio.h>
#include <string.h>
#include <stdarg.h>
#include "duktape.h"
#include "duk_logging.h"

#define DUK_LOGGING_FLUSH  

static const char duk__log_level_strings[] = {
	'T', 'R', 'C', 'D', 'B', 'G', 'I', 'N', 'F',
	'W', 'R', 'N', 'E', 'R', 'R', 'F', 'T', 'L'
};

static const char *duk__log_method_names[] = {
	"trace", "debug", "info", "warn", "error", "fatal"
};

static duk_ret_t duk__logger_constructor(duk_context *ctx) {
	duk_idx_t nargs;

	if (!duk_is_constructor_call(ctx)) {
		return DUK_RET_TYPE_ERROR;
	}

	nargs = duk_get_top(ctx);
	duk_set_top(ctx, 1);

	duk_push_this(ctx);

	if (nargs == 0) {

		duk_inspect_callstack_entry(ctx, -2);
		if (duk_is_object(ctx, -1)) {
			if (duk_get_prop_string(ctx, -1, "function")) {
				if (duk_get_prop_string(ctx, -1, "fileName")) {
					if (duk_is_string(ctx, -1)) {
						duk_replace(ctx, 0);
					}
				}
			}
		}

	}

	if (duk_is_string(ctx, 0)) {
		duk_dup(ctx, 0);
		duk_put_prop_string(ctx, 1, "n");
	} else {

	}

	duk_compact(ctx, 1);

	return 0;  
}

static duk_ret_t duk__logger_prototype_fmt(duk_context *ctx) {
	if (duk_get_prop_string(ctx, 0, "toLogString")) {

		duk_dup(ctx, 0);
		duk_call_method(ctx, 0);

		return 1;
	}

	duk_pop(ctx);
	duk_to_string(ctx, 0);
	return 1;
}

static duk_ret_t duk__logger_prototype_raw(duk_context *ctx) {
	const char *data;
	duk_size_t data_len;

	data = (const char *) duk_require_buffer(ctx, 0, &data_len);
	fwrite((const void *) data, 1, data_len, stderr);
	fputc((int) '\n', stderr);
#if defined(DUK_LOGGING_FLUSH)
	fflush(stderr);
#endif
	return 0;
}

static duk_ret_t duk__logger_prototype_log_shared(duk_context *ctx) {
	duk_double_t now;
	duk_time_components comp;
	duk_small_int_t entry_lev;
	duk_small_int_t logger_lev;
	duk_int_t nargs;
	duk_int_t i;
	duk_size_t tot_len;
	const duk_uint8_t *arg_str;
	duk_size_t arg_len;
	duk_uint8_t *buf, *p;
	const duk_uint8_t *q;
	duk_uint8_t date_buf[32];  
	duk_size_t date_len;
	duk_small_int_t rc;

	entry_lev = duk_get_current_magic(ctx);
	if (entry_lev < DUK_LOG_TRACE || entry_lev > DUK_LOG_FATAL) {

		return 0;
	}
	nargs = duk_get_top(ctx);

	duk_push_this(ctx);

	duk_get_prop_string(ctx, -1, "l");
	logger_lev = (duk_small_int_t) duk_get_int(ctx, -1);
	if (entry_lev < logger_lev) {
		return 0;
	}

	now = duk_get_now(ctx);
	duk_time_to_components(ctx, now, &comp);
	sprintf((char *) date_buf, "%04d-%02d-%02dT%02d:%02d:%02d.%03dZ",
	        (int) comp.year, (int) comp.month + 1, (int) comp.day,
	        (int) comp.hours, (int) comp.minutes, (int) comp.seconds,
	        (int) comp.milliseconds);

	date_len = strlen((const char *) date_buf);

	duk_get_prop_string(ctx, -2, "n");
	duk_to_string(ctx, -1);

	tot_len = 0;
	tot_len += 3 +  
	           3 +  
	           date_len +  
	           duk_get_length(ctx, -1);  

	for (i = 0; i < nargs; i++) {

		if (duk_is_object(ctx, i)) {

			duk_push_string(ctx, "fmt");
			duk_dup(ctx, i);

			rc = duk_pcall_prop(ctx, -5 , 1 );
			if (rc) {

				;
			}
			duk_replace(ctx, i);
		}
		(void) duk_to_lstring(ctx, i, &arg_len);
		tot_len++;  
		tot_len += arg_len;
	}

	buf = (duk_uint8_t *) duk_push_fixed_buffer(ctx, tot_len);
	p = buf;

	memcpy((void *) p, (const void *) date_buf, (size_t) date_len);
	p += date_len;
	*p++ = (duk_uint8_t) ' ';

	q = (const duk_uint8_t *) duk__log_level_strings + (entry_lev * 3);
	memcpy((void *) p, (const void *) q, (size_t) 3);
	p += 3;

	*p++ = (duk_uint8_t) ' ';

	arg_str = (const duk_uint8_t *) duk_get_lstring(ctx, -2, &arg_len);
	memcpy((void *) p, (const void *) arg_str, (size_t) arg_len);
	p += arg_len;

	*p++ = (duk_uint8_t) ':';

	for (i = 0; i < nargs; i++) {
		*p++ = (duk_uint8_t) ' ';

		arg_str = (const duk_uint8_t *) duk_get_lstring(ctx, i, &arg_len);
		memcpy((void *) p, (const void *) arg_str, (size_t) arg_len);
		p += arg_len;
	}

	duk_push_string(ctx, "raw");
	duk_dup(ctx, -2);

	duk_call_prop(ctx, -6, 1);  

	return 0;
}

void duk_log_va(duk_context *ctx, duk_int_t level, const char *fmt, va_list ap) {
	if (level < 0) {
		level = 0;
	} else if (level > (int) (sizeof(duk__log_method_names) / sizeof(const char *)) - 1) {
		level = (int) (sizeof(duk__log_method_names) / sizeof(const char *)) - 1;
	}

	duk_push_global_stash(ctx);
	duk_get_prop_string(ctx, -1, "\xff" "logger:constructor");  
	duk_get_prop_string(ctx, -1, "clog");
	duk_get_prop_string(ctx, -1, duk__log_method_names[level]);
	duk_dup(ctx, -2);
	duk_push_vsprintf(ctx, fmt, ap);

	duk_call_method(ctx, 1 );

	duk_pop_n(ctx, 4);
}

void duk_log(duk_context *ctx, duk_int_t level, const char *fmt, ...) {
	va_list ap;

	va_start(ap, fmt);
	duk_log_va(ctx, level, fmt, ap);
	va_end(ap);
}

void duk_logging_init(duk_context *ctx, duk_uint_t flags) {

	(void) flags;

	duk_eval_string(ctx,
		"(function(cons,prot){"
		"Object.defineProperty(Duktape,'Logger',{value:cons,writable:true,configurable:true});"
		"Object.defineProperty(cons,'prototype',{value:prot});"
		"Object.defineProperty(cons,'clog',{value:new Duktape.Logger('C'),writable:true,configurable:true});"
		"});");

	duk_push_c_function(ctx, duk__logger_constructor, DUK_VARARGS );  
	duk_push_object(ctx);  

	duk_push_string(ctx, "name");
	duk_push_string(ctx, "Logger");
	duk_def_prop(ctx, -4, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_FORCE);

	duk_dup_top(ctx);
	duk_put_prop_string(ctx, -2, "constructor");
	duk_push_int(ctx, 2);
	duk_put_prop_string(ctx, -2, "l");
	duk_push_string(ctx, "anon");
	duk_put_prop_string(ctx, -2, "n");
	duk_push_c_function(ctx, duk__logger_prototype_fmt, 1 );
	duk_put_prop_string(ctx, -2, "fmt");
	duk_push_c_function(ctx, duk__logger_prototype_raw, 1 );
	duk_put_prop_string(ctx, -2, "raw");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS );
	duk_set_magic(ctx, -1, 0);  
	duk_put_prop_string(ctx, -2, "trace");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS );
	duk_set_magic(ctx, -1, 1);  
	duk_put_prop_string(ctx, -2, "debug");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS );
	duk_set_magic(ctx, -1, 2);  
	duk_put_prop_string(ctx, -2, "info");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS );
	duk_set_magic(ctx, -1, 3);  
	duk_put_prop_string(ctx, -2, "warn");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS );
	duk_set_magic(ctx, -1, 4);  
	duk_put_prop_string(ctx, -2, "error");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS );
	duk_set_magic(ctx, -1, 5);  
	duk_put_prop_string(ctx, -2, "fatal");

	duk_call(ctx, 2);
	duk_pop(ctx);
}
