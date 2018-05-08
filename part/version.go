
package params

import (
	"fmt"
)

const (
	VersionMajor = 0        
	VersionMinor = 8        
	VersionPatch = 1        
	VersionMeta  = "master" 
)

var Version = func() string {
	v := fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

func VersionWithCommit(gitCommit string) string {
	vsn := Version
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	return vsn
}
