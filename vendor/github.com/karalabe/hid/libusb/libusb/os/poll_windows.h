
#pragma once

#if defined(_MSC_VER)

#pragma warning(disable:4127) 
#endif

#if !defined(STATUS_REPARSE)	
#define STATUS_REPARSE ((LONG)0x00000104L)
#endif
#define STATUS_COMPLETED_SYNCHRONOUSLY	STATUS_REPARSE
#if defined(_WIN32_WCE)

#define HasOverlappedIoCompleted(lpOverlapped) (((DWORD)(lpOverlapped)->Internal) != STATUS_PENDING)
#endif
#define HasOverlappedIoCompletedSync(lpOverlapped)	(((DWORD)(lpOverlapped)->Internal) == STATUS_COMPLETED_SYNCHRONOUSLY)

#define DUMMY_HANDLE ((HANDLE)(LONG_PTR)-2)

enum windows_version {
	WINDOWS_CE = -2,
	WINDOWS_UNDEFINED = -1,
	WINDOWS_UNSUPPORTED = 0,
	WINDOWS_XP = 0x51,
	WINDOWS_2003 = 0x52,	
	WINDOWS_VISTA = 0x60,
	WINDOWS_7 = 0x61,
	WINDOWS_8 = 0x62,
	WINDOWS_8_1_OR_LATER = 0x63,
	WINDOWS_MAX
};
extern int windows_version;

#define MAX_FDS     256

#define POLLIN      0x0001    
#define POLLPRI     0x0002    
#define POLLOUT     0x0004    
#define POLLERR     0x0008    
#define POLLHUP     0x0010    
#define POLLNVAL    0x0020    

struct pollfd {
    int fd;           
    short events;     
    short revents;    
};

enum rw_type {
	RW_NONE,
	RW_READ,
	RW_WRITE,
};

typedef int cancel_transfer(struct usbi_transfer *itransfer);

struct winfd {
	int fd;							
	HANDLE handle;					
	OVERLAPPED* overlapped;			
	struct usbi_transfer *itransfer;		
	cancel_transfer *cancel_fn;		
	enum rw_type rw;				
};
extern const struct winfd INVALID_WINFD;

int usbi_pipe(int pipefd[2]);
int usbi_poll(struct pollfd *fds, unsigned int nfds, int timeout);
ssize_t usbi_write(int fd, const void *buf, size_t count);
ssize_t usbi_read(int fd, void *buf, size_t count);
int usbi_close(int fd);

void init_polling(void);
void exit_polling(void);
struct winfd usbi_create_fd(HANDLE handle, int access_mode, 
	struct usbi_transfer *transfer, cancel_transfer *cancel_fn);
void usbi_free_fd(struct winfd* winfd);
struct winfd fd_to_winfd(int fd);
struct winfd handle_to_winfd(HANDLE handle);
struct winfd overlapped_to_winfd(OVERLAPPED* overlapped);

#if defined(DDKBUILD)
#include <winsock.h>	
#endif

#if !defined(TIMESPEC_TO_TIMEVAL)
#define TIMESPEC_TO_TIMEVAL(tv, ts) {                   \
	(tv)->tv_sec = (long)(ts)->tv_sec;                  \
	(tv)->tv_usec = (long)(ts)->tv_nsec / 1000;         \
}
#endif
#if !defined(timersub)
#define timersub(a, b, result)                          \
do {                                                    \
	(result)->tv_sec = (a)->tv_sec - (b)->tv_sec;       \
	(result)->tv_usec = (a)->tv_usec - (b)->tv_usec;    \
	if ((result)->tv_usec < 0) {                        \
		--(result)->tv_sec;                             \
		(result)->tv_usec += 1000000;                   \
	}                                                   \
} while (0)
#endif
