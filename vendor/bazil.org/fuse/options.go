package fuse

import (
	"errors"
	"strings"
)

func dummyOption(conf *mountConfig) error {
	return nil
}

type mountConfig struct {
	options          map[string]string
	maxReadahead     uint32
	initFlags        InitFlags
	osxfuseLocations []OSXFUSEPaths
}

func escapeComma(s string) string {
	s = strings.Replace(s, `\`, `\\`, -1)
	s = strings.Replace(s, `,`, `\,`, -1)
	return s
}

func (m *mountConfig) getOptions() string {
	var opts []string
	for k, v := range m.options {
		k = escapeComma(k)
		if v != "" {
			k += "=" + escapeComma(v)
		}
		opts = append(opts, k)
	}
	return strings.Join(opts, ",")
}

type mountOption func(*mountConfig) error

type MountOption mountOption

func FSName(name string) MountOption {
	return func(conf *mountConfig) error {
		conf.options["fsname"] = name
		return nil
	}
}

func Subtype(fstype string) MountOption {
	return func(conf *mountConfig) error {
		conf.options["subtype"] = fstype
		return nil
	}
}

func LocalVolume() MountOption {
	return localVolume
}

func VolumeName(name string) MountOption {
	return volumeName(name)
}

func NoAppleDouble() MountOption {
	return noAppleDouble
}

func NoAppleXattr() MountOption {
	return noAppleXattr
}

func ExclCreate() MountOption {
	return exclCreate
}

func DaemonTimeout(name string) MountOption {
	return daemonTimeout(name)
}

var ErrCannotCombineAllowOtherAndAllowRoot = errors.New("cannot combine AllowOther and AllowRoot")

func AllowOther() MountOption {
	return func(conf *mountConfig) error {
		if _, ok := conf.options["allow_root"]; ok {
			return ErrCannotCombineAllowOtherAndAllowRoot
		}
		conf.options["allow_other"] = ""
		return nil
	}
}

func AllowRoot() MountOption {
	return func(conf *mountConfig) error {
		if _, ok := conf.options["allow_other"]; ok {
			return ErrCannotCombineAllowOtherAndAllowRoot
		}
		conf.options["allow_root"] = ""
		return nil
	}
}

func AllowDev() MountOption {
	return func(conf *mountConfig) error {
		conf.options["dev"] = ""
		return nil
	}
}

func AllowSUID() MountOption {
	return func(conf *mountConfig) error {
		conf.options["suid"] = ""
		return nil
	}
}

func DefaultPermissions() MountOption {
	return func(conf *mountConfig) error {
		conf.options["default_permissions"] = ""
		return nil
	}
}

func ReadOnly() MountOption {
	return func(conf *mountConfig) error {
		conf.options["ro"] = ""
		return nil
	}
}

func MaxReadahead(n uint32) MountOption {
	return func(conf *mountConfig) error {
		conf.maxReadahead = n
		return nil
	}
}

func AsyncRead() MountOption {
	return func(conf *mountConfig) error {
		conf.initFlags |= InitAsyncRead
		return nil
	}
}

func WritebackCache() MountOption {
	return func(conf *mountConfig) error {
		conf.initFlags |= InitWritebackCache
		return nil
	}
}

type OSXFUSEPaths struct {

	DevicePrefix string

	Load string

	Mount string

	DaemonVar string
}

var (
	OSXFUSELocationV3 = OSXFUSEPaths{
		DevicePrefix: "/dev/osxfuse",
		Load:         "/Library/Filesystems/osxfuse.fs/Contents/Resources/load_osxfuse",
		Mount:        "/Library/Filesystems/osxfuse.fs/Contents/Resources/mount_osxfuse",
		DaemonVar:    "MOUNT_OSXFUSE_DAEMON_PATH",
	}
	OSXFUSELocationV2 = OSXFUSEPaths{
		DevicePrefix: "/dev/osxfuse",
		Load:         "/Library/Filesystems/osxfusefs.fs/Support/load_osxfusefs",
		Mount:        "/Library/Filesystems/osxfusefs.fs/Support/mount_osxfusefs",
		DaemonVar:    "MOUNT_FUSEFS_DAEMON_PATH",
	}
)

func OSXFUSELocations(paths ...OSXFUSEPaths) MountOption {
	return func(conf *mountConfig) error {
		if len(paths) == 0 {
			return errors.New("must specify at least one location for OSXFUSELocations")
		}

		conf.osxfuseLocations = append(conf.osxfuseLocations[:0], paths...)
		return nil
	}
}

func AllowNonEmptyMount() MountOption {
	return func(conf *mountConfig) error {
		conf.options["nonempty"] = ""
		return nil
	}
}
