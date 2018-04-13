#!/usr/bin/awk -f
# Use of this source code is governed by the Apache License 2.0
# that can be found in the COPYING file.

BEGIN {
  max = 100;
}

# Expand tabs to 4 spaces.
{
  gsub(/\t/, "    ");
}

length() > max {
  errors++;
  print FILENAME ":" FNR ": Line too long (" length() "/" max ")";
}

END {
  if (errors >= 125) {
    errors = 125;
  }
  exit errors;
}
