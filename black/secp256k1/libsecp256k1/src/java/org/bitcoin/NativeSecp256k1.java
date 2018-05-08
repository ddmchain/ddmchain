
package org.bitcoin;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;

import java.math.BigInteger;
import com.google.common.base.Preconditions;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReentrantReadWriteLock;
import static org.bitcoin.NativeSecp256k1Util.*;

public class NativeSecp256k1 {

    private static final ReentrantReadWriteLock rwl = new ReentrantReadWriteLock();
    private static final Lock r = rwl.readLock();
    private static final Lock w = rwl.writeLock();
    private static ThreadLocal<ByteBuffer> nativeECDSABuffer = new ThreadLocal<ByteBuffer>();

    public static boolean verify(byte[] data, byte[] signature, byte[] pub) throws AssertFailException{
        Preconditions.checkArgument(data.length == 32 && signature.length <= 520 && pub.length <= 520);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < 520) {
            byteBuff = ByteBuffer.allocateDirect(520);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(data);
        byteBuff.put(signature);
        byteBuff.put(pub);

        byte[][] retByteArray;

        r.lock();
        try {
          return secp256k1_ecdsa_verify(byteBuff, Secp256k1Context.getContext(), signature.length, pub.length) == 1;
        } finally {
          r.unlock();
        }
    }

    public static byte[] sign(byte[] data, byte[] sec) throws AssertFailException{
        Preconditions.checkArgument(data.length == 32 && sec.length <= 32);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < 32 + 32) {
            byteBuff = ByteBuffer.allocateDirect(32 + 32);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(data);
        byteBuff.put(sec);

        byte[][] retByteArray;

        r.lock();
        try {
          retByteArray = secp256k1_ecdsa_sign(byteBuff, Secp256k1Context.getContext());
        } finally {
          r.unlock();
        }

        byte[] sigArr = retByteArray[0];
        int sigLen = new BigInteger(new byte[] { retByteArray[1][0] }).intValue();
        int retVal = new BigInteger(new byte[] { retByteArray[1][1] }).intValue();

        assertEquals(sigArr.length, sigLen, "Got bad signature length.");

        return retVal == 0 ? new byte[0] : sigArr;
    }

    public static boolean secKeyVerify(byte[] seckey) {
        Preconditions.checkArgument(seckey.length == 32);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < seckey.length) {
            byteBuff = ByteBuffer.allocateDirect(seckey.length);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(seckey);

        r.lock();
        try {
          return secp256k1_ec_seckey_verify(byteBuff,Secp256k1Context.getContext()) == 1;
        } finally {
          r.unlock();
        }
    }

    public static byte[] computePubkey(byte[] seckey) throws AssertFailException{
        Preconditions.checkArgument(seckey.length == 32);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < seckey.length) {
            byteBuff = ByteBuffer.allocateDirect(seckey.length);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(seckey);

        byte[][] retByteArray;

        r.lock();
        try {
          retByteArray = secp256k1_ec_pubkey_create(byteBuff, Secp256k1Context.getContext());
        } finally {
          r.unlock();
        }

        byte[] pubArr = retByteArray[0];
        int pubLen = new BigInteger(new byte[] { retByteArray[1][0] }).intValue();
        int retVal = new BigInteger(new byte[] { retByteArray[1][1] }).intValue();

        assertEquals(pubArr.length, pubLen, "Got bad pubkey length.");

        return retVal == 0 ? new byte[0]: pubArr;
    }

    public static synchronized void cleanup() {
        w.lock();
        try {
          secp256k1_destroy_context(Secp256k1Context.getContext());
        } finally {
          w.unlock();
        }
    }

    public static long cloneContext() {
       r.lock();
       try {
        return secp256k1_ctx_clone(Secp256k1Context.getContext());
       } finally { r.unlock(); }
    }

    public static byte[] privKeyTweakMul(byte[] privkey, byte[] tweak) throws AssertFailException{
        Preconditions.checkArgument(privkey.length == 32);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < privkey.length + tweak.length) {
            byteBuff = ByteBuffer.allocateDirect(privkey.length + tweak.length);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(privkey);
        byteBuff.put(tweak);

        byte[][] retByteArray;
        r.lock();
        try {
          retByteArray = secp256k1_privkey_tweak_mul(byteBuff,Secp256k1Context.getContext());
        } finally {
          r.unlock();
        }

        byte[] privArr = retByteArray[0];

        int privLen = (byte) new BigInteger(new byte[] { retByteArray[1][0] }).intValue() & 0xFF;
        int retVal = new BigInteger(new byte[] { retByteArray[1][1] }).intValue();

        assertEquals(privArr.length, privLen, "Got bad pubkey length.");

        assertEquals(retVal, 1, "Failed return value check.");

        return privArr;
    }

    public static byte[] privKeyTweakAdd(byte[] privkey, byte[] tweak) throws AssertFailException{
        Preconditions.checkArgument(privkey.length == 32);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < privkey.length + tweak.length) {
            byteBuff = ByteBuffer.allocateDirect(privkey.length + tweak.length);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(privkey);
        byteBuff.put(tweak);

        byte[][] retByteArray;
        r.lock();
        try {
          retByteArray = secp256k1_privkey_tweak_add(byteBuff,Secp256k1Context.getContext());
        } finally {
          r.unlock();
        }

        byte[] privArr = retByteArray[0];

        int privLen = (byte) new BigInteger(new byte[] { retByteArray[1][0] }).intValue() & 0xFF;
        int retVal = new BigInteger(new byte[] { retByteArray[1][1] }).intValue();

        assertEquals(privArr.length, privLen, "Got bad pubkey length.");

        assertEquals(retVal, 1, "Failed return value check.");

        return privArr;
    }

    public static byte[] pubKeyTweakAdd(byte[] pubkey, byte[] tweak) throws AssertFailException{
        Preconditions.checkArgument(pubkey.length == 33 || pubkey.length == 65);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < pubkey.length + tweak.length) {
            byteBuff = ByteBuffer.allocateDirect(pubkey.length + tweak.length);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(pubkey);
        byteBuff.put(tweak);

        byte[][] retByteArray;
        r.lock();
        try {
          retByteArray = secp256k1_pubkey_tweak_add(byteBuff,Secp256k1Context.getContext(), pubkey.length);
        } finally {
          r.unlock();
        }

        byte[] pubArr = retByteArray[0];

        int pubLen = (byte) new BigInteger(new byte[] { retByteArray[1][0] }).intValue() & 0xFF;
        int retVal = new BigInteger(new byte[] { retByteArray[1][1] }).intValue();

        assertEquals(pubArr.length, pubLen, "Got bad pubkey length.");

        assertEquals(retVal, 1, "Failed return value check.");

        return pubArr;
    }

    public static byte[] pubKeyTweakMul(byte[] pubkey, byte[] tweak) throws AssertFailException{
        Preconditions.checkArgument(pubkey.length == 33 || pubkey.length == 65);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < pubkey.length + tweak.length) {
            byteBuff = ByteBuffer.allocateDirect(pubkey.length + tweak.length);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(pubkey);
        byteBuff.put(tweak);

        byte[][] retByteArray;
        r.lock();
        try {
          retByteArray = secp256k1_pubkey_tweak_mul(byteBuff,Secp256k1Context.getContext(), pubkey.length);
        } finally {
          r.unlock();
        }

        byte[] pubArr = retByteArray[0];

        int pubLen = (byte) new BigInteger(new byte[] { retByteArray[1][0] }).intValue() & 0xFF;
        int retVal = new BigInteger(new byte[] { retByteArray[1][1] }).intValue();

        assertEquals(pubArr.length, pubLen, "Got bad pubkey length.");

        assertEquals(retVal, 1, "Failed return value check.");

        return pubArr;
    }

    public static byte[] createECDHSecret(byte[] seckey, byte[] pubkey) throws AssertFailException{
        Preconditions.checkArgument(seckey.length <= 32 && pubkey.length <= 65);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < 32 + pubkey.length) {
            byteBuff = ByteBuffer.allocateDirect(32 + pubkey.length);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(seckey);
        byteBuff.put(pubkey);

        byte[][] retByteArray;
        r.lock();
        try {
          retByteArray = secp256k1_ecdh(byteBuff, Secp256k1Context.getContext(), pubkey.length);
        } finally {
          r.unlock();
        }

        byte[] resArr = retByteArray[0];
        int retVal = new BigInteger(new byte[] { retByteArray[1][0] }).intValue();

        assertEquals(resArr.length, 32, "Got bad result length.");
        assertEquals(retVal, 1, "Failed return value check.");

        return resArr;
    }

    public static synchronized boolean randomize(byte[] seed) throws AssertFailException{
        Preconditions.checkArgument(seed.length == 32 || seed == null);

        ByteBuffer byteBuff = nativeECDSABuffer.get();
        if (byteBuff == null || byteBuff.capacity() < seed.length) {
            byteBuff = ByteBuffer.allocateDirect(seed.length);
            byteBuff.order(ByteOrder.nativeOrder());
            nativeECDSABuffer.set(byteBuff);
        }
        byteBuff.rewind();
        byteBuff.put(seed);

        w.lock();
        try {
          return secp256k1_context_randomize(byteBuff, Secp256k1Context.getContext()) == 1;
        } finally {
          w.unlock();
        }
    }

    private static native long secp256k1_ctx_clone(long context);

    private static native int secp256k1_context_randomize(ByteBuffer byteBuff, long context);

    private static native byte[][] secp256k1_privkey_tweak_add(ByteBuffer byteBuff, long context);

    private static native byte[][] secp256k1_privkey_tweak_mul(ByteBuffer byteBuff, long context);

    private static native byte[][] secp256k1_pubkey_tweak_add(ByteBuffer byteBuff, long context, int pubLen);

    private static native byte[][] secp256k1_pubkey_tweak_mul(ByteBuffer byteBuff, long context, int pubLen);

    private static native void secp256k1_destroy_context(long context);

    private static native int secp256k1_ecdsa_verify(ByteBuffer byteBuff, long context, int sigLen, int pubLen);

    private static native byte[][] secp256k1_ecdsa_sign(ByteBuffer byteBuff, long context);

    private static native int secp256k1_ec_seckey_verify(ByteBuffer byteBuff, long context);

    private static native byte[][] secp256k1_ec_pubkey_create(ByteBuffer byteBuff, long context);

    private static native byte[][] secp256k1_ec_pubkey_parse(ByteBuffer byteBuff, long context, int inputLen);

    private static native byte[][] secp256k1_ecdh(ByteBuffer byteBuff, long context, int inputLen);

}
