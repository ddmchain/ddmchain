
package gosigar

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/elastic/gosigar/sys/windows"
	"github.com/pkg/errors"
)

type Win32_Process struct {
	CommandLine string
}

type Win32_OperatingSystem struct {
	LastBootUpTime time.Time
}

var (

	version = windows.GetWindowsVersion()

	processQueryLimitedInfoAccess = windows.PROCESS_QUERY_LIMITED_INFORMATION

	bootTime     *time.Time
	bootTimeLock sync.Mutex
)

func init() {
	if !version.IsWindowsVistaOrGreater() {

		processQueryLimitedInfoAccess = syscall.PROCESS_QUERY_INFORMATION
	}
}

func (self *LoadAverage) Get() error {
	return ErrNotImplemented{runtime.GOOS}
}

func (self *FDUsage) Get() error {
	return ErrNotImplemented{runtime.GOOS}
}

func (self *ProcEnv) Get(pid int) error {
	return ErrNotImplemented{runtime.GOOS}
}

func (self *ProcExe) Get(pid int) error {
	return ErrNotImplemented{runtime.GOOS}
}

func (self *ProcFDUsage) Get(pid int) error {
	return ErrNotImplemented{runtime.GOOS}
}

func (self *Uptime) Get() error {

	if !version.IsWindowsVistaOrGreater() {
		return ErrNotImplemented{runtime.GOOS}
	}

	bootTimeLock.Lock()
	defer bootTimeLock.Unlock()
	if bootTime == nil {
		os, err := getWin32OperatingSystem()
		if err != nil {
			return errors.Wrap(err, "failed to get boot time using WMI")
		}
		bootTime = &os.LastBootUpTime
	}

	self.Length = time.Since(*bootTime).Seconds()
	return nil
}

func (self *Mem) Get() error {
	memoryStatusEx, err := windows.GlobalMemoryStatusEx()
	if err != nil {
		return errors.Wrap(err, "GlobalMemoryStatusEx failed")
	}

	self.Total = memoryStatusEx.TotalPhys
	self.Free = memoryStatusEx.AvailPhys
	self.Used = self.Total - self.Free
	self.ActualFree = self.Free
	self.ActualUsed = self.Used
	return nil
}

func (self *Swap) Get() error {
	memoryStatusEx, err := windows.GlobalMemoryStatusEx()
	if err != nil {
		return errors.Wrap(err, "GlobalMemoryStatusEx failed")
	}

	self.Total = memoryStatusEx.TotalPageFile
	self.Free = memoryStatusEx.AvailPageFile
	self.Used = self.Total - self.Free
	return nil
}

func (self *Cpu) Get() error {
	idle, kernel, user, err := windows.GetSystemTimes()
	if err != nil {
		return errors.Wrap(err, "GetSystemTimes failed")
	}

	self.Idle = uint64(idle / time.Millisecond)
	self.Sys = uint64(kernel / time.Millisecond)
	self.User = uint64(user / time.Millisecond)
	return nil
}

func (self *CpuList) Get() error {
	cpus, err := windows.NtQuerySystemProcessorPerformanceInformation()
	if err != nil {
		return errors.Wrap(err, "NtQuerySystemProcessorPerformanceInformation failed")
	}

	self.List = make([]Cpu, 0, len(cpus))
	for _, cpu := range cpus {
		self.List = append(self.List, Cpu{
			Idle: uint64(cpu.IdleTime / time.Millisecond),
			Sys:  uint64(cpu.KernelTime / time.Millisecond),
			User: uint64(cpu.UserTime / time.Millisecond),
		})
	}
	return nil
}

func (self *FileSystemList) Get() error {
	drives, err := windows.GetLogicalDriveStrings()
	if err != nil {
		return errors.Wrap(err, "GetLogicalDriveStrings failed")
	}

	for _, drive := range drives {
		dt, err := windows.GetDriveType(drive)
		if err != nil {
			return errors.Wrapf(err, "GetDriveType failed")
		}

		self.List = append(self.List, FileSystem{
			DirName:  drive,
			DevName:  drive,
			TypeName: dt.String(),
		})
	}
	return nil
}

func (self *ProcList) Get() error {
	pids, err := windows.EnumProcesses()
	if err != nil {
		return errors.Wrap(err, "EnumProcesses failed")
	}

	self.List = make([]int, 0, len(pids))
	for _, pid := range pids {
		self.List = append(self.List, int(pid))
	}
	return nil
}

func (self *ProcState) Get(pid int) error {
	var errs []error

	var err error
	self.Name, err = getProcName(pid)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "getProcName failed"))
	}

	self.State, err = getProcStatus(pid)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "getProcStatus failed"))
	}

	self.Ppid, err = getParentPid(pid)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "getParentPid failed"))
	}

	self.Username, err = getProcCredName(pid)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "getProcCredName failed"))
	}

	if len(errs) > 0 {
		errStrs := make([]string, 0, len(errs))
		for _, e := range errs {
			errStrs = append(errStrs, e.Error())
		}
		return errors.New(strings.Join(errStrs, "; "))
	}
	return nil
}

func getProcName(pid int) (string, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return "", errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	filename, err := windows.GetProcessImageFileName(handle)
	if err != nil {
		return "", errors.Wrapf(err, "GetProcessImageFileName failed for pid=%v", pid)
	}

	return filepath.Base(filename), nil
}

func getProcStatus(pid int) (RunState, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return RunStateUnknown, errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	err = syscall.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return RunStateUnknown, errors.Wrapf(err, "GetExitCodeProcess failed for pid=%v")
	}

	if exitCode == 259 { 
		return RunStateRun, nil
	}
	return RunStateSleep, nil
}

func getParentPid(pid int) (int, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return RunStateUnknown, errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	procInfo, err := windows.NtQueryProcessBasicInformation(handle)
	if err != nil {
		return 0, errors.Wrapf(err, "NtQueryProcessBasicInformation failed for pid=%v", pid)
	}

	return int(procInfo.InheritedFromUniqueProcessID), nil
}

func getProcCredName(pid int) (string, error) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	var token syscall.Token
	err = syscall.OpenProcessToken(handle, syscall.TOKEN_QUERY, &token)
	if err != nil {
		return "", errors.Wrapf(err, "OpenProcessToken failed for pid=%v", pid)
	}

	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return "", errors.Wrapf(err, "GetTokenInformation failed for pid=%v", pid)
	}

	err = token.Close()
	if err != nil {
		return "", errors.Wrapf(err, "failed while closing process token handle for pid=%v", pid)
	}

	account, domain, _, err := tokenUser.User.Sid.LookupAccount("")
	if err != nil {
		sid, sidErr := tokenUser.User.Sid.String()
		if sidErr != nil {
			return "", errors.Wrapf(err, "failed while looking up account name for pid=%v", pid)
		}
		return "", errors.Wrapf(err, "failed while looking up account name for SID=%v of pid=%v", sid, pid)
	}

	return fmt.Sprintf(`%s\%s`, domain, account), nil
}

func (self *ProcMem) Get(pid int) error {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	counters, err := windows.GetProcessMemoryInfo(handle)
	if err != nil {
		return errors.Wrapf(err, "GetProcessMemoryInfo failed for pid=%v", pid)
	}

	self.Resident = uint64(counters.WorkingSetSize)
	self.Size = uint64(counters.PrivateUsage)
	return nil
}

func (self *ProcTime) Get(pid int) error {
	cpu, err := getProcTimes(pid)
	if err != nil {
		return err
	}

	self.StartTime = uint64(cpu.CreationTime.Nanoseconds() / 1e6)

	self.User = uint64(windows.FiletimeToDuration(&cpu.UserTime).Nanoseconds() / 1e6)
	self.Sys = uint64(windows.FiletimeToDuration(&cpu.KernelTime).Nanoseconds() / 1e6)
	self.Total = self.User + self.Sys

	return nil
}

func getProcTimes(pid int) (*syscall.Rusage, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return nil, errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	var cpu syscall.Rusage
	if err := syscall.GetProcessTimes(handle, &cpu.CreationTime, &cpu.ExitTime, &cpu.KernelTime, &cpu.UserTime); err != nil {
		return nil, errors.Wrapf(err, "GetProcessTimes failed for pid=%v", pid)
	}

	return &cpu, nil
}

func (self *ProcArgs) Get(pid int) error {

	if !version.IsWindowsVistaOrGreater() {
		return ErrNotImplemented{runtime.GOOS}
	}

	process, err := getWin32Process(int32(pid))
	if err != nil {
		return errors.Wrapf(err, "ProcArgs failed for pid=%v", pid)
	}

	self.List = []string{process.CommandLine}
	return nil
}

func (self *FileSystemUsage) Get(path string) error {
	freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes, err := windows.GetDiskFreeSpaceEx(path)
	if err != nil {
		return errors.Wrap(err, "GetDiskFreeSpaceEx failed")
	}

	self.Total = totalNumberOfBytes
	self.Free = totalNumberOfFreeBytes
	self.Used = self.Total - self.Free
	self.Avail = freeBytesAvailable
	return nil
}

func getWin32Process(pid int32) (Win32_Process, error) {
	var dst []Win32_Process
	query := fmt.Sprintf("WHERE ProcessId = %d", pid)
	q := wmi.CreateQuery(&dst, query)
	err := wmi.Query(q, &dst)
	if err != nil {
		return Win32_Process{}, fmt.Errorf("could not get Win32_Process %s: %v", query, err)
	}
	if len(dst) < 1 {
		return Win32_Process{}, fmt.Errorf("could not get Win32_Process %s: Process not found", query)
	}
	return dst[0], nil
}

func getWin32OperatingSystem() (Win32_OperatingSystem, error) {
	var dst []Win32_OperatingSystem
	q := wmi.CreateQuery(&dst, "")
	err := wmi.Query(q, &dst)
	if err != nil {
		return Win32_OperatingSystem{}, errors.Wrap(err, "wmi query for Win32_OperatingSystem failed")
	}
	if len(dst) != 1 {
		return Win32_OperatingSystem{}, errors.New("wmi query for Win32_OperatingSystem failed")
	}
	return dst[0], nil
}

func (self *Rusage) Get(who int) error {
	if who != 0 {
		return ErrNotImplemented{runtime.GOOS}
	}

	pid := os.Getpid()
	cpu, err := getProcTimes(pid)
	if err != nil {
		return err
	}

	self.Utime = windows.FiletimeToDuration(&cpu.UserTime)
	self.Stime = windows.FiletimeToDuration(&cpu.KernelTime)

	return nil
}
