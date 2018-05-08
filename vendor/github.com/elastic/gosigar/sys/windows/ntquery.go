// +build windows

package windows

import (
	"bytes"
	"encoding/binary"
	"io"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

const sizeofSystemProcessorPerformanceInformation = 48

type ProcessBasicInformation struct {
	ExitStatus                   uint
	PebBaseAddress               uintptr
	AffinityMask                 uint
	BasePriority                 uint
	UniqueProcessID              uint
	InheritedFromUniqueProcessID uint
}

func NtQueryProcessBasicInformation(handle syscall.Handle) (ProcessBasicInformation, error) {
	var processBasicInfo ProcessBasicInformation
	processBasicInfoPtr := (*byte)(unsafe.Pointer(&processBasicInfo))
	size := uint32(unsafe.Sizeof(processBasicInfo))
	ntStatus, _ := _NtQueryInformationProcess(handle, 0, processBasicInfoPtr, size, nil)
	if ntStatus != 0 {
		return ProcessBasicInformation{}, errors.Errorf("NtQueryInformationProcess failed, NTSTATUS=0x%X", ntStatus)
	}

	return processBasicInfo, nil
}

type SystemProcessorPerformanceInformation struct {
	IdleTime   time.Duration 
	KernelTime time.Duration 
	UserTime   time.Duration 
}

type _SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION struct {
	IdleTime   int64
	KernelTime int64
	UserTime   int64
	Reserved1  [2]int64
	Reserved2  uint32
}

func NtQuerySystemProcessorPerformanceInformation() ([]SystemProcessorPerformanceInformation, error) {

	const STATUS_SUCCESS = 0

	const systemProcessorPerformanceInformation = 8

	b := make([]byte, runtime.NumCPU()*sizeofSystemProcessorPerformanceInformation)

	var returnLength uint32
	ntStatus, _ := _NtQuerySystemInformation(systemProcessorPerformanceInformation, &b[0], uint32(len(b)), &returnLength)
	if ntStatus != STATUS_SUCCESS {
		return nil, errors.Errorf("NtQuerySystemInformation failed, NTSTATUS=0x%X, bufLength=%v, returnLength=%v", ntStatus, len(b), returnLength)
	}

	return readSystemProcessorPerformanceInformationBuffer(b)
}

func readSystemProcessorPerformanceInformationBuffer(b []byte) ([]SystemProcessorPerformanceInformation, error) {
	n := len(b) / sizeofSystemProcessorPerformanceInformation
	r := bytes.NewReader(b)

	rtn := make([]SystemProcessorPerformanceInformation, 0, n)
	for i := 0; i < n; i++ {
		_, err := r.Seek(int64(i*sizeofSystemProcessorPerformanceInformation), io.SeekStart)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to seek to cpuN=%v in buffer", i)
		}

		times := make([]uint64, 3)
		for j := range times {
			err := binary.Read(r, binary.LittleEndian, &times[j])
			if err != nil {
				return nil, errors.Wrapf(err, "failed reading cpu times for cpuN=%v", i)
			}
		}

		idleTime := time.Duration(times[0] * 100)
		kernelTime := time.Duration(times[1] * 100)
		userTime := time.Duration(times[2] * 100)

		rtn = append(rtn, SystemProcessorPerformanceInformation{
			IdleTime:   idleTime,
			KernelTime: kernelTime - idleTime, 
			UserTime:   userTime,
		})
	}

	return rtn, nil
}
