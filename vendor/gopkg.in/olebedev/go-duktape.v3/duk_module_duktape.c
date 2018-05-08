
#include "duktape.h"
#include "duk_module_duktape.h"

#if defined(_MSC_VER) && (_MSC_VER < 1900)
#define snprintf _snprintf
#endif

#if 0  
#define DUK__ASSERT(x) do { \
		if (!(x)) { \
			fprintf(stderr, "ASSERTION FAILED at %s:%d: " #x "\n", __FILE__, __LINE__); \
			fflush(stderr);  \
		} \
	} while (0)
#define DUK__ASSERT_TOP(ctx,val) do { \
		DUK__ASSERT(duk_get_top((ctx)) == (val)); \
	} while (0)
#else
#define DUK__ASSERT(x) do { (void) (x); } while (0)
#define DUK__ASSERT_TOP(ctx,val) do { (void) ctx; (void) (val); } while (0)
#endif

static void duk__resolve_module_id(duk_context *ctx, const char *req_id, const char *mod_id) {
	duk_uint8_t buf[DUK_COMMONJS_MODULE_ID_LIMIT];
	duk_uint8_t *p;
	duk_uint8_t *q;
	duk_uint8_t *q_last;  
	duk_int_t int_rc;

	DUK__ASSERT(req_id != NULL);

	if (mod_id != NULL && req_id[0] == '.') {
		int_rc = snprintf((char *) buf, sizeof(buf), "%s/../%s", mod_id, req_id);
	} else {
		int_rc = snprintf((char *) buf, sizeof(buf), "%s", req_id);
	}
	if (int_rc >= (duk_int_t) sizeof(buf) || int_rc < 0) {

		goto resolve_error;
	}
	DUK__ASSERT(strlen((const char *) buf) < sizeof(buf));  

	p = buf;
	q = buf;
	for (;;) {
		duk_uint_fast8_t c;

		DUK__ASSERT(p >= q);  

		q_last = q;

		c = *p++;
		if (c == 0) {
			goto resolve_error;
		} else if (c == '.') {
			c = *p++;
			if (c == '/') {

				goto eat_dup_slashes;
			}
			if (c == '.' && *p == '/') {

				p++;  
				DUK__ASSERT(q >= buf);
				if (q == buf) {
					goto resolve_error;
				}
				DUK__ASSERT(*(q - 1) == '/');
				q--;  
				for (;;) {

					DUK__ASSERT(q >= buf);
					if (q == buf) {
						break;
					}
					if (*(q - 1) == '/') {
						break;
					}
					q--;
				}
				goto eat_dup_slashes;
			}
			goto resolve_error;
		} else if (c == '/') {

			goto resolve_error;
		} else {
			for (;;) {

				*q++ = c;
				c = *p++;
				if (c == 0) {

					goto loop_done;
				} else if (c == '/') {
					*q++ = '/';
					break;
				} else {

				}
			}
		}

	 eat_dup_slashes:
		for (;;) {

			c = *p;
			if (c != '/') {
				break;
			}
			p++;
		}
	}
 loop_done:

	DUK__ASSERT(q >= buf);
	duk_push_lstring(ctx, (const char *) buf, (size_t) (q - buf));

	DUK__ASSERT(q >= q_last);
	DUK__ASSERT(q_last >= buf);
	duk_push_lstring(ctx, (const char *) q_last, (size_t) (q - q_last));
	return;

 resolve_error:
	(void) duk_type_error(ctx, "cannot resolve module id: %s", (const char *) req_id);
}

#define DUK__IDX_REQUESTED_ID   0   
#define DUK__IDX_REQUIRE        1   
#define DUK__IDX_REQUIRE_ID     2   
#define DUK__IDX_RESOLVED_ID    3   
#define DUK__IDX_LASTCOMP       4   
#define DUK__IDX_DUKTAPE        5   
#define DUK__IDX_MODLOADED      6   
#define DUK__IDX_UNDEFINED      7   
#define DUK__IDX_FRESH_REQUIRE  8   
#define DUK__IDX_EXPORTS        9   
#define DUK__IDX_MODULE         10  

static duk_ret_t duk__require(duk_context *ctx) {
	const char *str_req_id;  
	const char *str_mod_id;  
	duk_int_t pcall_rc;

	str_req_id = duk_require_string(ctx, DUK__IDX_REQUESTED_ID);
	duk_push_current_function(ctx);
	duk_get_prop_string(ctx, -1, "id");
	str_mod_id = duk_get_string(ctx, DUK__IDX_REQUIRE_ID);  
	duk__resolve_module_id(ctx, str_req_id, str_mod_id);
	str_req_id = NULL;
	str_mod_id = NULL;

	DUK__ASSERT_TOP(ctx, DUK__IDX_LASTCOMP + 1);

	duk_push_global_stash(ctx);
	duk_get_prop_string(ctx, -1, "\xff" "module:Duktape");
	duk_remove(ctx, -2);  
	duk_get_prop_string(ctx, DUK__IDX_DUKTAPE, "modLoaded");  
	duk_require_type_mask(ctx, DUK__IDX_MODLOADED, DUK_TYPE_MASK_OBJECT);
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODLOADED + 1);

	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	if (duk_get_prop(ctx, DUK__IDX_MODLOADED)) {

		duk_get_prop_string(ctx, -1, "exports");  
		return 1;
	}
	DUK__ASSERT_TOP(ctx, DUK__IDX_UNDEFINED + 1);

	duk_push_c_function(ctx, duk__require, 1 );
	duk_push_string(ctx, "name");
	duk_push_string(ctx, "require");
	duk_def_prop(ctx, DUK__IDX_FRESH_REQUIRE, DUK_DEFPROP_HAVE_VALUE);  
	duk_push_string(ctx, "id");
	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	duk_def_prop(ctx, DUK__IDX_FRESH_REQUIRE, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_SET_CONFIGURABLE);  

	duk_push_object(ctx);  
	duk_push_object(ctx);  
	duk_push_string(ctx, "exports");
	duk_dup(ctx, DUK__IDX_EXPORTS);
	duk_def_prop(ctx, DUK__IDX_MODULE, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_SET_WRITABLE | DUK_DEFPROP_SET_CONFIGURABLE);  
	duk_push_string(ctx, "id");
	duk_dup(ctx, DUK__IDX_RESOLVED_ID);  
	duk_def_prop(ctx, DUK__IDX_MODULE, DUK_DEFPROP_HAVE_VALUE);  
	duk_compact(ctx, DUK__IDX_MODULE);  
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 1);

	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	duk_dup(ctx, DUK__IDX_MODULE);
	duk_put_prop(ctx, DUK__IDX_MODLOADED);  

	duk_push_string(ctx, "(function(require,exports,module){");

	duk_get_prop_string(ctx, DUK__IDX_DUKTAPE, "modSearch");  
	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	duk_dup(ctx, DUK__IDX_FRESH_REQUIRE);
	duk_dup(ctx, DUK__IDX_EXPORTS);
	duk_dup(ctx, DUK__IDX_MODULE);  
	pcall_rc = duk_pcall(ctx, 4 );  
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 3);

	if (pcall_rc != DUK_EXEC_SUCCESS) {

		goto delete_rethrow;
	}

	if (!duk_is_string(ctx, -1)) {

		goto return_exports;
	}

	duk_push_string(ctx, "\n})");  
	duk_concat(ctx, 3);
	if (!duk_get_prop_string(ctx, DUK__IDX_MODULE, "filename")) {

		duk_pop(ctx);
		duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	}
	pcall_rc = duk_pcompile(ctx, DUK_COMPILE_EVAL);
	if (pcall_rc != DUK_EXEC_SUCCESS) {
		goto delete_rethrow;
	}
	pcall_rc = duk_pcall(ctx, 0);  
	if (pcall_rc != DUK_EXEC_SUCCESS) {
		goto delete_rethrow;
	}

	duk_push_string(ctx, "name");
	if (!duk_get_prop_string(ctx, DUK__IDX_MODULE, "name")) {

		duk_pop(ctx);
		duk_dup(ctx, DUK__IDX_LASTCOMP);
	}
	duk_def_prop(ctx, -3, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_FORCE);

	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 2);

	duk_dup(ctx, DUK__IDX_EXPORTS);  
	duk_dup(ctx, DUK__IDX_FRESH_REQUIRE);  
	duk_get_prop_string(ctx, DUK__IDX_MODULE, "exports");  
	duk_dup(ctx, DUK__IDX_MODULE);  
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 6);

	pcall_rc = duk_pcall_method(ctx, 3 );
	if (pcall_rc != DUK_EXEC_SUCCESS) {

		goto delete_rethrow;
	}

	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 2);

 return_exports:
	duk_get_prop_string(ctx, DUK__IDX_MODULE, "exports");
	duk_compact(ctx, -1);  
	return 1;  

 delete_rethrow:
	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	duk_del_prop(ctx, DUK__IDX_MODLOADED);  
	(void) duk_throw(ctx);  
	return 0;  
}

void duk_module_duktape_init(duk_context *ctx) {

	duk_push_global_stash(ctx);
	duk_get_global_string(ctx, "Duktape");
	duk_put_prop_string(ctx, -2, "\xff" "module:Duktape");
	duk_pop(ctx);

	duk_eval_string(ctx,
		"(function(req){"
		"var D=Object.defineProperty;"
		"D(req,'name',{value:'require'});"
		"D(this,'require',{value:req,writable:true,configurable:true});"
		"D(Duktape,'modLoaded',{value:Object.create(null),writable:true,configurable:true});"
		"})");
	duk_push_c_function(ctx, duk__require, 1 );
	duk_call(ctx, 1);
	duk_pop(ctx);
}

#undef DUK__ASSERT
#undef DUK__ASSERT_TOP
#undef DUK__IDX_REQUESTED_ID
#undef DUK__IDX_REQUIRE
#undef DUK__IDX_REQUIRE_ID
#undef DUK__IDX_RESOLVED_ID
#undef DUK__IDX_LASTCOMP
#undef DUK__IDX_DUKTAPE
#undef DUK__IDX_MODLOADED
#undef DUK__IDX_UNDEFINED
#undef DUK__IDX_FRESH_REQUIRE
#undef DUK__IDX_EXPORTS
#undef DUK__IDX_MODULE
