
// +build darwin,!kqueue

package notify

/*
#include <CoreServices/CoreServices.h>

typedef void (*CFRunLoopPerformCallBack)(void*);

void gosource(void *);
void gostream(uintptr_t, uintptr_t, size_t, uintptr_t, uintptr_t, uintptr_t);

static FSEventStreamRef EventStreamCreate(FSEventStreamContext * context, uintptr_t info, CFArrayRef paths, FSEventStreamEventId since, CFTimeInterval latency, FSEventStreamCreateFlags flags) {
	context->info = (void*) info;
	return FSEventStreamCreate(NULL, (FSEventStreamCallback) gostream, context, paths, since, latency, flags);
}

#cgo LDFLAGS: -framework CoreServices
*/
import "C"

import (
	"errors"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

var nilstream C.FSEventStreamRef

var (
	latency C.CFTimeInterval
	flags   = C.FSEventStreamCreateFlags(C.kFSEventStreamCreateFlagFileEvents | C.kFSEventStreamCreateFlagNoDefer)
	since   = uint64(C.FSEventsGetCurrentEventId())
)

var runloop C.CFRunLoopRef 
var wg sync.WaitGroup      

var source = C.CFRunLoopSourceCreate(nil, 0, &C.CFRunLoopSourceContext{
	perform: (C.CFRunLoopPerformCallBack)(C.gosource),
})

var (
	errCreate = os.NewSyscallError("FSEventStreamCreate", errors.New("NULL"))
	errStart  = os.NewSyscallError("FSEventStreamStart", errors.New("false"))
)

func init() {
	wg.Add(1)
	go func() {

		runtime.LockOSThread()
		runloop = C.CFRunLoopGetCurrent()
		C.CFRunLoopAddSource(runloop, source, C.kCFRunLoopDefaultMode)
		C.CFRunLoopRun()
		panic("runloop has just unexpectedly stopped")
	}()
	C.CFRunLoopSourceSignal(source)
}

//export gosource
func gosource(unsafe.Pointer) {
	wg.Done()
}

//export gostream
func gostream(_, info uintptr, n C.size_t, paths, flags, ids uintptr) {
	const (
		offchar = unsafe.Sizeof((*C.char)(nil))
		offflag = unsafe.Sizeof(C.FSEventStreamEventFlags(0))
		offid   = unsafe.Sizeof(C.FSEventStreamEventId(0))
	)
	if n == 0 {
		return
	}
	ev := make([]FSEvent, 0, int(n))
	for i := uintptr(0); i < uintptr(n); i++ {
		switch flags := *(*uint32)(unsafe.Pointer((flags + i*offflag))); {
		case flags&uint32(FSEventsEventIdsWrapped) != 0:
			atomic.StoreUint64(&since, uint64(C.FSEventsGetCurrentEventId()))
		default:
			ev = append(ev, FSEvent{
				Path:  C.GoString(*(**C.char)(unsafe.Pointer(paths + i*offchar))),
				Flags: flags,
				ID:    *(*uint64)(unsafe.Pointer(ids + i*offid)),
			})
		}

	}
	streamFuncs.get(info)(ev)
}

type streamFunc func([]FSEvent)

var streamFuncs = streamFuncRegistry{m: map[uintptr]streamFunc{}}

type streamFuncRegistry struct {
	mu sync.Mutex
	m  map[uintptr]streamFunc
	i  uintptr
}

func (r *streamFuncRegistry) get(id uintptr) streamFunc {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.m[id]
}

func (r *streamFuncRegistry) add(fn streamFunc) uintptr {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.i++
	r.m[r.i] = fn
	return r.i
}

func (r *streamFuncRegistry) delete(id uintptr) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.m, id)
}

type stream struct {
	path string
	ref  C.FSEventStreamRef
	info uintptr
}

func newStream(path string, fn streamFunc) *stream {
	return &stream{
		path: path,
		info: streamFuncs.add(fn),
	}
}

func (s *stream) Start() error {
	if s.ref != nilstream {
		return nil
	}
	wg.Wait()
	p := C.CFStringCreateWithCStringNoCopy(nil, C.CString(s.path), C.kCFStringEncodingUTF8, nil)
	path := C.CFArrayCreate(nil, (*unsafe.Pointer)(unsafe.Pointer(&p)), 1, nil)
	ctx := C.FSEventStreamContext{}
	ref := C.EventStreamCreate(&ctx, C.uintptr_t(s.info), path, C.FSEventStreamEventId(atomic.LoadUint64(&since)), latency, flags)
	if ref == nilstream {
		return errCreate
	}
	C.FSEventStreamScheduleWithRunLoop(ref, runloop, C.kCFRunLoopDefaultMode)
	if C.FSEventStreamStart(ref) == C.Boolean(0) {
		C.FSEventStreamInvalidate(ref)
		return errStart
	}
	C.CFRunLoopWakeUp(runloop)
	s.ref = ref
	return nil
}

func (s *stream) Stop() {
	if s.ref == nilstream {
		return
	}
	wg.Wait()
	C.FSEventStreamStop(s.ref)
	C.FSEventStreamInvalidate(s.ref)
	C.CFRunLoopWakeUp(runloop)
	s.ref = nilstream
	streamFuncs.delete(s.info)
}
