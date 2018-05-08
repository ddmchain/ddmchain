
#include <config.h>

#include <time.h>
#if defined(__linux__) || defined(__OpenBSD__)
# if defined(__OpenBSD__)
#  define _BSD_SOURCE
# endif
# include <unistd.h>
# include <sys/syscall.h>
#elif defined(__APPLE__)
# include <mach/mach.h>
#elif defined(__CYGWIN__)
# include <windows.h>
#endif

#include "threads_posix.h"
#include "libusbi.h"

int usbi_cond_timedwait(pthread_cond_t *cond,
	pthread_mutex_t *mutex, const struct timeval *tv)
{
	struct timespec timeout;
	int r;

	r = usbi_backend->clock_gettime(USBI_CLOCK_REALTIME, &timeout);
	if (r < 0)
		return r;

	timeout.tv_sec += tv->tv_sec;
	timeout.tv_nsec += tv->tv_usec * 1000;
	while (timeout.tv_nsec >= 1000000000L) {
		timeout.tv_nsec -= 1000000000L;
		timeout.tv_sec++;
	}

	return pthread_cond_timedwait(cond, mutex, &timeout);
}

int usbi_get_tid(void)
{
	int ret = -1;
#if defined(__ANDROID__)
	ret = gettid();
#elif defined(__linux__)
	ret = syscall(SYS_gettid);
#elif defined(__OpenBSD__)

	ret = syscall(SYS_getthrid);
#elif defined(__APPLE__)
	ret = mach_thread_self();
	mach_port_deallocate(mach_task_self(), ret);
#elif defined(__CYGWIN__)
	ret = GetCurrentThreadId();
#endif

	return ret;
}
