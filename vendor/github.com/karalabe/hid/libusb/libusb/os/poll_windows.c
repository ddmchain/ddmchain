
#include <config.h>

#include <errno.h>
#include <stdio.h>
#include <stdlib.h>

#include "libusbi.h"

#if defined(DEBUG_POLL_WINDOWS)
#define poll_dbg usbi_dbg
#else

#if defined(_MSC_VER) && (_MSC_VER < 1400)
#define poll_dbg
#else
#define poll_dbg(...)
#endif
#endif

#if defined(_PREFAST_)
#pragma warning(disable:28719)
#endif

#define CHECK_INIT_POLLING do {if(!is_polling_set) init_polling();} while(0)

const struct winfd INVALID_WINFD = {-1, INVALID_HANDLE_VALUE, NULL, NULL, NULL, RW_NONE};
struct winfd poll_fd[MAX_FDS];

struct {
	CRITICAL_SECTION mutex; 

	HANDLE original_handle;
	DWORD thread_id;
} _poll_fd[MAX_FDS];

BOOLEAN is_polling_set = FALSE;
LONG pipe_number = 0;
static volatile LONG compat_spinlock = 0;

#if !defined(_WIN32_WCE)

static BOOL (__stdcall *pCancelIoEx)(HANDLE, LPOVERLAPPED) = NULL;
#define Use_Duplicate_Handles (pCancelIoEx == NULL)

static inline void setup_cancel_io(void)
{
	HMODULE hKernel32 = GetModuleHandleA("KERNEL32");
	if (hKernel32 != NULL) {
		pCancelIoEx = (BOOL (__stdcall *)(HANDLE,LPOVERLAPPED))
			GetProcAddress(hKernel32, "CancelIoEx");
	}
	usbi_dbg("Will use CancelIo%s for I/O cancellation",
		Use_Duplicate_Handles?"":"Ex");
}

static inline BOOL cancel_io(int _index)
{
	if ((_index < 0) || (_index >= MAX_FDS)) {
		return FALSE;
	}

	if ( (poll_fd[_index].fd < 0) || (poll_fd[_index].handle == INVALID_HANDLE_VALUE)
	  || (poll_fd[_index].handle == 0) || (poll_fd[_index].overlapped == NULL) ) {
		return TRUE;
	}
	if (poll_fd[_index].itransfer && poll_fd[_index].cancel_fn) {

		(*poll_fd[_index].cancel_fn)(poll_fd[_index].itransfer);
		return TRUE;
	}
	if (pCancelIoEx != NULL) {
		return (*pCancelIoEx)(poll_fd[_index].handle, poll_fd[_index].overlapped);
	}
	if (_poll_fd[_index].thread_id == GetCurrentThreadId()) {
		return CancelIo(poll_fd[_index].handle);
	}
	usbi_warn(NULL, "Unable to cancel I/O that was started from another thread");
	return FALSE;
}
#else
#define Use_Duplicate_Handles FALSE

static __inline void setup_cancel_io()
{

}

static __inline BOOL cancel_io(int _index)
{
	if ((_index < 0) || (_index >= MAX_FDS)) {
		return FALSE;
	}
	if ( (poll_fd[_index].fd < 0) || (poll_fd[_index].handle == INVALID_HANDLE_VALUE)
	  || (poll_fd[_index].handle == 0) || (poll_fd[_index].overlapped == NULL) ) {
		return TRUE;
	}
	if (poll_fd[_index].itransfer && poll_fd[_index].cancel_fn) {

		(*poll_fd[_index].cancel_fn)(poll_fd[_index].itransfer);
	}
	return TRUE;
}
#endif

void init_polling(void)
{
	int i;

	while (InterlockedExchange((LONG *)&compat_spinlock, 1) == 1) {
		SleepEx(0, TRUE);
	}
	if (!is_polling_set) {
		setup_cancel_io();
		for (i=0; i<MAX_FDS; i++) {
			poll_fd[i] = INVALID_WINFD;
			_poll_fd[i].original_handle = INVALID_HANDLE_VALUE;
			_poll_fd[i].thread_id = 0;
			InitializeCriticalSection(&_poll_fd[i].mutex);
		}
		is_polling_set = TRUE;
	}
	InterlockedExchange((LONG *)&compat_spinlock, 0);
}

static int _fd_to_index_and_lock(int fd)
{
	int i;

	if (fd < 0)
		return -1;

	for (i=0; i<MAX_FDS; i++) {
		if (poll_fd[i].fd == fd) {
			EnterCriticalSection(&_poll_fd[i].mutex);

			if (poll_fd[i].fd != fd) {
				LeaveCriticalSection(&_poll_fd[i].mutex);
				continue;
			}
			return i;
		}
	}
	return -1;
}

static OVERLAPPED *create_overlapped(void)
{
	OVERLAPPED *overlapped = (OVERLAPPED*) calloc(1, sizeof(OVERLAPPED));
	if (overlapped == NULL) {
		return NULL;
	}
	overlapped->hEvent = CreateEvent(NULL, TRUE, FALSE, NULL);
	if(overlapped->hEvent == NULL) {
		free (overlapped);
		return NULL;
	}
	return overlapped;
}

static void free_overlapped(OVERLAPPED *overlapped)
{
	if (overlapped == NULL)
		return;

	if ( (overlapped->hEvent != 0)
	  && (overlapped->hEvent != INVALID_HANDLE_VALUE) ) {
		CloseHandle(overlapped->hEvent);
	}
	free(overlapped);
}

void exit_polling(void)
{
	int i;

	while (InterlockedExchange((LONG *)&compat_spinlock, 1) == 1) {
		SleepEx(0, TRUE);
	}
	if (is_polling_set) {
		is_polling_set = FALSE;

		for (i=0; i<MAX_FDS; i++) {

			cancel_io(i);

			EnterCriticalSection(&_poll_fd[i].mutex);
			free_overlapped(poll_fd[i].overlapped);
			if (Use_Duplicate_Handles) {

				if (_poll_fd[i].original_handle != INVALID_HANDLE_VALUE) {
					CloseHandle(poll_fd[i].handle);
				}
			}
			poll_fd[i] = INVALID_WINFD;
			LeaveCriticalSection(&_poll_fd[i].mutex);
			DeleteCriticalSection(&_poll_fd[i].mutex);
		}
	}
	InterlockedExchange((LONG *)&compat_spinlock, 0);
}

int usbi_pipe(int filedes[2])
{
	int i;
	OVERLAPPED* overlapped;

	CHECK_INIT_POLLING;

	overlapped = create_overlapped();

	if (overlapped == NULL) {
		return -1;
	}

	overlapped->Internal = STATUS_PENDING;
	overlapped->InternalHigh = 0;

	for (i=0; i<MAX_FDS; i++) {
		if (poll_fd[i].fd < 0) {
			EnterCriticalSection(&_poll_fd[i].mutex);

			if (poll_fd[i].fd >= 0) {
				LeaveCriticalSection(&_poll_fd[i].mutex);
				continue;
			}

			poll_fd[i].fd = i;

			filedes[0] = poll_fd[i].fd;

			filedes[1] = filedes[0];

			poll_fd[i].handle = DUMMY_HANDLE;
			poll_fd[i].overlapped = overlapped;

			poll_fd[i].rw = RW_READ;
			_poll_fd[i].original_handle = INVALID_HANDLE_VALUE;
			LeaveCriticalSection(&_poll_fd[i].mutex);
			return 0;
		}
	}
	free_overlapped(overlapped);
	return -1;
}

struct winfd usbi_create_fd(HANDLE handle, int access_mode, struct usbi_transfer *itransfer, cancel_transfer *cancel_fn)
{
	int i;
	struct winfd wfd = INVALID_WINFD;
	OVERLAPPED* overlapped = NULL;

	CHECK_INIT_POLLING;

	if ((handle == 0) || (handle == INVALID_HANDLE_VALUE)) {
		return INVALID_WINFD;
	}

	wfd.itransfer = itransfer;
	wfd.cancel_fn = cancel_fn;

	if ((access_mode != RW_READ) && (access_mode != RW_WRITE)) {
		usbi_warn(NULL, "only one of RW_READ or RW_WRITE are supported. "
			"If you want to poll for R/W simultaneously, create multiple fds from the same handle.");
		return INVALID_WINFD;
	}
	if (access_mode == RW_READ) {
		wfd.rw = RW_READ;
	} else {
		wfd.rw = RW_WRITE;
	}

	overlapped = create_overlapped();
	if(overlapped == NULL) {
		return INVALID_WINFD;
	}

	for (i=0; i<MAX_FDS; i++) {
		if (poll_fd[i].fd < 0) {
			EnterCriticalSection(&_poll_fd[i].mutex);

			if (poll_fd[i].fd >= 0) {
				LeaveCriticalSection(&_poll_fd[i].mutex);
				continue;
			}

			wfd.fd = i;

			if (Use_Duplicate_Handles) {
				_poll_fd[i].thread_id = GetCurrentThreadId();
				if (!DuplicateHandle(GetCurrentProcess(), handle, GetCurrentProcess(),
					&wfd.handle, 0, TRUE, DUPLICATE_SAME_ACCESS)) {
					usbi_dbg("could not duplicate handle for CancelIo - using original one");
					wfd.handle = handle;

					_poll_fd[i].original_handle = INVALID_HANDLE_VALUE;
				} else {
					_poll_fd[i].original_handle = handle;
				}
			} else {
				wfd.handle = handle;
			}
			wfd.overlapped = overlapped;
			memcpy(&poll_fd[i], &wfd, sizeof(struct winfd));
			LeaveCriticalSection(&_poll_fd[i].mutex);
			return wfd;
		}
	}
	free_overlapped(overlapped);
	return INVALID_WINFD;
}

static void _free_index(int _index)
{

	cancel_io(_index);

	if (Use_Duplicate_Handles) {
		if (_poll_fd[_index].original_handle != INVALID_HANDLE_VALUE) {
			CloseHandle(poll_fd[_index].handle);
		}
		_poll_fd[_index].original_handle = INVALID_HANDLE_VALUE;
		_poll_fd[_index].thread_id = 0;
	}
	free_overlapped(poll_fd[_index].overlapped);
	poll_fd[_index] = INVALID_WINFD;
}

void usbi_free_fd(struct winfd *wfd)
{
	int _index;

	CHECK_INIT_POLLING;

	_index = _fd_to_index_and_lock(wfd->fd);
	if (_index < 0) {
		return;
	}
	_free_index(_index);
	*wfd = INVALID_WINFD;
	LeaveCriticalSection(&_poll_fd[_index].mutex);
}

struct winfd fd_to_winfd(int fd)
{
	int i;
	struct winfd wfd;

	CHECK_INIT_POLLING;

	if (fd < 0)
		return INVALID_WINFD;

	for (i=0; i<MAX_FDS; i++) {
		if (poll_fd[i].fd == fd) {
			EnterCriticalSection(&_poll_fd[i].mutex);

			if (poll_fd[i].fd != fd) {
				LeaveCriticalSection(&_poll_fd[i].mutex);
				continue;
			}
			memcpy(&wfd, &poll_fd[i], sizeof(struct winfd));
			LeaveCriticalSection(&_poll_fd[i].mutex);
			return wfd;
		}
	}
	return INVALID_WINFD;
}

struct winfd handle_to_winfd(HANDLE handle)
{
	int i;
	struct winfd wfd;

	CHECK_INIT_POLLING;

	if ((handle == 0) || (handle == INVALID_HANDLE_VALUE))
		return INVALID_WINFD;

	for (i=0; i<MAX_FDS; i++) {
		if (poll_fd[i].handle == handle) {
			EnterCriticalSection(&_poll_fd[i].mutex);

			if (poll_fd[i].handle != handle) {
				LeaveCriticalSection(&_poll_fd[i].mutex);
				continue;
			}
			memcpy(&wfd, &poll_fd[i], sizeof(struct winfd));
			LeaveCriticalSection(&_poll_fd[i].mutex);
			return wfd;
		}
	}
	return INVALID_WINFD;
}

struct winfd overlapped_to_winfd(OVERLAPPED* overlapped)
{
	int i;
	struct winfd wfd;

	CHECK_INIT_POLLING;

	if (overlapped == NULL)
		return INVALID_WINFD;

	for (i=0; i<MAX_FDS; i++) {
		if (poll_fd[i].overlapped == overlapped) {
			EnterCriticalSection(&_poll_fd[i].mutex);

			if (poll_fd[i].overlapped != overlapped) {
				LeaveCriticalSection(&_poll_fd[i].mutex);
				continue;
			}
			memcpy(&wfd, &poll_fd[i], sizeof(struct winfd));
			LeaveCriticalSection(&_poll_fd[i].mutex);
			return wfd;
		}
	}
	return INVALID_WINFD;
}

int usbi_poll(struct pollfd *fds, unsigned int nfds, int timeout)
{
	unsigned i;
	int _index, object_index, triggered;
	HANDLE *handles_to_wait_on;
	int *handle_to_index;
	DWORD nb_handles_to_wait_on = 0;
	DWORD ret;

	CHECK_INIT_POLLING;

	triggered = 0;
	handles_to_wait_on = (HANDLE*) calloc(nfds+1, sizeof(HANDLE));	
	handle_to_index = (int*) calloc(nfds, sizeof(int));
	if ((handles_to_wait_on == NULL) || (handle_to_index == NULL)) {
		errno = ENOMEM;
		triggered = -1;
		goto poll_exit;
	}

	for (i = 0; i < nfds; ++i) {
		fds[i].revents = 0;

		if ((fds[i].events & ~POLLIN) && (!(fds[i].events & POLLOUT))) {
			fds[i].revents |= POLLERR;
			errno = EACCES;
			usbi_warn(NULL, "unsupported set of events");
			triggered = -1;
			goto poll_exit;
		}

		_index = _fd_to_index_and_lock(fds[i].fd);
		poll_dbg("fd[%d]=%d: (overlapped=%p) got events %04X", i, poll_fd[_index].fd, poll_fd[_index].overlapped, fds[i].events);

		if ( (_index < 0) || (poll_fd[_index].handle == INVALID_HANDLE_VALUE)
		  || (poll_fd[_index].handle == 0) || (poll_fd[_index].overlapped == NULL)) {
			fds[i].revents |= POLLNVAL | POLLERR;
			errno = EBADF;
			if (_index >= 0) {
				LeaveCriticalSection(&_poll_fd[_index].mutex);
			}
			usbi_warn(NULL, "invalid fd");
			triggered = -1;
			goto poll_exit;
		}

		if ((fds[i].events & POLLIN) && (poll_fd[_index].rw != RW_READ)) {
			fds[i].revents |= POLLNVAL | POLLERR;
			errno = EBADF;
			usbi_warn(NULL, "attempted POLLIN on fd without READ access");
			LeaveCriticalSection(&_poll_fd[_index].mutex);
			triggered = -1;
			goto poll_exit;
		}

		if ((fds[i].events & POLLOUT) && (poll_fd[_index].rw != RW_WRITE)) {
			fds[i].revents |= POLLNVAL | POLLERR;
			errno = EBADF;
			usbi_warn(NULL, "attempted POLLOUT on fd without WRITE access");
			LeaveCriticalSection(&_poll_fd[_index].mutex);
			triggered = -1;
			goto poll_exit;
		}

		if ( (HasOverlappedIoCompleted(poll_fd[_index].overlapped))
		  || (HasOverlappedIoCompletedSync(poll_fd[_index].overlapped)) ) {
			poll_dbg("  completed");

			fds[i].revents = fds[i].events;
			triggered++;
		} else {
			handles_to_wait_on[nb_handles_to_wait_on] = poll_fd[_index].overlapped->hEvent;
			handle_to_index[nb_handles_to_wait_on] = i;
			nb_handles_to_wait_on++;
		}
		LeaveCriticalSection(&_poll_fd[_index].mutex);
	}

	if ((timeout != 0) && (triggered == 0) && (nb_handles_to_wait_on != 0)) {
		if (timeout < 0) {
			poll_dbg("starting infinite wait for %u handles...", (unsigned int)nb_handles_to_wait_on);
		} else {
			poll_dbg("starting %d ms wait for %u handles...", timeout, (unsigned int)nb_handles_to_wait_on);
		}
		ret = WaitForMultipleObjects(nb_handles_to_wait_on, handles_to_wait_on,
			FALSE, (timeout<0)?INFINITE:(DWORD)timeout);
		object_index = ret-WAIT_OBJECT_0;
		if ((object_index >= 0) && ((DWORD)object_index < nb_handles_to_wait_on)) {
			poll_dbg("  completed after wait");
			i = handle_to_index[object_index];
			_index = _fd_to_index_and_lock(fds[i].fd);
			fds[i].revents = fds[i].events;
			triggered++;
			if (_index >= 0) {
				LeaveCriticalSection(&_poll_fd[_index].mutex);
			}
		} else if (ret == WAIT_TIMEOUT) {
			poll_dbg("  timed out");
			triggered = 0;	
		} else {
			errno = EIO;
			triggered = -1;	
		}
	}

poll_exit:
	if (handles_to_wait_on != NULL) {
		free(handles_to_wait_on);
	}
	if (handle_to_index != NULL) {
		free(handle_to_index);
	}
	return triggered;
}

int usbi_close(int fd)
{
	int _index;
	int r = -1;

	CHECK_INIT_POLLING;

	_index = _fd_to_index_and_lock(fd);

	if (_index < 0) {
		errno = EBADF;
	} else {
		free_overlapped(poll_fd[_index].overlapped);
		poll_fd[_index] = INVALID_WINFD;
		LeaveCriticalSection(&_poll_fd[_index].mutex);
	}
	return r;
}

ssize_t usbi_write(int fd, const void *buf, size_t count)
{
	int _index;
	UNUSED(buf);

	CHECK_INIT_POLLING;

	if (count != sizeof(unsigned char)) {
		usbi_err(NULL, "this function should only used for signaling");
		return -1;
	}

	_index = _fd_to_index_and_lock(fd);

	if ( (_index < 0) || (poll_fd[_index].overlapped == NULL) ) {
		errno = EBADF;
		if (_index >= 0) {
			LeaveCriticalSection(&_poll_fd[_index].mutex);
		}
		return -1;
	}

	poll_dbg("set pipe event (fd = %d, thread = %08X)", _index, (unsigned int)GetCurrentThreadId());
	SetEvent(poll_fd[_index].overlapped->hEvent);
	poll_fd[_index].overlapped->Internal = STATUS_WAIT_0;

	poll_fd[_index].overlapped->InternalHigh++;

	LeaveCriticalSection(&_poll_fd[_index].mutex);
	return sizeof(unsigned char);
}

ssize_t usbi_read(int fd, void *buf, size_t count)
{
	int _index;
	ssize_t r = -1;
	UNUSED(buf);

	CHECK_INIT_POLLING;

	if (count != sizeof(unsigned char)) {
		usbi_err(NULL, "this function should only used for signaling");
		return -1;
	}

	_index = _fd_to_index_and_lock(fd);

	if (_index < 0) {
		errno = EBADF;
		return -1;
	}

	if (WaitForSingleObject(poll_fd[_index].overlapped->hEvent, INFINITE) != WAIT_OBJECT_0) {
		usbi_warn(NULL, "waiting for event failed: %u", (unsigned int)GetLastError());
		errno = EIO;
		goto out;
	}

	poll_dbg("clr pipe event (fd = %d, thread = %08X)", _index, (unsigned int)GetCurrentThreadId());
	poll_fd[_index].overlapped->InternalHigh--;

	if (poll_fd[_index].overlapped->InternalHigh <= 0) {
		ResetEvent(poll_fd[_index].overlapped->hEvent);
		poll_fd[_index].overlapped->Internal = STATUS_PENDING;
	}

	r = sizeof(unsigned char);

out:
	LeaveCriticalSection(&_poll_fd[_index].mutex);
	return r;
}
