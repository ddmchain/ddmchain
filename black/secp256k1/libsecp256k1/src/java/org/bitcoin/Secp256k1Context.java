
package org.bitcoin;

public class Secp256k1Context {
  private static final boolean enabled; 
  private static final long context; 

  static { 
      boolean isEnabled = true;
      long contextRef = -1;
      try {
          System.loadLibrary("secp256k1");
          contextRef = secp256k1_init_context();
      } catch (UnsatisfiedLinkError e) {
          System.out.println("UnsatisfiedLinkError: " + e.toString());
          isEnabled = false;
      }
      enabled = isEnabled;
      context = contextRef;
  }

  public static boolean isEnabled() {
     return enabled;
  }

  public static long getContext() {
     if(!enabled) return -1; 
     return context;
  }

  private static native long secp256k1_init_context();
}
