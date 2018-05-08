package windows

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

var (
	sizeofUint32                  = 4
	sizeofProcessEntry32          = uint32(unsafe.Sizeof(ProcessEntry32{}))
	sizeofProcessMemoryCountersEx = uint32(unsafe.Sizeof(ProcessMemoryCountersEx{}))
	sizeofMemoryStatusEx          = uint32(unsafe.Sizeof(MemoryStatusEx{}))
)

const (
	PROCESS_QUERY_LIMITED_INFORMATION uint32 = 0x1000
	PROCESS_VM_READ                   uint32 = 0x0010
)

const MAX_PATH = 260

type DriveType uint32

const (
	DRIVE_UNKNOWN DriveType = iota
	DRIVE_NO_ROOT_DIR
	DRIVE_REMOVABLE
	DRIVE_FIXED
	DRIVE_REMOTE
	DRIVE_CDROM
	DRIVE_RAMDISK
)

func (dt DriveType) String() string {
	names := map[DriveType]string{
		DRIVE_UNKNOWN:     "unknown",
		DRIVE_NO_ROOT_DIR: "invalid",
		DRIVE_REMOVABLE:   "removable",
		DRIVE_FIXED:       "fixed",
		DRIVE_REMOTE:      "remote",
		DRIVE_CDROM:       "cdrom",
		DRIVE_RAMDISK:     "ramdisk",
	}

	name, found := names[dt]
	if !found {
		return "unknown DriveType value"
	}
	return name
}

const (
	TH32CS_INHERIT      uint32 = 0x80000000 
	TH32CS_SNAPHEAPLIST uint32 = 0x00000001 
	TH32CS_SNAPMODULE   uint32 = 0x00000008 
	TH32CS_SNAPMODULE32 uint32 = 0x00000010 
	TH32CS_SNAPPROCESS  uint32 = 0x00000002 
	TH32CS_SNAPTHREAD   uint32 = 0x00000004 
)

type ProcessEntry32 struct {
	size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	exeFile           [MAX_PATH]uint16
}

func (p ProcessEntry32) ExeFile() string {
	return syscall.UTF16ToString(p.exeFile[:])
}

func (p ProcessEntry32) String() string {
	return fmt.Sprintf("{CntUsage:%v ProcessID:%v DefaultHeapID:%v ModuleID:%v "+
		"CntThreads:%v ParentProcessID:%v PriorityClassBase:%v Flags:%v ExeFile:%v",
		p.CntUsage, p.ProcessID, p.DefaultHeapID, p.ModuleID, p.CntThreads,
		p.ParentProcessID, p.PriorityClassBase, p.Flags, p.ExeFile())
}

type MemoryStatusEx struct {
	length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type ProcessMemoryCountersEx struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

func GetLogicalDriveStrings() ([]string, error) {

	bufferLength, err := _GetLogicalDriveStringsW(0, nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetLogicalDriveStringsW failed to get buffer length")
	}
	if bufferLength < 0 {
		return nil, errors.New("GetLogicalDriveStringsW returned an invalid buffer length")
	}

	buffer := make([]uint16, bufferLength)
	_, err = _GetLogicalDriveStringsW(uint32(len(buffer)), &buffer[0])
	if err != nil {
		return nil, errors.Wrap(err, "GetLogicalDriveStringsW failed")
	}

	var startIdx int
	var drivesUTF16 [][]uint16
	for i, value := range buffer {
		if value == 0 {
			drivesUTF16 = append(drivesUTF16, buffer[startIdx:i])
			startIdx = i + 1
		}
	}

	drives := make([]string, 0, len(drivesUTF16))
	for _, driveUTF16 := range drivesUTF16 {
		if len(driveUTF16) > 0 {
			drives = append(drives, syscall.UTF16ToString(driveUTF16))
		}
	}

	return drives, nil
}

func GlobalMemoryStatusEx() (MemoryStatusEx, error) {
	memoryStatusEx := MemoryStatusEx{length: sizeofMemoryStatusEx}
	err := _GlobalMemoryStatusEx(&memoryStatusEx)
	if err != nil {
		return MemoryStatusEx{}, errors.Wrap(err, "GlobalMemoryStatusEx failed")
	}

	return memoryStatusEx, nil
}

func GetProcessMemoryInfo(handle syscall.Handle) (ProcessMemoryCountersEx, error) {
	processMemoryCountersEx := ProcessMemoryCountersEx{cb: sizeofProcessMemoryCountersEx}
	err := _GetProcessMemoryInfo(handle, &processMemoryCountersEx, processMemoryCountersEx.cb)
	if err != nil {
		return ProcessMemoryCountersEx{}, errors.Wrap(err, "GetProcessMemoryInfo failed")
	}

	return processMemoryCountersEx, nil
}

func GetProcessImageFileName(handle syscall.Handle) (string, error) {
	buffer := make([]uint16, MAX_PATH)
	_, err := _GetProcessImageFileName(handle, &buffer[0], uint32(len(buffer)))
	if err != nil {
		return "", errors.Wrap(err, "GetProcessImageFileName failed")
	}

	return syscall.UTF16ToString(buffer), nil
}

func GetSystemTimes() (idle, kernel, user time.Duration, err error) {
	var idleTime, kernelTime, userTime syscall.Filetime
	err = _GetSystemTimes(&idleTime, &kernelTime, &userTime)
	if err != nil {
		return 0, 0, 0, errors.Wrap(err, "GetSystemTimes failed")
	}

	idle = FiletimeToDuration(&idleTime)
	kernel = FiletimeToDuration(&kernelTime) 
	user = FiletimeToDuration(&userTime)

	return idle, kernel - idle, user, nil
}

func FiletimeToDuration(ft *syscall.Filetime) time.Duration {
	n := int64(ft.HighDateTime)<<32 + int64(ft.LowDateTime) 
	return time.Duration(n * 100)
}

func GetDriveType(rootPathName string) (DriveType, error) {
	rootPathNamePtr, err := syscall.UTF16PtrFromString(rootPathName)
	if err != nil {
		return DRIVE_UNKNOWN, errors.Wrapf(err, "UTF16PtrFromString failed for rootPathName=%v", rootPathName)
	}

	dt, err := _GetDriveType(rootPathNamePtr)
	if err != nil {
		return DRIVE_UNKNOWN, errors.Wrapf(err, "GetDriveType failed for rootPathName=%v", rootPathName)
	}

	return dt, nil
}

func EnumProcesses() ([]uint32, error) {
	enumProcesses := func(size int) ([]uint32, error) {
		var (
			pids         = make([]uint32, size)
			sizeBytes    = len(pids) * sizeofUint32
			bytesWritten uint32
		)

		err := _EnumProcesses(&pids[0], uint32(sizeBytes), &bytesWritten)

		pidsWritten := int(bytesWritten) / sizeofUint32
		if int(bytesWritten)%sizeofUint32 != 0 || pidsWritten > len(pids) {
			return nil, errors.Errorf("EnumProcesses returned an invalid bytesWritten value of %v", bytesWritten)
		}
		pids = pids[:pidsWritten]

		return pids, err
	}

	size := 2048
	var pids []uint32
	for tries := 0; tries < 5; tries++ {
		var err error
		pids, err = enumProcesses(size)
		if err != nil {
			return nil, errors.Wrap(err, "EnumProcesses failed")
		}

		if len(pids) < size {
			break
		}

		size *= 2
	}

	return pids, nil
}

func GetDiskFreeSpaceEx(directoryName string) (freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64, err error) {
	directoryNamePtr, err := syscall.UTF16PtrFromString(directoryName)
	if err != nil {
		return 0, 0, 0, errors.Wrapf(err, "UTF16PtrFromString failed for directoryName=%v", directoryName)
	}

	err = _GetDiskFreeSpaceEx(directoryNamePtr, &freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return 0, 0, 0, err
	}

	return freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes, nil
}

func CreateToolhelp32Snapshot(flags, pid uint32) (syscall.Handle, error) {
	h, err := _CreateToolhelp32Snapshot(flags, pid)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	if h == syscall.InvalidHandle {
		return syscall.InvalidHandle, syscall.GetLastError()
	}

	return h, nil
}

func Process32First(handle syscall.Handle) (ProcessEntry32, error) {
	processEntry32 := ProcessEntry32{size: sizeofProcessEntry32}
	err := _Process32First(handle, &processEntry32)
	if err != nil {
		return ProcessEntry32{}, errors.Wrap(err, "Process32First failed")
	}

	return processEntry32, nil
}

func Process32Next(handle syscall.Handle) (ProcessEntry32, error) {
	processEntry32 := ProcessEntry32{size: sizeofProcessEntry32}
	err := _Process32Next(handle, &processEntry32)
	if err != nil {
		return ProcessEntry32{}, errors.Wrap(err, "Process32Next failed")
	}

	return processEntry32, nil
}

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

