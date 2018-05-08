
#if defined HAVE_CONFIG_H
#include "libsecp256k1-config.h"
#endif

#include <stdio.h>
#include <stdlib.h>

#include <time.h>

#undef USE_ECMULT_STATIC_PRECOMPUTATION

#ifndef EXHAUSTIVE_TEST_ORDER

#define EXHAUSTIVE_TEST_ORDER 13
#define EXHAUSTIVE_TEST_LAMBDA 9   
#endif

#include "include/secp256k1.h"
#include "group.h"
#include "secp256k1.c"
#include "testrand_impl.h"

#ifdef ENABLE_MODULE_RECOVERY
#include "src/modules/recovery/main_impl.h"
#include "include/secp256k1_recovery.h"
#endif

void ge_equals_ge(const secp256k1_ge *a, const secp256k1_ge *b) {
    CHECK(a->infinity == b->infinity);
    if (a->infinity) {
        return;
    }
    CHECK(secp256k1_fe_equal_var(&a->x, &b->x));
    CHECK(secp256k1_fe_equal_var(&a->y, &b->y));
}

void ge_equals_gej(const secp256k1_ge *a, const secp256k1_gej *b) {
    secp256k1_fe z2s;
    secp256k1_fe u1, u2, s1, s2;
    CHECK(a->infinity == b->infinity);
    if (a->infinity) {
        return;
    }

    secp256k1_fe_sqr(&z2s, &b->z);
    secp256k1_fe_mul(&u1, &a->x, &z2s);
    u2 = b->x; secp256k1_fe_normalize_weak(&u2);
    secp256k1_fe_mul(&s1, &a->y, &z2s); secp256k1_fe_mul(&s1, &s1, &b->z);
    s2 = b->y; secp256k1_fe_normalize_weak(&s2);
    CHECK(secp256k1_fe_equal_var(&u1, &u2));
    CHECK(secp256k1_fe_equal_var(&s1, &s2));
}

void random_fe(secp256k1_fe *x) {
    unsigned char bin[32];
    do {
        secp256k1_rand256(bin);
        if (secp256k1_fe_set_b32(x, bin)) {
            return;
        }
    } while(1);
}

int secp256k1_nonce_function_smallint(unsigned char *nonce32, const unsigned char *msg32,
                                      const unsigned char *key32, const unsigned char *algo16,
                                      void *data, unsigned int attempt) {
    secp256k1_scalar s;
    int *idata = data;
    (void)msg32;
    (void)key32;
    (void)algo16;

    if (attempt > 0) {
        *idata = (*idata + 1) % EXHAUSTIVE_TEST_ORDER;
    }
    secp256k1_scalar_set_int(&s, *idata);
    secp256k1_scalar_get_b32(nonce32, &s);
    return 1;
}

#ifdef USE_ENDOMORPHISM
void test_exhaustive_endomorphism(const secp256k1_ge *group, int order) {
    int i;
    for (i = 0; i < order; i++) {
        secp256k1_ge res;
        secp256k1_ge_mul_lambda(&res, &group[i]);
        ge_equals_ge(&group[i * EXHAUSTIVE_TEST_LAMBDA % EXHAUSTIVE_TEST_ORDER], &res);
    }
}
#endif

void test_exhaustive_addition(const secp256k1_ge *group, const secp256k1_gej *groupj, int order) {
    int i, j;

    CHECK(secp256k1_ge_is_infinity(&group[0]));
    CHECK(secp256k1_gej_is_infinity(&groupj[0]));
    for (i = 1; i < order; i++) {
        CHECK(!secp256k1_ge_is_infinity(&group[i]));
        CHECK(!secp256k1_gej_is_infinity(&groupj[i]));
    }

    for (j = 0; j < order; j++) {
        secp256k1_fe fe_inv;
        secp256k1_fe_inv(&fe_inv, &groupj[j].z);
        for (i = 0; i < order; i++) {
            secp256k1_ge zless_gej;
            secp256k1_gej tmp;

            secp256k1_gej_add_var(&tmp, &groupj[i], &groupj[j], NULL);
            ge_equals_gej(&group[(i + j) % order], &tmp);

            if (j > 0) {
                secp256k1_gej_add_ge(&tmp, &groupj[i], &group[j]);
                ge_equals_gej(&group[(i + j) % order], &tmp);
            }

            secp256k1_gej_add_ge_var(&tmp, &groupj[i], &group[j], NULL);
            ge_equals_gej(&group[(i + j) % order], &tmp);

            zless_gej.infinity = groupj[j].infinity;
            zless_gej.x = groupj[j].x;
            zless_gej.y = groupj[j].y;
            secp256k1_gej_add_zinv_var(&tmp, &groupj[i], &zless_gej, &fe_inv);
            ge_equals_gej(&group[(i + j) % order], &tmp);
        }
    }

    for (i = 0; i < order; i++) {
        secp256k1_gej tmp;
        if (i > 0) {
            secp256k1_gej_double_nonzero(&tmp, &groupj[i], NULL);
            ge_equals_gej(&group[(2 * i) % order], &tmp);
        }
        secp256k1_gej_double_var(&tmp, &groupj[i], NULL);
        ge_equals_gej(&group[(2 * i) % order], &tmp);
    }

    for (i = 1; i < order; i++) {
        secp256k1_ge tmp;
        secp256k1_gej tmpj;
        secp256k1_ge_neg(&tmp, &group[i]);
        ge_equals_ge(&group[order - i], &tmp);
        secp256k1_gej_neg(&tmpj, &groupj[i]);
        ge_equals_gej(&group[order - i], &tmpj);
    }
}

void test_exhaustive_ecmult(const secp256k1_context *ctx, const secp256k1_ge *group, const secp256k1_gej *groupj, int order) {
    int i, j, r_log;
    for (r_log = 1; r_log < order; r_log++) {
        for (j = 0; j < order; j++) {
            for (i = 0; i < order; i++) {
                secp256k1_gej tmp;
                secp256k1_scalar na, ng;
                secp256k1_scalar_set_int(&na, i);
                secp256k1_scalar_set_int(&ng, j);

                secp256k1_ecmult(&ctx->ecmult_ctx, &tmp, &groupj[r_log], &na, &ng);
                ge_equals_gej(&group[(i * r_log + j) % order], &tmp);

                if (i > 0) {
                    secp256k1_ecmult_const(&tmp, &group[i], &ng);
                    ge_equals_gej(&group[(i * j) % order], &tmp);
                }
            }
        }
    }
}

void r_from_k(secp256k1_scalar *r, const secp256k1_ge *group, int k) {
    secp256k1_fe x;
    unsigned char x_bin[32];
    k %= EXHAUSTIVE_TEST_ORDER;
    x = group[k].x;
    secp256k1_fe_normalize(&x);
    secp256k1_fe_get_b32(x_bin, &x);
    secp256k1_scalar_set_b32(r, x_bin, NULL);
}

void test_exhaustive_verify(const secp256k1_context *ctx, const secp256k1_ge *group, int order) {
    int s, r, msg, key;
    for (s = 1; s < order; s++) {
        for (r = 1; r < order; r++) {
            for (msg = 1; msg < order; msg++) {
                for (key = 1; key < order; key++) {
                    secp256k1_ge nonconst_ge;
                    secp256k1_ecdsa_signature sig;
                    secp256k1_pubkey pk;
                    secp256k1_scalar sk_s, msg_s, r_s, s_s;
                    secp256k1_scalar s_times_k_s, msg_plus_r_times_sk_s;
                    int k, should_verify;
                    unsigned char msg32[32];

                    secp256k1_scalar_set_int(&s_s, s);
                    secp256k1_scalar_set_int(&r_s, r);
                    secp256k1_scalar_set_int(&msg_s, msg);
                    secp256k1_scalar_set_int(&sk_s, key);

                    should_verify = 0;
                    for (k = 0; k < order; k++) {
                        secp256k1_scalar check_x_s;
                        r_from_k(&check_x_s, group, k);
                        if (r_s == check_x_s) {
                            secp256k1_scalar_set_int(&s_times_k_s, k);
                            secp256k1_scalar_mul(&s_times_k_s, &s_times_k_s, &s_s);
                            secp256k1_scalar_mul(&msg_plus_r_times_sk_s, &r_s, &sk_s);
                            secp256k1_scalar_add(&msg_plus_r_times_sk_s, &msg_plus_r_times_sk_s, &msg_s);
                            should_verify |= secp256k1_scalar_eq(&s_times_k_s, &msg_plus_r_times_sk_s);
                        }
                    }

                    should_verify &= !secp256k1_scalar_is_high(&s_s);

                    secp256k1_ecdsa_signature_save(&sig, &r_s, &s_s);
                    memcpy(&nonconst_ge, &group[sk_s], sizeof(nonconst_ge));
                    secp256k1_pubkey_save(&pk, &nonconst_ge);
                    secp256k1_scalar_get_b32(msg32, &msg_s);
                    CHECK(should_verify ==
                          secp256k1_ecdsa_verify(ctx, &sig, msg32, &pk));
                }
            }
        }
    }
}

void test_exhaustive_sign(const secp256k1_context *ctx, const secp256k1_ge *group, int order) {
    int i, j, k;

    for (i = 1; i < order; i++) {  
        for (j = 1; j < order; j++) {  
            for (k = 1; k < order; k++) {  
                const int starting_k = k;
                secp256k1_ecdsa_signature sig;
                secp256k1_scalar sk, msg, r, s, expected_r;
                unsigned char sk32[32], msg32[32];
                secp256k1_scalar_set_int(&msg, i);
                secp256k1_scalar_set_int(&sk, j);
                secp256k1_scalar_get_b32(sk32, &sk);
                secp256k1_scalar_get_b32(msg32, &msg);

                secp256k1_ecdsa_sign(ctx, &sig, msg32, sk32, secp256k1_nonce_function_smallint, &k);

                secp256k1_ecdsa_signature_load(ctx, &r, &s, &sig);

                r_from_k(&expected_r, group, k);
                CHECK(r == expected_r);
                CHECK((k * s) % order == (i + r * j) % order ||
                      (k * (EXHAUSTIVE_TEST_ORDER - s)) % order == (i + r * j) % order);

                if (k < starting_k) {
                    break;
                }
            }
        }
    }

}

#ifdef ENABLE_MODULE_RECOVERY
void test_exhaustive_recovery_sign(const secp256k1_context *ctx, const secp256k1_ge *group, int order) {
    int i, j, k;

    for (i = 1; i < order; i++) {  
        for (j = 1; j < order; j++) {  
            for (k = 1; k < order; k++) {  
                const int starting_k = k;
                secp256k1_fe r_dot_y_normalized;
                secp256k1_ecdsa_recoverable_signature rsig;
                secp256k1_ecdsa_signature sig;
                secp256k1_scalar sk, msg, r, s, expected_r;
                unsigned char sk32[32], msg32[32];
                int expected_recid;
                int recid;
                secp256k1_scalar_set_int(&msg, i);
                secp256k1_scalar_set_int(&sk, j);
                secp256k1_scalar_get_b32(sk32, &sk);
                secp256k1_scalar_get_b32(msg32, &msg);

                secp256k1_ecdsa_sign_recoverable(ctx, &rsig, msg32, sk32, secp256k1_nonce_function_smallint, &k);

                secp256k1_ecdsa_recoverable_signature_load(ctx, &r, &s, &recid, &rsig);
                r_from_k(&expected_r, group, k);
                CHECK(r == expected_r);
                CHECK((k * s) % order == (i + r * j) % order ||
                      (k * (EXHAUSTIVE_TEST_ORDER - s)) % order == (i + r * j) % order);

                r_dot_y_normalized = group[k].y;
                secp256k1_fe_normalize(&r_dot_y_normalized);

                if ((k * s) % order == (i + r * j) % order) {
                    expected_recid = secp256k1_fe_is_odd(&r_dot_y_normalized) ? 1 : 0;
                } else {
                    expected_recid = secp256k1_fe_is_odd(&r_dot_y_normalized) ? 0 : 1;
                }
                CHECK(recid == expected_recid);

                secp256k1_ecdsa_recoverable_signature_convert(ctx, &sig, &rsig);
                secp256k1_ecdsa_signature_load(ctx, &r, &s, &sig);

                r_from_k(&expected_r, group, k);
                CHECK(r == expected_r);
                CHECK((k * s) % order == (i + r * j) % order ||
                      (k * (EXHAUSTIVE_TEST_ORDER - s)) % order == (i + r * j) % order);

                if (k < starting_k) {
                    break;
                }
            }
        }
    }
}

void test_exhaustive_recovery_verify(const secp256k1_context *ctx, const secp256k1_ge *group, int order) {

    int s, r, msg, key;
    for (s = 1; s < order; s++) {
        for (r = 1; r < order; r++) {
            for (msg = 1; msg < order; msg++) {
                for (key = 1; key < order; key++) {
                    secp256k1_ge nonconst_ge;
                    secp256k1_ecdsa_recoverable_signature rsig;
                    secp256k1_ecdsa_signature sig;
                    secp256k1_pubkey pk;
                    secp256k1_scalar sk_s, msg_s, r_s, s_s;
                    secp256k1_scalar s_times_k_s, msg_plus_r_times_sk_s;
                    int recid = 0;
                    int k, should_verify;
                    unsigned char msg32[32];

                    secp256k1_scalar_set_int(&s_s, s);
                    secp256k1_scalar_set_int(&r_s, r);
                    secp256k1_scalar_set_int(&msg_s, msg);
                    secp256k1_scalar_set_int(&sk_s, key);
                    secp256k1_scalar_get_b32(msg32, &msg_s);

                    should_verify = 0;
                    for (k = 0; k < order; k++) {
                        secp256k1_scalar check_x_s;
                        r_from_k(&check_x_s, group, k);
                        if (r_s == check_x_s) {
                            secp256k1_scalar_set_int(&s_times_k_s, k);
                            secp256k1_scalar_mul(&s_times_k_s, &s_times_k_s, &s_s);
                            secp256k1_scalar_mul(&msg_plus_r_times_sk_s, &r_s, &sk_s);
                            secp256k1_scalar_add(&msg_plus_r_times_sk_s, &msg_plus_r_times_sk_s, &msg_s);
                            should_verify |= secp256k1_scalar_eq(&s_times_k_s, &msg_plus_r_times_sk_s);
                        }
                    }

                    should_verify &= !secp256k1_scalar_is_high(&s_s);

                    secp256k1_ecdsa_recoverable_signature_save(&rsig, &r_s, &s_s, recid);
                    secp256k1_ecdsa_recoverable_signature_convert(ctx, &sig, &rsig);
                    memcpy(&nonconst_ge, &group[sk_s], sizeof(nonconst_ge));
                    secp256k1_pubkey_save(&pk, &nonconst_ge);
                    CHECK(should_verify ==
                          secp256k1_ecdsa_verify(ctx, &sig, msg32, &pk));
                }
            }
        }
    }
}
#endif

int main(void) {
    int i;
    secp256k1_gej groupj[EXHAUSTIVE_TEST_ORDER];
    secp256k1_ge group[EXHAUSTIVE_TEST_ORDER];

    secp256k1_context *ctx = secp256k1_context_create(SECP256K1_CONTEXT_SIGN | SECP256K1_CONTEXT_VERIFY);

    secp256k1_gej_set_infinity(&groupj[0]);
    secp256k1_ge_set_gej(&group[0], &groupj[0]);
    for (i = 1; i < EXHAUSTIVE_TEST_ORDER; i++) {

        secp256k1_fe z;
        random_fe(&z);

        secp256k1_gej_add_ge(&groupj[i], &groupj[i - 1], &secp256k1_ge_const_g);
        secp256k1_ge_set_gej(&group[i], &groupj[i]);
        secp256k1_gej_rescale(&groupj[i], &z);

        {
            secp256k1_scalar scalar_i;
            secp256k1_gej generatedj;
            secp256k1_ge generated;

            secp256k1_scalar_set_int(&scalar_i, i);
            secp256k1_ecmult_gen(&ctx->ecmult_gen_ctx, &generatedj, &scalar_i);
            secp256k1_ge_set_gej(&generated, &generatedj);

            CHECK(group[i].infinity == 0);
            CHECK(generated.infinity == 0);
            CHECK(secp256k1_fe_equal_var(&generated.x, &group[i].x));
            CHECK(secp256k1_fe_equal_var(&generated.y, &group[i].y));
        }
    }

#ifdef USE_ENDOMORPHISM
    test_exhaustive_endomorphism(group, EXHAUSTIVE_TEST_ORDER);
#endif
    test_exhaustive_addition(group, groupj, EXHAUSTIVE_TEST_ORDER);
    test_exhaustive_ecmult(ctx, group, groupj, EXHAUSTIVE_TEST_ORDER);
    test_exhaustive_sign(ctx, group, EXHAUSTIVE_TEST_ORDER);
    test_exhaustive_verify(ctx, group, EXHAUSTIVE_TEST_ORDER);

#ifdef ENABLE_MODULE_RECOVERY
    test_exhaustive_recovery_sign(ctx, group, EXHAUSTIVE_TEST_ORDER);
    test_exhaustive_recovery_verify(ctx, group, EXHAUSTIVE_TEST_ORDER);
#endif

    secp256k1_context_destroy(ctx);
    return 0;
}