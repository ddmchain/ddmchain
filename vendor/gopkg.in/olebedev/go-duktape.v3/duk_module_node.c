
#include "duktape.h"
#include "duk_module_node.h"

#if DUK_VERSION >= 19999
static duk_int_t duk__eval_module_source(duk_context *ctx, void *udata);
#else
static duk_int_t duk__eval_module_source(duk_context *ctx);
#endif
static void duk__push_module_object(duk_context *ctx, const char *id, duk_bool_t main);

static duk_bool_t duk__get_cached_module(duk_context *ctx, const char *id) {
	duk_push_global_stash(ctx);
	(void) duk_get_prop_string(ctx, -1, "\xff" "requireCache");
	if (duk_get_prop_string(ctx, -1, id)) {
		duk_remove(ctx, -2);
		duk_remove(ctx, -2);
		return 1;
	} else {
		duk_pop_3(ctx);
		return 0;
	}
}

static void duk__put_cached_module(duk_context *ctx) {

	duk_push_global_stash(ctx);
	(void) duk_get_prop_string(ctx, -1, "\xff" "requireCache");
	duk_dup(ctx, -3);

	(void) duk_get_prop_string(ctx, -1, "id");
	duk_dup(ctx, -2);
	duk_put_prop(ctx, -4);

	duk_pop_3(ctx);  
}

static void duk__del_cached_module(duk_context *ctx, const char *id) {
	duk_push_global_stash(ctx);
	(void) duk_get_prop_string(ctx, -1, "\xff" "requireCache");
	duk_del_prop_string(ctx, -1, id);
	duk_pop_2(ctx);
}

static duk_ret_t duk__handle_require(duk_context *ctx) {

	const char *id;
	const char *parent_id;
	duk_idx_t module_idx;
	duk_idx_t stash_idx;
	duk_int_t ret;

	duk_push_global_stash(ctx);
	stash_idx = duk_normalize_index(ctx, -1);

	duk_push_current_function(ctx);
	(void) duk_get_prop_string(ctx, -1, "\xff" "moduleId");
	parent_id = duk_require_string(ctx, -1);
	(void) parent_id;  

	id = duk_require_string(ctx, 0);

	(void) duk_get_prop_string(ctx, stash_idx, "\xff" "modResolve");
	duk_dup(ctx, 0);   
	duk_dup(ctx, -3);  
	duk_call(ctx, 2);

	id = duk_require_string(ctx, -1);

	if (duk__get_cached_module(ctx, id)) {
		goto have_module;  
	}

	duk__push_module_object(ctx, id, 0 );
	duk__put_cached_module(ctx);  

	module_idx = duk_normalize_index(ctx, -1);

	(void) duk_get_prop_string(ctx, stash_idx, "\xff" "modLoad");
	duk_dup(ctx, -3);  
	(void) duk_get_prop_string(ctx, module_idx, "exports");
	duk_dup(ctx, module_idx);
	ret = duk_pcall(ctx, 3);
	if (ret != DUK_EXEC_SUCCESS) {
		duk__del_cached_module(ctx, id);
		(void) duk_throw(ctx);  
	}

	if (duk_is_string(ctx, -1)) {
		duk_int_t ret;

#if DUK_VERSION >= 19999
		ret = duk_safe_call(ctx, duk__eval_module_source, NULL, 2, 1);
#else
		ret = duk_safe_call(ctx, duk__eval_module_source, 2, 1);
#endif
		if (ret != DUK_EXEC_SUCCESS) {
			duk__del_cached_module(ctx, id);
			(void) duk_throw(ctx);  
		}
	} else if (duk_is_undefined(ctx, -1)) {
		duk_pop(ctx);
	} else {
		duk__del_cached_module(ctx, id);
		(void) duk_type_error(ctx, "invalid module load callback return value");
	}

 have_module:

	(void) duk_get_prop_string(ctx, -1, "exports");
	return 1;
}

static void duk__push_require_function(duk_context *ctx, const char *id) {
	duk_push_c_function(ctx, duk__handle_require, 1);
	duk_push_string(ctx, "name");
	duk_push_string(ctx, "require");
	duk_def_prop(ctx, -3, DUK_DEFPROP_HAVE_VALUE);
	duk_push_string(ctx, id);
	duk_put_prop_string(ctx, -2, "\xff" "moduleId");

	duk_push_global_stash(ctx);
	(void) duk_get_prop_string(ctx, -1, "\xff" "requireCache");
	duk_put_prop_string(ctx, -3, "cache");
	duk_pop(ctx);

	duk_push_global_stash(ctx);
	(void) duk_get_prop_string(ctx, -1, "\xff" "mainModule");
	duk_put_prop_string(ctx, -3, "main");
	duk_pop(ctx);
}

static void duk__push_module_object(duk_context *ctx, const char *id, duk_bool_t main) {
	duk_push_object(ctx);

	if (main) {
		duk_push_global_stash(ctx);
		duk_dup(ctx, -2);
		duk_put_prop_string(ctx, -2, "\xff" "mainModule");
		duk_pop(ctx);
	}

	duk_push_string(ctx, id);
	duk_dup(ctx, -1);
	duk_put_prop_string(ctx, -3, "filename");
	duk_put_prop_string(ctx, -2, "id");

	duk_push_object(ctx);
	duk_put_prop_string(ctx, -2, "exports");

	duk_push_false(ctx);
	duk_put_prop_string(ctx, -2, "loaded");

	duk__push_require_function(ctx, id);
	duk_put_prop_string(ctx, -2, "require");
}

#if DUK_VERSION >= 19999
static duk_int_t duk__eval_module_source(duk_context *ctx, void *udata) {
#else
static duk_int_t duk__eval_module_source(duk_context *ctx) {
#endif
	const char *src;

#if DUK_VERSION >= 19999
	(void) udata;
#endif

	duk_push_string(ctx, "(function(exports,require,module,__filename,__dirname){");
	src = duk_require_string(ctx, -2);
	duk_push_string(ctx, (src[0] == '#' && src[1] == '!') ? "//" : "");  
	duk_dup(ctx, -3);  
	duk_push_string(ctx, "\n})");  
	duk_concat(ctx, 4);

	(void) duk_get_prop_string(ctx, -3, "filename");
	duk_compile(ctx, DUK_COMPILE_EVAL);
	duk_call(ctx, 0);

	duk_push_string(ctx, "name");
	duk_push_string(ctx, "main");
	duk_def_prop(ctx, -3, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_FORCE);

	(void) duk_get_prop_string(ctx, -3, "exports");   
	(void) duk_get_prop_string(ctx, -4, "require");   
	duk_dup(ctx, -5);                                 
	(void) duk_get_prop_string(ctx, -6, "filename");  
	duk_push_undefined(ctx);                          
	duk_call(ctx, 5);

	duk_push_true(ctx);
	duk_put_prop_string(ctx, -4, "loaded");

	duk_pop_2(ctx);

	return 1;
}

duk_ret_t duk_module_node_peval_main(duk_context *ctx, const char *path) {

	duk__push_module_object(ctx, path, 1 );

	duk_dup(ctx, 0);

#if DUK_VERSION >= 19999
	return duk_safe_call(ctx, duk__eval_module_source, NULL, 2, 1);
#else
	return duk_safe_call(ctx, duk__eval_module_source, 2, 1);
#endif
}

void duk_module_node_init(duk_context *ctx) {

	duk_idx_t options_idx;

	duk_require_object_coercible(ctx, -1);  
	options_idx = duk_require_normalize_index(ctx, -1);

	duk_push_global_stash(ctx);
#if DUK_VERSION >= 19999
	duk_push_bare_object(ctx);
#else
	duk_push_object(ctx);
	duk_push_undefined(ctx);
	duk_set_prototype(ctx, -2);
#endif
	duk_put_prop_string(ctx, -2, "\xff" "requireCache");
	duk_pop(ctx);

	duk_push_global_stash(ctx);
	duk_get_prop_string(ctx, options_idx, "resolve");
	duk_require_function(ctx, -1);
	duk_put_prop_string(ctx, -2, "\xff" "modResolve");
	duk_get_prop_string(ctx, options_idx, "load");
	duk_require_function(ctx, -1);
	duk_put_prop_string(ctx, -2, "\xff" "modLoad");
	duk_pop(ctx);

	duk_push_global_stash(ctx);
	duk_push_undefined(ctx);
	duk_put_prop_string(ctx, -2, "\xff" "mainModule");
	duk_pop(ctx);

	duk_push_global_object(ctx);
	duk_push_string(ctx, "require");
	duk__push_require_function(ctx, "");
	duk_def_prop(ctx, -3, DUK_DEFPROP_HAVE_VALUE |
	                      DUK_DEFPROP_SET_WRITABLE |
	                      DUK_DEFPROP_SET_CONFIGURABLE);
	duk_pop(ctx);

	duk_pop(ctx);  
}
