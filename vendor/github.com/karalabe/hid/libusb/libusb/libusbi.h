
#ifndef LIBUSBI_H
#define LIBUSBI_H

#include <config.h>

#include <stdlib.h>

#include <stddef.h>
#include <stdint.h>
#include <time.h>
#include <stdarg.h>
#ifdef HAVE_POLL_H
#include <poll.h>
#endif
#ifdef HAVE_MISSING_H
#include <missing.h>
#endif

#include "libusb.h"
#include "version.h"

#define API_EXPORTED LIBUSB_CALL DEFAULT_VISIBILITY

#ifdef __cplusplus
extern "C" {
#endif

#define DEVICE_DESC_LENGTH	18

#define USB_MAXENDPOINTS	32
#define USB_MAXINTERFACES	32
#define USB_MAXCONFIG		8

#define USBI_CAP_HAS_HID_ACCESS			0x00010000
#define USBI_CAP_SUPPORTS_DETACH_KERNEL_DRIVER	0x00020000

#define USBI_MAX_LOG_LEN	1024

#define USBI_LOG_LINE_END	"\n"

#define UNUSED(var)		do { (void)(var); } while(0)

#if !defined(ARRAYSIZE)
#define ARRAYSIZE(array) (sizeof(array) / sizeof(array[0]))
#endif

struct list_head {
	struct list_head *prev, *next;
};

#define list_entry(ptr, type, member) \
	((type *)((uintptr_t)(ptr) - (uintptr_t)offsetof(type, member)))

#define list_first_entry(ptr, type, member) \
	list_entry((ptr)->next, type, member)

#define list_for_each_entry(pos, head, member, type)			\
	for (pos = list_entry((head)->next, type, member);		\
		 &pos->member != (head);				\
		 pos = list_entry(pos->member.next, type, member))

#define list_for_each_entry_safe(pos, n, head, member, type)		\
	for (pos = list_entry((head)->next, type, member),		\
		 n = list_entry(pos->member.next, type, member);	\
		 &pos->member != (head);				\
		 pos = n, n = list_entry(n->member.next, type, member))

#define list_empty(entry) ((entry)->next == (entry))

static inline void list_init(struct list_head *entry)
{
	entry->prev = entry->next = entry;
}

static inline void list_add(struct list_head *entry, struct list_head *head)
{
	entry->next = head->next;
	entry->prev = head;

	head->next->prev = entry;
	head->next = entry;
}

static inline void list_add_tail(struct list_head *entry,
	struct list_head *head)
{
	entry->next = head;
	entry->prev = head->prev;

	head->prev->next = entry;
	head->prev = entry;
}

static inline void list_del(struct list_head *entry)
{
	entry->next->prev = entry->prev;
	entry->prev->next = entry->next;
	entry->next = entry->prev = NULL;
}

static inline void *usbi_reallocf(void *ptr, size_t size)
{
	void *ret = realloc(ptr, size);
	if (!ret)
		free(ptr);
	return ret;
}

#define container_of(ptr, type, member) ({			\
	const typeof( ((type *)0)->member ) *mptr = (ptr);	\
	(type *)( (char *)mptr - offsetof(type,member) );})

#ifndef MIN
#define MIN(a, b)	((a) < (b) ? (a) : (b))
#endif
#ifndef MAX
#define MAX(a, b)	((a) > (b) ? (a) : (b))
#endif

#define TIMESPEC_IS_SET(ts) ((ts)->tv_sec != 0 || (ts)->tv_nsec != 0)

#if defined(_WIN32) || defined(__CYGWIN__) || defined(_WIN32_WCE)
#define TIMEVAL_TV_SEC_TYPE	long
#else
#define TIMEVAL_TV_SEC_TYPE	time_t
#endif

#ifndef TIMESPEC_TO_TIMEVAL
#define TIMESPEC_TO_TIMEVAL(tv, ts)					\
	do {								\
		(tv)->tv_sec = (TIMEVAL_TV_SEC_TYPE) (ts)->tv_sec;	\
		(tv)->tv_usec = (ts)->tv_nsec / 1000;			\
	} while (0)
#endif

void usbi_log(struct libusb_context *ctx, enum libusb_log_level level,
	const char *function, const char *format, ...);

void usbi_log_v(struct libusb_context *ctx, enum libusb_log_level level,
	const char *function, const char *format, va_list args);

#if !defined(_MSC_VER) || _MSC_VER >= 1400

#ifdef ENABLE_LOGGING
#define _usbi_log(ctx, level, ...) usbi_log(ctx, level, __FUNCTION__, __VA_ARGS__)
#define usbi_dbg(...) _usbi_log(NULL, LIBUSB_LOG_LEVEL_DEBUG, __VA_ARGS__)
#else
#define _usbi_log(ctx, level, ...) do { (void)(ctx); } while(0)
#define usbi_dbg(...) do {} while(0)
#endif

#define usbi_info(ctx, ...) _usbi_log(ctx, LIBUSB_LOG_LEVEL_INFO, __VA_ARGS__)
#define usbi_warn(ctx, ...) _usbi_log(ctx, LIBUSB_LOG_LEVEL_WARNING, __VA_ARGS__)
#define usbi_err(ctx, ...) _usbi_log(ctx, LIBUSB_LOG_LEVEL_ERROR, __VA_ARGS__)

#else 

#ifdef ENABLE_LOGGING
#define LOG_BODY(ctxt, level)				\
{							\
	va_list args;					\
	va_start(args, format);				\
	usbi_log_v(ctxt, level, "", format, args);	\
	va_end(args);					\
}
#else
#define LOG_BODY(ctxt, level)				\
{							\
	(void)(ctxt);					\
}
#endif

static inline void usbi_info(struct libusb_context *ctx, const char *format, ...)
	LOG_BODY(ctx, LIBUSB_LOG_LEVEL_INFO)
static inline void usbi_warn(struct libusb_context *ctx, const char *format, ...)
	LOG_BODY(ctx, LIBUSB_LOG_LEVEL_WARNING)
static inline void usbi_err(struct libusb_context *ctx, const char *format, ...)
	LOG_BODY(ctx, LIBUSB_LOG_LEVEL_ERROR)

static inline void usbi_dbg(const char *format, ...)
	LOG_BODY(NULL, LIBUSB_LOG_LEVEL_DEBUG)

#endif 

#define USBI_GET_CONTEXT(ctx)				\
	do {						\
		if (!(ctx))				\
			(ctx) = usbi_default_context;	\
	} while(0)

#define DEVICE_CTX(dev)		((dev)->ctx)
#define HANDLE_CTX(handle)	(DEVICE_CTX((handle)->dev))
#define TRANSFER_CTX(transfer)	(HANDLE_CTX((transfer)->dev_handle))
#define ITRANSFER_CTX(transfer) \
	(TRANSFER_CTX(USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer)))

#define IS_EPIN(ep)		(0 != ((ep) & LIBUSB_ENDPOINT_IN))
#define IS_EPOUT(ep)		(!IS_EPIN(ep))
#define IS_XFERIN(xfer)		(0 != ((xfer)->endpoint & LIBUSB_ENDPOINT_IN))
#define IS_XFEROUT(xfer)	(!IS_XFERIN(xfer))

#if defined(THREADS_POSIX)
#include "os/threads_posix.h"
#elif defined(OS_WINDOWS) || defined(OS_WINCE)
#include "os/threads_windows.h"
#endif

extern struct libusb_context *usbi_default_context;

struct pollfd;

struct libusb_context {
	int debug;
	int debug_fixed;

	int event_pipe[2];

	struct list_head usb_devs;
	usbi_mutex_t usb_devs_lock;

	struct list_head open_devs;
	usbi_mutex_t open_devs_lock;

	struct list_head hotplug_cbs;
	usbi_mutex_t hotplug_cbs_lock;

	struct list_head flying_transfers;

	usbi_mutex_t flying_transfers_lock;

	libusb_pollfd_added_cb fd_added_cb;
	libusb_pollfd_removed_cb fd_removed_cb;
	void *fd_cb_user_data;

	usbi_mutex_t events_lock;

	int event_handler_active;

	usbi_tls_key_t event_handling_key;

	usbi_mutex_t event_waiters_lock;
	usbi_cond_t event_waiters_cond;

	usbi_mutex_t event_data_lock;

	unsigned int event_flags;

	unsigned int device_close;

	struct list_head ipollfds;
	struct pollfd *pollfds;
	POLL_NFDS_TYPE pollfds_cnt;

	struct list_head hotplug_msgs;

	struct list_head completed_transfers;

#ifdef USBI_TIMERFD_AVAILABLE

	int timerfd;
#endif

	struct list_head list;
};

enum usbi_event_flags {

	USBI_EVENT_POLLFDS_MODIFIED = 1 << 0,

	USBI_EVENT_USER_INTERRUPT = 1 << 1,
};

#define usbi_handling_events(ctx) \
	(usbi_tls_key_get((ctx)->event_handling_key) != NULL)

#define usbi_start_event_handling(ctx) \
	usbi_tls_key_set((ctx)->event_handling_key, ctx)

#define usbi_end_event_handling(ctx) \
	usbi_tls_key_set((ctx)->event_handling_key, NULL)

#define usbi_pending_events(ctx) \
	((ctx)->event_flags || (ctx)->device_close \
	 || !list_empty(&(ctx)->hotplug_msgs) || !list_empty(&(ctx)->completed_transfers))

#ifdef USBI_TIMERFD_AVAILABLE
#define usbi_using_timerfd(ctx) ((ctx)->timerfd >= 0)
#else
#define usbi_using_timerfd(ctx) (0)
#endif

struct libusb_device {

	usbi_mutex_t lock;
	int refcnt;

	struct libusb_context *ctx;

	uint8_t bus_number;
	uint8_t port_number;
	struct libusb_device* parent_dev;
	uint8_t device_address;
	uint8_t num_configurations;
	enum libusb_speed speed;

	struct list_head list;
	unsigned long session_data;

	struct libusb_device_descriptor device_descriptor;
	int attached;

	unsigned char os_priv
#if defined(__STDC_VERSION__) && (__STDC_VERSION__ >= 199901L)
	[] 
#else
	[0] 
#endif
#if defined(OS_SUNOS)
	__attribute__ ((aligned (8)));
#else
	;
#endif
};

struct libusb_device_handle {

	usbi_mutex_t lock;
	unsigned long claimed_interfaces;

	struct list_head list;
	struct libusb_device *dev;
	int auto_detach_kernel_driver;
	unsigned char os_priv
#if defined(__STDC_VERSION__) && (__STDC_VERSION__ >= 199901L)
	[] 
#else
	[0] 
#endif
#if defined(OS_SUNOS)
	__attribute__ ((aligned (8)));
#else
	;
#endif
};

enum {
	USBI_CLOCK_MONOTONIC,
	USBI_CLOCK_REALTIME
};

struct usbi_transfer {
	int num_iso_packets;
	struct list_head list;
	struct list_head completed_list;
	struct timeval timeout;
	int transferred;
	uint32_t stream_id;
	uint8_t state_flags;   
	uint8_t timeout_flags; 

	usbi_mutex_t lock;
};

enum usbi_transfer_state_flags {

	USBI_TRANSFER_IN_FLIGHT = 1 << 0,

	USBI_TRANSFER_CANCELLING = 1 << 1,

	USBI_TRANSFER_DEVICE_DISAPPEARED = 1 << 2,
};

enum usbi_transfer_timeout_flags {

	USBI_TRANSFER_OS_HANDLES_TIMEOUT = 1 << 0,

	USBI_TRANSFER_TIMEOUT_HANDLED = 1 << 1,

	USBI_TRANSFER_TIMED_OUT = 1 << 2,
};

#define USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer)			\
	((struct libusb_transfer *)(((unsigned char *)(transfer))	\
		+ sizeof(struct usbi_transfer)))
#define LIBUSB_TRANSFER_TO_USBI_TRANSFER(transfer)			\
	((struct usbi_transfer *)(((unsigned char *)(transfer))		\
		- sizeof(struct usbi_transfer)))

static inline void *usbi_transfer_get_os_priv(struct usbi_transfer *transfer)
{
	return ((unsigned char *)transfer) + sizeof(struct usbi_transfer)
		+ sizeof(struct libusb_transfer)
		+ (transfer->num_iso_packets
			* sizeof(struct libusb_iso_packet_descriptor));
}

struct usb_descriptor_header {
	uint8_t bLength;
	uint8_t bDescriptorType;
};

int usbi_io_init(struct libusb_context *ctx);
void usbi_io_exit(struct libusb_context *ctx);

struct libusb_device *usbi_alloc_device(struct libusb_context *ctx,
	unsigned long session_id);
struct libusb_device *usbi_get_device_by_session_id(struct libusb_context *ctx,
	unsigned long session_id);
int usbi_sanitize_device(struct libusb_device *dev);
void usbi_handle_disconnect(struct libusb_device_handle *dev_handle);

int usbi_handle_transfer_completion(struct usbi_transfer *itransfer,
	enum libusb_transfer_status status);
int usbi_handle_transfer_cancellation(struct usbi_transfer *transfer);
void usbi_signal_transfer_completion(struct usbi_transfer *transfer);

int usbi_parse_descriptor(const unsigned char *source, const char *descriptor,
	void *dest, int host_endian);
int usbi_device_cache_descriptor(libusb_device *dev);
int usbi_get_config_index_by_value(struct libusb_device *dev,
	uint8_t bConfigurationValue, int *idx);

void usbi_connect_device (struct libusb_device *dev);
void usbi_disconnect_device (struct libusb_device *dev);

int usbi_signal_event(struct libusb_context *ctx);
int usbi_clear_event(struct libusb_context *ctx);

#if defined(OS_LINUX) || defined(OS_DARWIN) || defined(OS_OPENBSD) || defined(OS_NETBSD) ||\
	defined(OS_HAIKU) || defined(OS_SUNOS)
#include <unistd.h>
#include "os/poll_posix.h"
#elif defined(OS_WINDOWS) || defined(OS_WINCE)
#include "os/poll_windows.h"
#endif

#if defined(_MSC_VER) && (_MSC_VER < 1900)
#define snprintf usbi_snprintf
#define vsnprintf usbi_vsnprintf
int usbi_snprintf(char *dst, size_t size, const char *format, ...);
int usbi_vsnprintf(char *dst, size_t size, const char *format, va_list ap);
#define LIBUSB_PRINTF_WIN32
#endif

struct usbi_pollfd {

	struct libusb_pollfd pollfd;

	struct list_head list;
};

int usbi_add_pollfd(struct libusb_context *ctx, int fd, short events);
void usbi_remove_pollfd(struct libusb_context *ctx, int fd);

struct discovered_devs {
	size_t len;
	size_t capacity;
	struct libusb_device *devices
#if defined(__STDC_VERSION__) && (__STDC_VERSION__ >= 199901L)
	[] 
#else
	[0] 
#endif
	;
};

struct discovered_devs *discovered_devs_append(
	struct discovered_devs *discdevs, struct libusb_device *dev);

struct usbi_os_backend {

	const char *name;

	uint32_t caps;

	int (*init)(struct libusb_context *ctx);

	void (*exit)(void);

	int (*get_device_list)(struct libusb_context *ctx,
		struct discovered_devs **discdevs);

	void (*hotplug_poll)(void);

	int (*open)(struct libusb_device_handle *dev_handle);

	void (*close)(struct libusb_device_handle *dev_handle);

	int (*get_device_descriptor)(struct libusb_device *device,
		unsigned char *buffer, int *host_endian);

	int (*get_active_config_descriptor)(struct libusb_device *device,
		unsigned char *buffer, size_t len, int *host_endian);

	int (*get_config_descriptor)(struct libusb_device *device,
		uint8_t config_index, unsigned char *buffer, size_t len,
		int *host_endian);

	int (*get_config_descriptor_by_value)(struct libusb_device *device,
		uint8_t bConfigurationValue, unsigned char **buffer,
		int *host_endian);

	int (*get_configuration)(struct libusb_device_handle *dev_handle, int *config);

	int (*set_configuration)(struct libusb_device_handle *dev_handle, int config);

	int (*claim_interface)(struct libusb_device_handle *dev_handle, int interface_number);

	int (*release_interface)(struct libusb_device_handle *dev_handle, int interface_number);

	int (*set_interface_altsetting)(struct libusb_device_handle *dev_handle,
		int interface_number, int altsetting);

	int (*clear_halt)(struct libusb_device_handle *dev_handle,
		unsigned char endpoint);

	int (*reset_device)(struct libusb_device_handle *dev_handle);

	int (*alloc_streams)(struct libusb_device_handle *dev_handle,
		uint32_t num_streams, unsigned char *endpoints, int num_endpoints);

	int (*free_streams)(struct libusb_device_handle *dev_handle,
		unsigned char *endpoints, int num_endpoints);

	unsigned char *(*dev_mem_alloc)(struct libusb_device_handle *handle,
		size_t len);

	int (*dev_mem_free)(struct libusb_device_handle *handle,
		unsigned char *buffer, size_t len);

	int (*kernel_driver_active)(struct libusb_device_handle *dev_handle,
		int interface_number);

	int (*detach_kernel_driver)(struct libusb_device_handle *dev_handle,
		int interface_number);

	int (*attach_kernel_driver)(struct libusb_device_handle *dev_handle,
		int interface_number);

	void (*destroy_device)(struct libusb_device *dev);

	int (*submit_transfer)(struct usbi_transfer *itransfer);

	int (*cancel_transfer)(struct usbi_transfer *itransfer);

	void (*clear_transfer_priv)(struct usbi_transfer *itransfer);

	int (*handle_events)(struct libusb_context *ctx,
		struct pollfd *fds, POLL_NFDS_TYPE nfds, int num_ready);

	int (*handle_transfer_completion)(struct usbi_transfer *itransfer);

	int (*clock_gettime)(int clkid, struct timespec *tp);

#ifdef USBI_TIMERFD_AVAILABLE

	clockid_t (*get_timerfd_clockid)(void);
#endif

	size_t device_priv_size;

	size_t device_handle_priv_size;

	size_t transfer_priv_size;
};

extern const struct usbi_os_backend * const usbi_backend;

extern const struct usbi_os_backend linux_usbfs_backend;
extern const struct usbi_os_backend darwin_backend;
extern const struct usbi_os_backend openbsd_backend;
extern const struct usbi_os_backend netbsd_backend;
extern const struct usbi_os_backend windows_backend;
extern const struct usbi_os_backend usbdk_backend;
extern const struct usbi_os_backend wince_backend;
extern const struct usbi_os_backend haiku_usb_raw_backend;
extern const struct usbi_os_backend sunos_backend;

extern struct list_head active_contexts_list;
extern usbi_mutex_static_t active_contexts_lock;

#ifdef __cplusplus
}
#endif

#endif
