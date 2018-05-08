
package monotime

import (
	"time"
	_ "unsafe" //go:linkname
)

//go:noescape
//go:linkname nanotime runtime.nanotime
func nanotime() int64

func Now() uint64 {
	return uint64(nanotime())
}

func Since(t uint64) time.Duration {
	return time.Duration(Now() - t)
}
