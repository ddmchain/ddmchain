
#include <stdio.h>
#include <string.h>
#include "duktape.h"
#include "duk_print_alert.h"

#define DUK_PRINT_ALERT_FLUSH   
#undef DUK_PRINT_ALERT_SMALL    

#if defined(DUK_PRINT_ALERT_SMALL)
static duk_ret_t duk__print_alert_helper(duk_context *ctx, FILE *fh) {
	duk_idx_t nargs;

	nargs = duk_get_top(ctx);

	if (nargs == 1 && duk_is_buffer(ctx, 0)) {
		buf = (const duk_uint8_t *) duk_get_buffer(ctx, 0, &sz_buf);
		fwrite((const void *) buf, 1, (size_t) sz_buf, fh);
	} else {
		duk_push_string(ctx, " ");
		duk_insert(ctx, 0);
		duk_concat(ctx, nargs);
		fprintf(fh, "%s\n", duk_require_string(ctx, -1));
	}

#if defined(DUK_PRINT_ALERT_FLUSH)
	fflush(fh);
#endif

	return 0;
}
#else

static duk_ret_t duk__print_alert_helper(duk_context *ctx, FILE *fh) {
	duk_idx_t nargs;
	const duk_uint8_t *buf;
	duk_size_t sz_buf;
	const char nl = (const char) '\n';
	duk_uint8_t buf_stack[256];

	nargs = duk_get_top(ctx);

	if (nargs == 1 && duk_is_buffer(ctx, 0)) {
		buf = (const duk_uint8_t *) duk_get_buffer(ctx, 0, &sz_buf);
	} else if (nargs > 0) {
		duk_idx_t i;
		duk_size_t sz_str;
		const duk_uint8_t *p_str;
		duk_uint8_t *p;

		sz_buf = (duk_size_t) nargs;  
		for (i = 0; i < nargs; i++) {
			(void) duk_to_lstring(ctx, i, &sz_str);
			sz_buf += sz_str;
		}

		if (sz_buf <= sizeof(buf_stack)) {
			p = (duk_uint8_t *) buf_stack;
		} else {
			p = (duk_uint8_t *) duk_push_fixed_buffer(ctx, sz_buf);
		}

		buf = (const duk_uint8_t *) p;
		for (i = 0; i < nargs; i++) {
			p_str = (const duk_uint8_t *) duk_get_lstring(ctx, i, &sz_str);
			memcpy((void *) p, (const void *) p_str, sz_str);
			p += sz_str;
			*p++ = (duk_uint8_t) (i == nargs - 1 ? '\n' : ' ');
		}
	} else {
		buf = (const duk_uint8_t *) &nl;
		sz_buf = 1;
	}

	if (sz_buf > 0) {
		fwrite((const void *) buf, 1, (size_t) sz_buf, fh);
#if defined(DUK_PRINT_ALERT_FLUSH)
		fflush(fh);
#endif
	}

	return 0;
}
#endif

static duk_ret_t duk__print(duk_context *ctx) {
	return duk__print_alert_helper(ctx, stdout);
}

static duk_ret_t duk__alert(duk_context *ctx) {
	return duk__print_alert_helper(ctx, stderr);
}

void duk_print_alert_init(duk_context *ctx, duk_uint_t flags) {
	(void) flags;  

	duk_push_global_object(ctx);
	duk_push_string(ctx, "print");
	duk_push_c_function(ctx, duk__print, DUK_VARARGS);
	duk_def_prop(ctx, -3, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_SET_WRITABLE | DUK_DEFPROP_SET_CONFIGURABLE);
	duk_push_string(ctx, "alert");
	duk_push_c_function(ctx, duk__alert, DUK_VARARGS);
	duk_def_prop(ctx, -3, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_SET_WRITABLE | DUK_DEFPROP_SET_CONFIGURABLE);
	duk_pop(ctx);
}
