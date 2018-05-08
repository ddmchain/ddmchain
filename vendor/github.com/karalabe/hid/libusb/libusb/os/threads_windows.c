
#include <config.h>

#include <objbase.h>
#include <errno.h>

#include "libusbi.h"

struct usbi_cond_perthread {
	struct list_head list;
	DWORD tid;
	HANDLE event;
};

int usbi_mutex_static_lock(usbi_mutex_static_t *mutex)
{
	if (!mutex)
		return EINVAL;
	while (InterlockedExchange(mutex, 1) == 1)
		SleepEx(0, TRUE);
	return 0;
}

int usbi_mutex_static_unlock(usbi_mutex_static_t *mutex)
{
	if (!mutex)
		return EINVAL;
	InterlockedExchange(mutex, 0);
	return 0;
}

int usbi_mutex_init(usbi_mutex_t *mutex)
{
	if (!mutex)
		return EINVAL;
	*mutex = CreateMutex(NULL, FALSE, NULL);
	if (!*mutex)
		return ENOMEM;
	return 0;
}

int usbi_mutex_lock(usbi_mutex_t *mutex)
{
	DWORD result;

	if (!mutex)
		return EINVAL;
	result = WaitForSingleObject(*mutex, INFINITE);
	if (result == WAIT_OBJECT_0 || result == WAIT_ABANDONED)
		return 0; 
	else
		return EINVAL; 

}

int usbi_mutex_unlock(usbi_mutex_t *mutex)
{
	if (!mutex)
		return EINVAL;
	if (ReleaseMutex(*mutex))
		return 0;
	else
		return EPERM;
}

int usbi_mutex_trylock(usbi_mutex_t *mutex)
{
	DWORD result;

	if (!mutex)
		return EINVAL;
	result = WaitForSingleObject(*mutex, 0);
	if (result == WAIT_OBJECT_0 || result == WAIT_ABANDONED)
		return 0; 
	else if (result == WAIT_TIMEOUT)
		return EBUSY;
	else
		return EINVAL; 

}

int usbi_mutex_destroy(usbi_mutex_t *mutex)
{

	if (!mutex || !CloseHandle(*mutex))
		return EINVAL;
	*mutex = NULL;
	return 0;
}

int usbi_cond_init(usbi_cond_t *cond)
{
	if (!cond)
		return EINVAL;
	list_init(&cond->waiters);
	list_init(&cond->not_waiting);
	return 0;
}

int usbi_cond_destroy(usbi_cond_t *cond)
{

	struct usbi_cond_perthread *pos, *next_pos;

	if(!cond)
		return EINVAL;
	if (!list_empty(&cond->waiters))
		return EBUSY; 
	list_for_each_entry_safe(pos, next_pos, &cond->not_waiting, list, struct usbi_cond_perthread) {
		CloseHandle(pos->event);
		list_del(&pos->list);
		free(pos);
	}
	return 0;
}

int usbi_cond_broadcast(usbi_cond_t *cond)
{

	int fail = 0;
	struct usbi_cond_perthread *pos;

	if (!cond)
		return EINVAL;
	list_for_each_entry(pos, &cond->waiters, list, struct usbi_cond_perthread) {
		if (!SetEvent(pos->event))
			fail = 1;
	}

	return fail ? EINVAL : 0;
}

__inline static int usbi_cond_intwait(usbi_cond_t *cond,
	usbi_mutex_t *mutex, DWORD timeout_ms)
{
	struct usbi_cond_perthread *pos;
	int r, found = 0;
	DWORD r2, tid = GetCurrentThreadId();

	if (!cond || !mutex)
		return EINVAL;
	list_for_each_entry(pos, &cond->not_waiting, list, struct usbi_cond_perthread) {
		if(tid == pos->tid) {
			found = 1;
			break;
		}
	}

	if (!found) {
		pos = calloc(1, sizeof(struct usbi_cond_perthread));
		if (!pos)
			return ENOMEM; 
		pos->tid = tid;
		pos->event = CreateEvent(NULL, FALSE, FALSE, NULL); 
		if (!pos->event) {
			free(pos);
			return ENOMEM;
		}
		list_add(&pos->list, &cond->not_waiting);
	}

	list_del(&pos->list); 
	list_add(&pos->list, &cond->waiters);

	r  = usbi_mutex_unlock(mutex);
	if (r)
		return r;

	r2 = WaitForSingleObject(pos->event, timeout_ms);
	r = usbi_mutex_lock(mutex);
	if (r)
		return r;

	list_del(&pos->list);
	list_add(&pos->list, &cond->not_waiting);

	if (r2 == WAIT_OBJECT_0)
		return 0;
	else if (r2 == WAIT_TIMEOUT)
		return ETIMEDOUT;
	else
		return EINVAL;
}

int usbi_cond_wait(usbi_cond_t *cond, usbi_mutex_t *mutex)
{
	return usbi_cond_intwait(cond, mutex, INFINITE);
}

int usbi_cond_timedwait(usbi_cond_t *cond,
	usbi_mutex_t *mutex, const struct timeval *tv)
{
	DWORD millis;

	millis = (DWORD)(tv->tv_sec * 1000) + (tv->tv_usec / 1000);

	if (tv->tv_usec % 1000)
		millis++;
	return usbi_cond_intwait(cond, mutex, millis);
}

int usbi_tls_key_create(usbi_tls_key_t *key)
{
	if (!key)
		return EINVAL;
	*key = TlsAlloc();
	if (*key == TLS_OUT_OF_INDEXES)
		return ENOMEM;
	else
		return 0;
}

void *usbi_tls_key_get(usbi_tls_key_t key)
{
	return TlsGetValue(key);
}

int usbi_tls_key_set(usbi_tls_key_t key, void *value)
{
	if (TlsSetValue(key, value))
		return 0;
	else
		return EINVAL;
}

int usbi_tls_key_delete(usbi_tls_key_t key)
{
	if (TlsFree(key))
		return 0;
	else
		return EINVAL;
}

int usbi_get_tid(void)
{
	return (int)GetCurrentThreadId();
}
