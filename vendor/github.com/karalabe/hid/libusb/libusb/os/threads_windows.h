
#ifndef LIBUSB_THREADS_WINDOWS_H
#define LIBUSB_THREADS_WINDOWS_H

#define usbi_mutex_static_t	volatile LONG
#define USBI_MUTEX_INITIALIZER	0

#define usbi_mutex_t		HANDLE

typedef struct usbi_cond {

	struct list_head waiters;
	struct list_head not_waiting;
} usbi_cond_t;

#if (!defined(HAVE_STRUCT_TIMESPEC) && !defined(_TIMESPEC_DEFINED))
#define HAVE_STRUCT_TIMESPEC 1
#define _TIMESPEC_DEFINED 1
struct timespec {
	long tv_sec;
	long tv_nsec;
};
#endif 

#ifndef ETIMEDOUT
#  define ETIMEDOUT 10060     
#endif

#define usbi_tls_key_t		DWORD

int usbi_mutex_static_lock(usbi_mutex_static_t *mutex);
int usbi_mutex_static_unlock(usbi_mutex_static_t *mutex);

int usbi_mutex_init(usbi_mutex_t *mutex);
int usbi_mutex_lock(usbi_mutex_t *mutex);
int usbi_mutex_unlock(usbi_mutex_t *mutex);
int usbi_mutex_trylock(usbi_mutex_t *mutex);
int usbi_mutex_destroy(usbi_mutex_t *mutex);

int usbi_cond_init(usbi_cond_t *cond);
int usbi_cond_wait(usbi_cond_t *cond, usbi_mutex_t *mutex);
int usbi_cond_timedwait(usbi_cond_t *cond,
	usbi_mutex_t *mutex, const struct timeval *tv);
int usbi_cond_broadcast(usbi_cond_t *cond);
int usbi_cond_destroy(usbi_cond_t *cond);

int usbi_tls_key_create(usbi_tls_key_t *key);
void *usbi_tls_key_get(usbi_tls_key_t key);
int usbi_tls_key_set(usbi_tls_key_t key, void *value);
int usbi_tls_key_delete(usbi_tls_key_t key);

int usbi_get_tid(void);

#endif 
