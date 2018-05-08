
#ifndef LIBUSB_THREADS_POSIX_H
#define LIBUSB_THREADS_POSIX_H

#include <pthread.h>
#ifdef HAVE_SYS_TIME_H
#include <sys/time.h>
#endif

#define usbi_mutex_static_t		pthread_mutex_t
#define USBI_MUTEX_INITIALIZER		PTHREAD_MUTEX_INITIALIZER
#define usbi_mutex_static_lock		pthread_mutex_lock
#define usbi_mutex_static_unlock	pthread_mutex_unlock

#define usbi_mutex_t			pthread_mutex_t
#define usbi_mutex_init(mutex)		pthread_mutex_init((mutex), NULL)
#define usbi_mutex_lock			pthread_mutex_lock
#define usbi_mutex_unlock		pthread_mutex_unlock
#define usbi_mutex_trylock		pthread_mutex_trylock
#define usbi_mutex_destroy		pthread_mutex_destroy

#define usbi_cond_t			pthread_cond_t
#define usbi_cond_init(cond)		pthread_cond_init((cond), NULL)
#define usbi_cond_wait			pthread_cond_wait
#define usbi_cond_broadcast		pthread_cond_broadcast
#define usbi_cond_destroy		pthread_cond_destroy

#define usbi_tls_key_t			pthread_key_t
#define usbi_tls_key_create(key)	pthread_key_create((key), NULL)
#define usbi_tls_key_get		pthread_getspecific
#define usbi_tls_key_set		pthread_setspecific
#define usbi_tls_key_delete		pthread_key_delete

int usbi_get_tid(void);

#endif 
