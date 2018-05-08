
#include <stdio.h>
#include <stdarg.h>
#include "duktape.h"
#include "duk_console.h"

static duk_ret_t duk__console_log_helper(duk_context *ctx, const char *error_name) {
	duk_idx_t i, n;
	duk_uint_t flags;

	flags = (duk_uint_t) duk_get_current_magic(ctx);

	n = duk_get_top(ctx);

	duk_get_global_string(ctx, "console");
	duk_get_prop_string(ctx, -1, "format");

	for (i = 0; i < n; i++) {
		if (duk_check_type_mask(ctx, i, DUK_TYPE_MASK_OBJECT)) {

			duk_dup(ctx, -1);  
			duk_dup(ctx, i);
			duk_call(ctx, 1);
			duk_replace(ctx, i);  
		}
	}

	duk_pop_2(ctx);

	duk_push_string(ctx, " ");
	duk_insert(ctx, 0);
	duk_join(ctx, n);

	if (error_name) {
		duk_push_error_object(ctx, DUK_ERR_ERROR, "%s", duk_require_string(ctx, -1));
		duk_push_string(ctx, "name");
		duk_push_string(ctx, error_name);
		duk_def_prop(ctx, -3, DUK_DEFPROP_FORCE | DUK_DEFPROP_HAVE_VALUE);  
		duk_get_prop_string(ctx, -1, "stack");
	}

	fprintf(stdout, "%s\n", duk_to_string(ctx, -1));
	if (flags & DUK_CONSOLE_FLUSH) {
		fflush(stdout);
	}
	return 0;
}

static duk_ret_t duk__console_assert(duk_context *ctx) {
	if (duk_to_boolean(ctx, 0)) {
		return 0;
	}
	duk_remove(ctx, 0);

	return duk__console_log_helper(ctx, "AssertionError");
}

static duk_ret_t duk__console_log(duk_context *ctx) {
	return duk__console_log_helper(ctx, NULL);
}

static duk_ret_t duk__console_trace(duk_context *ctx) {
	return duk__console_log_helper(ctx, "Trace");
}

static duk_ret_t duk__console_info(duk_context *ctx) {
	return duk__console_log_helper(ctx, NULL);
}

static duk_ret_t duk__console_warn(duk_context *ctx) {
	return duk__console_log_helper(ctx, NULL);
}

static duk_ret_t duk__console_error(duk_context *ctx) {
	return duk__console_log_helper(ctx, "Error");
}

static duk_ret_t duk__console_dir(duk_context *ctx) {

	return duk__console_log_helper(ctx, 0);
}

static void duk__console_reg_vararg_func(duk_context *ctx, duk_c_function func, const char *name, duk_uint_t flags) {
	duk_push_c_function(ctx, func, DUK_VARARGS);
	duk_push_string(ctx, "name");
	duk_push_string(ctx, name);
	duk_def_prop(ctx, -3, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_FORCE);  
	duk_set_magic(ctx, -1, (duk_int_t) flags);
	duk_put_prop_string(ctx, -2, name);
}

void duk_console_init(duk_context *ctx, duk_uint_t flags) {
	duk_push_object(ctx);

	duk_eval_string(ctx,
		"(function (E) {"
		    "return function format(v){"
		        "try{"
		            "return E('jx',v);"
		        "}catch(e){"
		            "return String(v);"  
		        "}"
		    "};"
		"})(Duktape.enc)");
	duk_put_prop_string(ctx, -2, "format");

	duk__console_reg_vararg_func(ctx, duk__console_assert, "assert", flags);
	duk__console_reg_vararg_func(ctx, duk__console_log, "log", flags);
	duk__console_reg_vararg_func(ctx, duk__console_log, "debug", flags);  
	duk__console_reg_vararg_func(ctx, duk__console_trace, "trace", flags);
	duk__console_reg_vararg_func(ctx, duk__console_info, "info", flags);
	duk__console_reg_vararg_func(ctx, duk__console_warn, "warn", flags);
	duk__console_reg_vararg_func(ctx, duk__console_error, "error", flags);
	duk__console_reg_vararg_func(ctx, duk__console_error, "exception", flags);  
	duk__console_reg_vararg_func(ctx, duk__console_dir, "dir", flags);

	duk_put_global_string(ctx, "console");

	if (flags & DUK_CONSOLE_PROXY_WRAPPER) {

		duk_peval_string_noresult(ctx,
			"(function(){"
			    "var D=function(){};"
			    "console=new Proxy(console,{"
			        "get:function(t,k){"
			            "var v=t[k];"
			            "return typeof v==='function'?v:D;"
			        "}"
			    "});"
			"})();"
		);
	}
}
