
#include "config.h"

#include <errno.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#ifdef HAVE_SYS_TYPES_H
#include <sys/types.h>
#endif
#ifdef HAVE_SYS_TIME_H
#include <sys/time.h>
#endif
#ifdef HAVE_SYSLOG_H
#include <syslog.h>
#endif

#ifdef __ANDROID__
#include <android/log.h>
#endif

#include "libusbi.h"
#include "hotplug.h"

#if defined(OS_LINUX)
const struct usbi_os_backend * const usbi_backend = &linux_usbfs_backend;
#elif defined(OS_DARWIN)
const struct usbi_os_backend * const usbi_backend = &darwin_backend;
#elif defined(OS_OPENBSD)
const struct usbi_os_backend * const usbi_backend = &openbsd_backend;
#elif defined(OS_NETBSD)
const struct usbi_os_backend * const usbi_backend = &netbsd_backend;
#elif defined(OS_WINDOWS)

#if defined(USE_USBDK)
const struct usbi_os_backend * const usbi_backend = &usbdk_backend;
#else
const struct usbi_os_backend * const usbi_backend = &windows_backend;
#endif

#elif defined(OS_WINCE)
const struct usbi_os_backend * const usbi_backend = &wince_backend;
#elif defined(OS_HAIKU)
const struct usbi_os_backend * const usbi_backend = &haiku_usb_raw_backend;
#elif defined (OS_SUNOS)
const struct usbi_os_backend * const usbi_backend = &sunos_backend;
#else
#error "Unsupported OS"
#endif

struct libusb_context *usbi_default_context = NULL;
static const struct libusb_version libusb_version_internal =
	{ LIBUSB_MAJOR, LIBUSB_MINOR, LIBUSB_MICRO, LIBUSB_NANO,
	  LIBUSB_RC, "http://libusb.info" };
static int default_context_refcnt = 0;
static usbi_mutex_static_t default_context_lock = USBI_MUTEX_INITIALIZER;
static struct timespec timestamp_origin = { 0, 0 };

usbi_mutex_static_t active_contexts_lock = USBI_MUTEX_INITIALIZER;
struct list_head active_contexts_list;

#define DISCOVERED_DEVICES_SIZE_STEP 8

static struct discovered_devs *discovered_devs_alloc(void)
{
	struct discovered_devs *ret =
		malloc(sizeof(*ret) + (sizeof(void *) * DISCOVERED_DEVICES_SIZE_STEP));

	if (ret) {
		ret->len = 0;
		ret->capacity = DISCOVERED_DEVICES_SIZE_STEP;
	}
	return ret;
}

static void discovered_devs_free(struct discovered_devs *discdevs)
{
	size_t i;

	for (i = 0; i < discdevs->len; i++)
		libusb_unref_device(discdevs->devices[i]);

	free(discdevs);
}

struct discovered_devs *discovered_devs_append(
	struct discovered_devs *discdevs, struct libusb_device *dev)
{
	size_t len = discdevs->len;
	size_t capacity;
	struct discovered_devs *new_discdevs;

	if (len < discdevs->capacity) {
		discdevs->devices[len] = libusb_ref_device(dev);
		discdevs->len++;
		return discdevs;
	}

	usbi_dbg("need to increase capacity");
	capacity = discdevs->capacity + DISCOVERED_DEVICES_SIZE_STEP;

	new_discdevs = realloc(discdevs,
		sizeof(*discdevs) + (sizeof(void *) * capacity));
	if (!new_discdevs) {
		discovered_devs_free(discdevs);
		return NULL;
	}

	discdevs = new_discdevs;
	discdevs->capacity = capacity;
	discdevs->devices[len] = libusb_ref_device(dev);
	discdevs->len++;

	return discdevs;
}

struct libusb_device *usbi_alloc_device(struct libusb_context *ctx,
	unsigned long session_id)
{
	size_t priv_size = usbi_backend->device_priv_size;
	struct libusb_device *dev = calloc(1, sizeof(*dev) + priv_size);
	int r;

	if (!dev)
		return NULL;

	r = usbi_mutex_init(&dev->lock);
	if (r) {
		free(dev);
		return NULL;
	}

	dev->ctx = ctx;
	dev->refcnt = 1;
	dev->session_data = session_id;
	dev->speed = LIBUSB_SPEED_UNKNOWN;

	if (!libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {
		usbi_connect_device (dev);
	}

	return dev;
}

void usbi_connect_device(struct libusb_device *dev)
{
	struct libusb_context *ctx = DEVICE_CTX(dev);

	dev->attached = 1;

	usbi_mutex_lock(&dev->ctx->usb_devs_lock);
	list_add(&dev->list, &dev->ctx->usb_devs);
	usbi_mutex_unlock(&dev->ctx->usb_devs_lock);

	if (libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) && dev->ctx->hotplug_msgs.next) {
		usbi_hotplug_notification(ctx, dev, LIBUSB_HOTPLUG_EVENT_DEVICE_ARRIVED);
	}
}

void usbi_disconnect_device(struct libusb_device *dev)
{
	struct libusb_context *ctx = DEVICE_CTX(dev);

	usbi_mutex_lock(&dev->lock);
	dev->attached = 0;
	usbi_mutex_unlock(&dev->lock);

	usbi_mutex_lock(&ctx->usb_devs_lock);
	list_del(&dev->list);
	usbi_mutex_unlock(&ctx->usb_devs_lock);

	if (libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) && dev->ctx->hotplug_msgs.next) {
		usbi_hotplug_notification(ctx, dev, LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT);
	}
}

int usbi_sanitize_device(struct libusb_device *dev)
{
	int r;
	uint8_t num_configurations;

	r = usbi_device_cache_descriptor(dev);
	if (r < 0)
		return r;

	num_configurations = dev->device_descriptor.bNumConfigurations;
	if (num_configurations > USB_MAXCONFIG) {
		usbi_err(DEVICE_CTX(dev), "too many configurations");
		return LIBUSB_ERROR_IO;
	} else if (0 == num_configurations)
		usbi_dbg("zero configurations, maybe an unauthorized device");

	dev->num_configurations = num_configurations;
	return 0;
}

struct libusb_device *usbi_get_device_by_session_id(struct libusb_context *ctx,
	unsigned long session_id)
{
	struct libusb_device *dev;
	struct libusb_device *ret = NULL;

	usbi_mutex_lock(&ctx->usb_devs_lock);
	list_for_each_entry(dev, &ctx->usb_devs, list, struct libusb_device)
		if (dev->session_data == session_id) {
			ret = libusb_ref_device(dev);
			break;
		}
	usbi_mutex_unlock(&ctx->usb_devs_lock);

	return ret;
}

ssize_t API_EXPORTED libusb_get_device_list(libusb_context *ctx,
	libusb_device ***list)
{
	struct discovered_devs *discdevs = discovered_devs_alloc();
	struct libusb_device **ret;
	int r = 0;
	ssize_t i, len;
	USBI_GET_CONTEXT(ctx);
	usbi_dbg("");

	if (!discdevs)
		return LIBUSB_ERROR_NO_MEM;

	if (libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {

		struct libusb_device *dev;

		if (usbi_backend->hotplug_poll)
			usbi_backend->hotplug_poll();

		usbi_mutex_lock(&ctx->usb_devs_lock);
		list_for_each_entry(dev, &ctx->usb_devs, list, struct libusb_device) {
			discdevs = discovered_devs_append(discdevs, dev);

			if (!discdevs) {
				r = LIBUSB_ERROR_NO_MEM;
				break;
			}
		}
		usbi_mutex_unlock(&ctx->usb_devs_lock);
	} else {

		r = usbi_backend->get_device_list(ctx, &discdevs);
	}

	if (r < 0) {
		len = r;
		goto out;
	}

	len = discdevs->len;
	ret = calloc(len + 1, sizeof(struct libusb_device *));
	if (!ret) {
		len = LIBUSB_ERROR_NO_MEM;
		goto out;
	}

	ret[len] = NULL;
	for (i = 0; i < len; i++) {
		struct libusb_device *dev = discdevs->devices[i];
		ret[i] = libusb_ref_device(dev);
	}
	*list = ret;

out:
	if (discdevs)
		discovered_devs_free(discdevs);
	return len;
}

void API_EXPORTED libusb_free_device_list(libusb_device **list,
	int unref_devices)
{
	if (!list)
		return;

	if (unref_devices) {
		int i = 0;
		struct libusb_device *dev;

		while ((dev = list[i++]) != NULL)
			libusb_unref_device(dev);
	}
	free(list);
}

uint8_t API_EXPORTED libusb_get_bus_number(libusb_device *dev)
{
	return dev->bus_number;
}

uint8_t API_EXPORTED libusb_get_port_number(libusb_device *dev)
{
	return dev->port_number;
}

int API_EXPORTED libusb_get_port_numbers(libusb_device *dev,
	uint8_t* port_numbers, int port_numbers_len)
{
	int i = port_numbers_len;
	struct libusb_context *ctx = DEVICE_CTX(dev);

	if (port_numbers_len <= 0)
		return LIBUSB_ERROR_INVALID_PARAM;

	while((dev) && (dev->port_number != 0)) {
		if (--i < 0) {
			usbi_warn(ctx, "port numbers array is too small");
			return LIBUSB_ERROR_OVERFLOW;
		}
		port_numbers[i] = dev->port_number;
		dev = dev->parent_dev;
	}
	if (i < port_numbers_len)
		memmove(port_numbers, &port_numbers[i], port_numbers_len - i);
	return port_numbers_len - i;
}

int API_EXPORTED libusb_get_port_path(libusb_context *ctx, libusb_device *dev,
	uint8_t* port_numbers, uint8_t port_numbers_len)
{
	UNUSED(ctx);

	return libusb_get_port_numbers(dev, port_numbers, port_numbers_len);
}

DEFAULT_VISIBILITY
libusb_device * LIBUSB_CALL libusb_get_parent(libusb_device *dev)
{
	return dev->parent_dev;
}

uint8_t API_EXPORTED libusb_get_device_address(libusb_device *dev)
{
	return dev->device_address;
}

int API_EXPORTED libusb_get_device_speed(libusb_device *dev)
{
	return dev->speed;
}

static const struct libusb_endpoint_descriptor *find_endpoint(
	struct libusb_config_descriptor *config, unsigned char endpoint)
{
	int iface_idx;
	for (iface_idx = 0; iface_idx < config->bNumInterfaces; iface_idx++) {
		const struct libusb_interface *iface = &config->interface[iface_idx];
		int altsetting_idx;

		for (altsetting_idx = 0; altsetting_idx < iface->num_altsetting;
				altsetting_idx++) {
			const struct libusb_interface_descriptor *altsetting
				= &iface->altsetting[altsetting_idx];
			int ep_idx;

			for (ep_idx = 0; ep_idx < altsetting->bNumEndpoints; ep_idx++) {
				const struct libusb_endpoint_descriptor *ep =
					&altsetting->endpoint[ep_idx];
				if (ep->bEndpointAddress == endpoint)
					return ep;
			}
		}
	}
	return NULL;
}

int API_EXPORTED libusb_get_max_packet_size(libusb_device *dev,
	unsigned char endpoint)
{
	struct libusb_config_descriptor *config;
	const struct libusb_endpoint_descriptor *ep;
	int r;

	r = libusb_get_active_config_descriptor(dev, &config);
	if (r < 0) {
		usbi_err(DEVICE_CTX(dev),
			"could not retrieve active config descriptor");
		return LIBUSB_ERROR_OTHER;
	}

	ep = find_endpoint(config, endpoint);
	if (!ep) {
		r = LIBUSB_ERROR_NOT_FOUND;
		goto out;
	}

	r = ep->wMaxPacketSize;

out:
	libusb_free_config_descriptor(config);
	return r;
}

int API_EXPORTED libusb_get_max_iso_packet_size(libusb_device *dev,
	unsigned char endpoint)
{
	struct libusb_config_descriptor *config;
	const struct libusb_endpoint_descriptor *ep;
	enum libusb_transfer_type ep_type;
	uint16_t val;
	int r;

	r = libusb_get_active_config_descriptor(dev, &config);
	if (r < 0) {
		usbi_err(DEVICE_CTX(dev),
			"could not retrieve active config descriptor");
		return LIBUSB_ERROR_OTHER;
	}

	ep = find_endpoint(config, endpoint);
	if (!ep) {
		r = LIBUSB_ERROR_NOT_FOUND;
		goto out;
	}

	val = ep->wMaxPacketSize;
	ep_type = (enum libusb_transfer_type) (ep->bmAttributes & 0x3);

	r = val & 0x07ff;
	if (ep_type == LIBUSB_TRANSFER_TYPE_ISOCHRONOUS
			|| ep_type == LIBUSB_TRANSFER_TYPE_INTERRUPT)
		r *= (1 + ((val >> 11) & 3));

out:
	libusb_free_config_descriptor(config);
	return r;
}

DEFAULT_VISIBILITY
libusb_device * LIBUSB_CALL libusb_ref_device(libusb_device *dev)
{
	usbi_mutex_lock(&dev->lock);
	dev->refcnt++;
	usbi_mutex_unlock(&dev->lock);
	return dev;
}

void API_EXPORTED libusb_unref_device(libusb_device *dev)
{
	int refcnt;

	if (!dev)
		return;

	usbi_mutex_lock(&dev->lock);
	refcnt = --dev->refcnt;
	usbi_mutex_unlock(&dev->lock);

	if (refcnt == 0) {
		usbi_dbg("destroy device %d.%d", dev->bus_number, dev->device_address);

		libusb_unref_device(dev->parent_dev);

		if (usbi_backend->destroy_device)
			usbi_backend->destroy_device(dev);

		if (!libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {

			usbi_disconnect_device(dev);
		}

		usbi_mutex_destroy(&dev->lock);
		free(dev);
	}
}

int usbi_signal_event(struct libusb_context *ctx)
{
	unsigned char dummy = 1;
	ssize_t r;

	r = usbi_write(ctx->event_pipe[1], &dummy, sizeof(dummy));
	if (r != sizeof(dummy)) {
		usbi_warn(ctx, "internal signalling write failed");
		return LIBUSB_ERROR_IO;
	}

	return 0;
}

int usbi_clear_event(struct libusb_context *ctx)
{
	unsigned char dummy;
	ssize_t r;

	r = usbi_read(ctx->event_pipe[0], &dummy, sizeof(dummy));
	if (r != sizeof(dummy)) {
		usbi_warn(ctx, "internal signalling read failed");
		return LIBUSB_ERROR_IO;
	}

	return 0;
}

int API_EXPORTED libusb_open(libusb_device *dev,
	libusb_device_handle **dev_handle)
{
	struct libusb_context *ctx = DEVICE_CTX(dev);
	struct libusb_device_handle *_dev_handle;
	size_t priv_size = usbi_backend->device_handle_priv_size;
	int r;
	usbi_dbg("open %d.%d", dev->bus_number, dev->device_address);

	if (!dev->attached) {
		return LIBUSB_ERROR_NO_DEVICE;
	}

	_dev_handle = malloc(sizeof(*_dev_handle) + priv_size);
	if (!_dev_handle)
		return LIBUSB_ERROR_NO_MEM;

	r = usbi_mutex_init(&_dev_handle->lock);
	if (r) {
		free(_dev_handle);
		return LIBUSB_ERROR_OTHER;
	}

	_dev_handle->dev = libusb_ref_device(dev);
	_dev_handle->auto_detach_kernel_driver = 0;
	_dev_handle->claimed_interfaces = 0;
	memset(&_dev_handle->os_priv, 0, priv_size);

	r = usbi_backend->open(_dev_handle);
	if (r < 0) {
		usbi_dbg("open %d.%d returns %d", dev->bus_number, dev->device_address, r);
		libusb_unref_device(dev);
		usbi_mutex_destroy(&_dev_handle->lock);
		free(_dev_handle);
		return r;
	}

	usbi_mutex_lock(&ctx->open_devs_lock);
	list_add(&_dev_handle->list, &ctx->open_devs);
	usbi_mutex_unlock(&ctx->open_devs_lock);
	*dev_handle = _dev_handle;

	return 0;
}

DEFAULT_VISIBILITY
libusb_device_handle * LIBUSB_CALL libusb_open_device_with_vid_pid(
	libusb_context *ctx, uint16_t vendor_id, uint16_t product_id)
{
	struct libusb_device **devs;
	struct libusb_device *found = NULL;
	struct libusb_device *dev;
	struct libusb_device_handle *dev_handle = NULL;
	size_t i = 0;
	int r;

	if (libusb_get_device_list(ctx, &devs) < 0)
		return NULL;

	while ((dev = devs[i++]) != NULL) {
		struct libusb_device_descriptor desc;
		r = libusb_get_device_descriptor(dev, &desc);
		if (r < 0)
			goto out;
		if (desc.idVendor == vendor_id && desc.idProduct == product_id) {
			found = dev;
			break;
		}
	}

	if (found) {
		r = libusb_open(found, &dev_handle);
		if (r < 0)
			dev_handle = NULL;
	}

out:
	libusb_free_device_list(devs, 1);
	return dev_handle;
}

static void do_close(struct libusb_context *ctx,
	struct libusb_device_handle *dev_handle)
{
	struct usbi_transfer *itransfer;
	struct usbi_transfer *tmp;

	usbi_mutex_lock(&ctx->flying_transfers_lock);

	list_for_each_entry_safe(itransfer, tmp, &ctx->flying_transfers, list, struct usbi_transfer) {
		struct libusb_transfer *transfer =
			USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);

		if (transfer->dev_handle != dev_handle)
			continue;

		usbi_mutex_lock(&itransfer->lock);
		if (!(itransfer->state_flags & USBI_TRANSFER_DEVICE_DISAPPEARED)) {
			usbi_err(ctx, "Device handle closed while transfer was still being processed, but the device is still connected as far as we know");

			if (itransfer->state_flags & USBI_TRANSFER_CANCELLING)
				usbi_warn(ctx, "A cancellation for an in-flight transfer hasn't completed but closing the device handle");
			else
				usbi_err(ctx, "A cancellation hasn't even been scheduled on the transfer for which the device is closing");
		}
		usbi_mutex_unlock(&itransfer->lock);

		list_del(&itransfer->list);
		transfer->dev_handle = NULL;

		usbi_dbg("Removed transfer %p from the in-flight list because device handle %p closed",
			 transfer, dev_handle);
	}
	usbi_mutex_unlock(&ctx->flying_transfers_lock);

	usbi_mutex_lock(&ctx->open_devs_lock);
	list_del(&dev_handle->list);
	usbi_mutex_unlock(&ctx->open_devs_lock);

	usbi_backend->close(dev_handle);
	libusb_unref_device(dev_handle->dev);
	usbi_mutex_destroy(&dev_handle->lock);
	free(dev_handle);
}

void API_EXPORTED libusb_close(libusb_device_handle *dev_handle)
{
	struct libusb_context *ctx;
	int handling_events;
	int pending_events;

	if (!dev_handle)
		return;
	usbi_dbg("");

	ctx = HANDLE_CTX(dev_handle);
	handling_events = usbi_handling_events(ctx);

	if (!handling_events) {

		usbi_mutex_lock(&ctx->event_data_lock);
		pending_events = usbi_pending_events(ctx);
		ctx->device_close++;
		if (!pending_events)
			usbi_signal_event(ctx);
		usbi_mutex_unlock(&ctx->event_data_lock);

		libusb_lock_events(ctx);
	}

	do_close(ctx, dev_handle);

	if (!handling_events) {

		usbi_mutex_lock(&ctx->event_data_lock);
		ctx->device_close--;
		pending_events = usbi_pending_events(ctx);
		if (!pending_events)
			usbi_clear_event(ctx);
		usbi_mutex_unlock(&ctx->event_data_lock);

		libusb_unlock_events(ctx);
	}
}

DEFAULT_VISIBILITY
libusb_device * LIBUSB_CALL libusb_get_device(libusb_device_handle *dev_handle)
{
	return dev_handle->dev;
}

int API_EXPORTED libusb_get_configuration(libusb_device_handle *dev_handle,
	int *config)
{
	int r = LIBUSB_ERROR_NOT_SUPPORTED;

	usbi_dbg("");
	if (usbi_backend->get_configuration)
		r = usbi_backend->get_configuration(dev_handle, config);

	if (r == LIBUSB_ERROR_NOT_SUPPORTED) {
		uint8_t tmp = 0;
		usbi_dbg("falling back to control message");
		r = libusb_control_transfer(dev_handle, LIBUSB_ENDPOINT_IN,
			LIBUSB_REQUEST_GET_CONFIGURATION, 0, 0, &tmp, 1, 1000);
		if (r == 0) {
			usbi_err(HANDLE_CTX(dev_handle), "zero bytes returned in ctrl transfer?");
			r = LIBUSB_ERROR_IO;
		} else if (r == 1) {
			r = 0;
			*config = tmp;
		} else {
			usbi_dbg("control failed, error %d", r);
		}
	}

	if (r == 0)
		usbi_dbg("active config %d", *config);

	return r;
}

int API_EXPORTED libusb_set_configuration(libusb_device_handle *dev_handle,
	int configuration)
{
	usbi_dbg("configuration %d", configuration);
	return usbi_backend->set_configuration(dev_handle, configuration);
}

int API_EXPORTED libusb_claim_interface(libusb_device_handle *dev_handle,
	int interface_number)
{
	int r = 0;

	usbi_dbg("interface %d", interface_number);
	if (interface_number >= USB_MAXINTERFACES)
		return LIBUSB_ERROR_INVALID_PARAM;

	if (!dev_handle->dev->attached)
		return LIBUSB_ERROR_NO_DEVICE;

	usbi_mutex_lock(&dev_handle->lock);
	if (dev_handle->claimed_interfaces & (1 << interface_number))
		goto out;

	r = usbi_backend->claim_interface(dev_handle, interface_number);
	if (r == 0)
		dev_handle->claimed_interfaces |= 1 << interface_number;

out:
	usbi_mutex_unlock(&dev_handle->lock);
	return r;
}

int API_EXPORTED libusb_release_interface(libusb_device_handle *dev_handle,
	int interface_number)
{
	int r;

	usbi_dbg("interface %d", interface_number);
	if (interface_number >= USB_MAXINTERFACES)
		return LIBUSB_ERROR_INVALID_PARAM;

	usbi_mutex_lock(&dev_handle->lock);
	if (!(dev_handle->claimed_interfaces & (1 << interface_number))) {
		r = LIBUSB_ERROR_NOT_FOUND;
		goto out;
	}

	r = usbi_backend->release_interface(dev_handle, interface_number);
	if (r == 0)
		dev_handle->claimed_interfaces &= ~(1 << interface_number);

out:
	usbi_mutex_unlock(&dev_handle->lock);
	return r;
}

int API_EXPORTED libusb_set_interface_alt_setting(libusb_device_handle *dev_handle,
	int interface_number, int alternate_setting)
{
	usbi_dbg("interface %d altsetting %d",
		interface_number, alternate_setting);
	if (interface_number >= USB_MAXINTERFACES)
		return LIBUSB_ERROR_INVALID_PARAM;

	usbi_mutex_lock(&dev_handle->lock);
	if (!dev_handle->dev->attached) {
		usbi_mutex_unlock(&dev_handle->lock);
		return LIBUSB_ERROR_NO_DEVICE;
	}

	if (!(dev_handle->claimed_interfaces & (1 << interface_number))) {
		usbi_mutex_unlock(&dev_handle->lock);
		return LIBUSB_ERROR_NOT_FOUND;
	}
	usbi_mutex_unlock(&dev_handle->lock);

	return usbi_backend->set_interface_altsetting(dev_handle, interface_number,
		alternate_setting);
}

int API_EXPORTED libusb_clear_halt(libusb_device_handle *dev_handle,
	unsigned char endpoint)
{
	usbi_dbg("endpoint %x", endpoint);
	if (!dev_handle->dev->attached)
		return LIBUSB_ERROR_NO_DEVICE;

	return usbi_backend->clear_halt(dev_handle, endpoint);
}

int API_EXPORTED libusb_reset_device(libusb_device_handle *dev_handle)
{
	usbi_dbg("");
	if (!dev_handle->dev->attached)
		return LIBUSB_ERROR_NO_DEVICE;

	return usbi_backend->reset_device(dev_handle);
}

int API_EXPORTED libusb_alloc_streams(libusb_device_handle *dev_handle,
	uint32_t num_streams, unsigned char *endpoints, int num_endpoints)
{
	usbi_dbg("streams %u eps %d", (unsigned) num_streams, num_endpoints);

	if (!dev_handle->dev->attached)
		return LIBUSB_ERROR_NO_DEVICE;

	if (usbi_backend->alloc_streams)
		return usbi_backend->alloc_streams(dev_handle, num_streams, endpoints,
						   num_endpoints);
	else
		return LIBUSB_ERROR_NOT_SUPPORTED;
}

int API_EXPORTED libusb_free_streams(libusb_device_handle *dev_handle,
	unsigned char *endpoints, int num_endpoints)
{
	usbi_dbg("eps %d", num_endpoints);

	if (!dev_handle->dev->attached)
		return LIBUSB_ERROR_NO_DEVICE;

	if (usbi_backend->free_streams)
		return usbi_backend->free_streams(dev_handle, endpoints,
						  num_endpoints);
	else
		return LIBUSB_ERROR_NOT_SUPPORTED;
}

DEFAULT_VISIBILITY
unsigned char * LIBUSB_CALL libusb_dev_mem_alloc(libusb_device_handle *dev_handle,
        size_t length)
{
	if (!dev_handle->dev->attached)
		return NULL;

	if (usbi_backend->dev_mem_alloc)
		return usbi_backend->dev_mem_alloc(dev_handle, length);
	else
		return NULL;
}

int API_EXPORTED libusb_dev_mem_free(libusb_device_handle *dev_handle,
	unsigned char *buffer, size_t length)
{
	if (usbi_backend->dev_mem_free)
		return usbi_backend->dev_mem_free(dev_handle, buffer, length);
	else
		return LIBUSB_ERROR_NOT_SUPPORTED;
}

int API_EXPORTED libusb_kernel_driver_active(libusb_device_handle *dev_handle,
	int interface_number)
{
	usbi_dbg("interface %d", interface_number);

	if (!dev_handle->dev->attached)
		return LIBUSB_ERROR_NO_DEVICE;

	if (usbi_backend->kernel_driver_active)
		return usbi_backend->kernel_driver_active(dev_handle, interface_number);
	else
		return LIBUSB_ERROR_NOT_SUPPORTED;
}

int API_EXPORTED libusb_detach_kernel_driver(libusb_device_handle *dev_handle,
	int interface_number)
{
	usbi_dbg("interface %d", interface_number);

	if (!dev_handle->dev->attached)
		return LIBUSB_ERROR_NO_DEVICE;

	if (usbi_backend->detach_kernel_driver)
		return usbi_backend->detach_kernel_driver(dev_handle, interface_number);
	else
		return LIBUSB_ERROR_NOT_SUPPORTED;
}

int API_EXPORTED libusb_attach_kernel_driver(libusb_device_handle *dev_handle,
	int interface_number)
{
	usbi_dbg("interface %d", interface_number);

	if (!dev_handle->dev->attached)
		return LIBUSB_ERROR_NO_DEVICE;

	if (usbi_backend->attach_kernel_driver)
		return usbi_backend->attach_kernel_driver(dev_handle, interface_number);
	else
		return LIBUSB_ERROR_NOT_SUPPORTED;
}

int API_EXPORTED libusb_set_auto_detach_kernel_driver(
	libusb_device_handle *dev_handle, int enable)
{
	if (!(usbi_backend->caps & USBI_CAP_SUPPORTS_DETACH_KERNEL_DRIVER))
		return LIBUSB_ERROR_NOT_SUPPORTED;

	dev_handle->auto_detach_kernel_driver = enable;
	return LIBUSB_SUCCESS;
}

void API_EXPORTED libusb_set_debug(libusb_context *ctx, int level)
{
	USBI_GET_CONTEXT(ctx);
	if (!ctx->debug_fixed)
		ctx->debug = level;
}

int API_EXPORTED libusb_init(libusb_context **context)
{
	struct libusb_device *dev, *next;
	char *dbg = getenv("LIBUSB_DEBUG");
	struct libusb_context *ctx;
	static int first_init = 1;
	int r = 0;

	usbi_mutex_static_lock(&default_context_lock);

	if (!timestamp_origin.tv_sec) {
		usbi_backend->clock_gettime(USBI_CLOCK_REALTIME, &timestamp_origin);
	}

	if (!context && usbi_default_context) {
		usbi_dbg("reusing default context");
		default_context_refcnt++;
		usbi_mutex_static_unlock(&default_context_lock);
		return 0;
	}

	ctx = calloc(1, sizeof(*ctx));
	if (!ctx) {
		r = LIBUSB_ERROR_NO_MEM;
		goto err_unlock;
	}

#ifdef ENABLE_DEBUG_LOGGING
	ctx->debug = LIBUSB_LOG_LEVEL_DEBUG;
#endif

	if (dbg) {
		ctx->debug = atoi(dbg);
		if (ctx->debug)
			ctx->debug_fixed = 1;
	}

	if (!usbi_default_context) {
		usbi_default_context = ctx;
		default_context_refcnt++;
		usbi_dbg("created default context");
	}

	usbi_dbg("libusb v%u.%u.%u.%u%s", libusb_version_internal.major, libusb_version_internal.minor,
		libusb_version_internal.micro, libusb_version_internal.nano, libusb_version_internal.rc);

	usbi_mutex_init(&ctx->usb_devs_lock);
	usbi_mutex_init(&ctx->open_devs_lock);
	usbi_mutex_init(&ctx->hotplug_cbs_lock);
	list_init(&ctx->usb_devs);
	list_init(&ctx->open_devs);
	list_init(&ctx->hotplug_cbs);

	usbi_mutex_static_lock(&active_contexts_lock);
	if (first_init) {
		first_init = 0;
		list_init (&active_contexts_list);
	}
	list_add (&ctx->list, &active_contexts_list);
	usbi_mutex_static_unlock(&active_contexts_lock);

	if (usbi_backend->init) {
		r = usbi_backend->init(ctx);
		if (r)
			goto err_free_ctx;
	}

	r = usbi_io_init(ctx);
	if (r < 0)
		goto err_backend_exit;

	usbi_mutex_static_unlock(&default_context_lock);

	if (context)
		*context = ctx;

	return 0;

err_backend_exit:
	if (usbi_backend->exit)
		usbi_backend->exit();
err_free_ctx:
	if (ctx == usbi_default_context) {
		usbi_default_context = NULL;
		default_context_refcnt--;
	}

	usbi_mutex_static_lock(&active_contexts_lock);
	list_del (&ctx->list);
	usbi_mutex_static_unlock(&active_contexts_lock);

	usbi_mutex_lock(&ctx->usb_devs_lock);
	list_for_each_entry_safe(dev, next, &ctx->usb_devs, list, struct libusb_device) {
		list_del(&dev->list);
		libusb_unref_device(dev);
	}
	usbi_mutex_unlock(&ctx->usb_devs_lock);

	usbi_mutex_destroy(&ctx->open_devs_lock);
	usbi_mutex_destroy(&ctx->usb_devs_lock);
	usbi_mutex_destroy(&ctx->hotplug_cbs_lock);

	free(ctx);
err_unlock:
	usbi_mutex_static_unlock(&default_context_lock);
	return r;
}

void API_EXPORTED libusb_exit(struct libusb_context *ctx)
{
	struct libusb_device *dev, *next;
	struct timeval tv = { 0, 0 };

	usbi_dbg("");
	USBI_GET_CONTEXT(ctx);

	usbi_mutex_static_lock(&default_context_lock);
	if (ctx == usbi_default_context) {
		if (--default_context_refcnt > 0) {
			usbi_dbg("not destroying default context");
			usbi_mutex_static_unlock(&default_context_lock);
			return;
		}
		usbi_dbg("destroying default context");
		usbi_default_context = NULL;
	}
	usbi_mutex_static_unlock(&default_context_lock);

	usbi_mutex_static_lock(&active_contexts_lock);
	list_del (&ctx->list);
	usbi_mutex_static_unlock(&active_contexts_lock);

	if (libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {
		usbi_hotplug_deregister_all(ctx);

		if (list_empty(&ctx->open_devs))
			libusb_handle_events_timeout(ctx, &tv);

		usbi_mutex_lock(&ctx->usb_devs_lock);
		list_for_each_entry_safe(dev, next, &ctx->usb_devs, list, struct libusb_device) {
			list_del(&dev->list);
			libusb_unref_device(dev);
		}
		usbi_mutex_unlock(&ctx->usb_devs_lock);
	}

	if (!list_empty(&ctx->usb_devs))
		usbi_warn(ctx, "some libusb_devices were leaked");
	if (!list_empty(&ctx->open_devs))
		usbi_warn(ctx, "application left some devices open");

	usbi_io_exit(ctx);
	if (usbi_backend->exit)
		usbi_backend->exit();

	usbi_mutex_destroy(&ctx->open_devs_lock);
	usbi_mutex_destroy(&ctx->usb_devs_lock);
	usbi_mutex_destroy(&ctx->hotplug_cbs_lock);
	free(ctx);
}

int API_EXPORTED libusb_has_capability(uint32_t capability)
{
	switch (capability) {
	case LIBUSB_CAP_HAS_CAPABILITY:
		return 1;
	case LIBUSB_CAP_HAS_HOTPLUG:
		return !(usbi_backend->get_device_list);
	case LIBUSB_CAP_HAS_HID_ACCESS:
		return (usbi_backend->caps & USBI_CAP_HAS_HID_ACCESS);
	case LIBUSB_CAP_SUPPORTS_DETACH_KERNEL_DRIVER:
		return (usbi_backend->caps & USBI_CAP_SUPPORTS_DETACH_KERNEL_DRIVER);
	}
	return 0;
}

#ifdef LIBUSB_PRINTF_WIN32

int usbi_snprintf(char *str, size_t size, const char *format, ...)
{
	va_list ap;
	int ret;

	va_start(ap, format);
	ret = usbi_vsnprintf(str, size, format, ap);
	va_end(ap);

	return ret;
}

int usbi_vsnprintf(char *str, size_t size, const char *format, va_list ap)
{
	int ret;

	ret = _vsnprintf(str, size, format, ap);
	if (ret < 0 || ret == (int)size) {

		str[size - 1] = '\0';
		if (ret < 0)
			ret = _vsnprintf(NULL, 0, format, ap);
	}

	return ret;
}
#endif

static void usbi_log_str(struct libusb_context *ctx,
	enum libusb_log_level level, const char * str)
{
#if defined(USE_SYSTEM_LOGGING_FACILITY)
#if defined(OS_WINDOWS)
	OutputDebugString(str);
#elif defined(OS_WINCE)

	WCHAR wbuf[USBI_MAX_LOG_LEN];
	MultiByteToWideChar(CP_UTF8, 0, str, -1, wbuf, sizeof(wbuf));
	OutputDebugStringW(wbuf);
#elif defined(__ANDROID__)
	int priority = ANDROID_LOG_UNKNOWN;
	switch (level) {
	case LIBUSB_LOG_LEVEL_INFO: priority = ANDROID_LOG_INFO; break;
	case LIBUSB_LOG_LEVEL_WARNING: priority = ANDROID_LOG_WARN; break;
	case LIBUSB_LOG_LEVEL_ERROR: priority = ANDROID_LOG_ERROR; break;
	case LIBUSB_LOG_LEVEL_DEBUG: priority = ANDROID_LOG_DEBUG; break;
	}
	__android_log_write(priority, "libusb", str);
#elif defined(HAVE_SYSLOG_FUNC)
	int syslog_level = LOG_INFO;
	switch (level) {
	case LIBUSB_LOG_LEVEL_INFO: syslog_level = LOG_INFO; break;
	case LIBUSB_LOG_LEVEL_WARNING: syslog_level = LOG_WARNING; break;
	case LIBUSB_LOG_LEVEL_ERROR: syslog_level = LOG_ERR; break;
	case LIBUSB_LOG_LEVEL_DEBUG: syslog_level = LOG_DEBUG; break;
	}
	syslog(syslog_level, "%s", str);
#else 
#warning System logging is not supported on this platform. Logging to stderr will be used instead.
	fputs(str, stderr);
#endif
#else
	fputs(str, stderr);
#endif 
	UNUSED(ctx);
	UNUSED(level);
}

void usbi_log_v(struct libusb_context *ctx, enum libusb_log_level level,
	const char *function, const char *format, va_list args)
{
	const char *prefix = "";
	char buf[USBI_MAX_LOG_LEN];
	struct timespec now;
	int global_debug, header_len, text_len;
	static int has_debug_header_been_displayed = 0;

#ifdef ENABLE_DEBUG_LOGGING
	global_debug = 1;
	UNUSED(ctx);
#else
	int ctx_level = 0;

	USBI_GET_CONTEXT(ctx);
	if (ctx) {
		ctx_level = ctx->debug;
	} else {
		char *dbg = getenv("LIBUSB_DEBUG");
		if (dbg)
			ctx_level = atoi(dbg);
	}
	global_debug = (ctx_level == LIBUSB_LOG_LEVEL_DEBUG);
	if (!ctx_level)
		return;
	if (level == LIBUSB_LOG_LEVEL_WARNING && ctx_level < LIBUSB_LOG_LEVEL_WARNING)
		return;
	if (level == LIBUSB_LOG_LEVEL_INFO && ctx_level < LIBUSB_LOG_LEVEL_INFO)
		return;
	if (level == LIBUSB_LOG_LEVEL_DEBUG && ctx_level < LIBUSB_LOG_LEVEL_DEBUG)
		return;
#endif

	usbi_backend->clock_gettime(USBI_CLOCK_REALTIME, &now);
	if ((global_debug) && (!has_debug_header_been_displayed)) {
		has_debug_header_been_displayed = 1;
		usbi_log_str(ctx, LIBUSB_LOG_LEVEL_DEBUG, "[timestamp] [threadID] facility level [function call] <message>" USBI_LOG_LINE_END);
		usbi_log_str(ctx, LIBUSB_LOG_LEVEL_DEBUG, "--------------------------------------------------------------------------------" USBI_LOG_LINE_END);
	}
	if (now.tv_nsec < timestamp_origin.tv_nsec) {
		now.tv_sec--;
		now.tv_nsec += 1000000000L;
	}
	now.tv_sec -= timestamp_origin.tv_sec;
	now.tv_nsec -= timestamp_origin.tv_nsec;

	switch (level) {
	case LIBUSB_LOG_LEVEL_INFO:
		prefix = "info";
		break;
	case LIBUSB_LOG_LEVEL_WARNING:
		prefix = "warning";
		break;
	case LIBUSB_LOG_LEVEL_ERROR:
		prefix = "error";
		break;
	case LIBUSB_LOG_LEVEL_DEBUG:
		prefix = "debug";
		break;
	case LIBUSB_LOG_LEVEL_NONE:
		return;
	default:
		prefix = "unknown";
		break;
	}

	if (global_debug) {
		header_len = snprintf(buf, sizeof(buf),
			"[%2d.%06d] [%08x] libusb: %s [%s] ",
			(int)now.tv_sec, (int)(now.tv_nsec / 1000L), usbi_get_tid(), prefix, function);
	} else {
		header_len = snprintf(buf, sizeof(buf),
			"libusb: %s [%s] ", prefix, function);
	}

	if (header_len < 0 || header_len >= (int)sizeof(buf)) {

		header_len = 0;
	}

	buf[header_len] = '\0';
	text_len = vsnprintf(buf + header_len, sizeof(buf) - header_len,
		format, args);
	if (text_len < 0 || text_len + header_len >= (int)sizeof(buf)) {

		text_len = sizeof(buf) - header_len;
	}
	if (header_len + text_len + sizeof(USBI_LOG_LINE_END) >= sizeof(buf)) {

		text_len -= (header_len + text_len + sizeof(USBI_LOG_LINE_END)) - sizeof(buf);
	}
	strcpy(buf + header_len + text_len, USBI_LOG_LINE_END);

	usbi_log_str(ctx, level, buf);
}

void usbi_log(struct libusb_context *ctx, enum libusb_log_level level,
	const char *function, const char *format, ...)
{
	va_list args;

	va_start (args, format);
	usbi_log_v(ctx, level, function, format, args);
	va_end (args);
}

DEFAULT_VISIBILITY const char * LIBUSB_CALL libusb_error_name(int error_code)
{
	switch (error_code) {
	case LIBUSB_ERROR_IO:
		return "LIBUSB_ERROR_IO";
	case LIBUSB_ERROR_INVALID_PARAM:
		return "LIBUSB_ERROR_INVALID_PARAM";
	case LIBUSB_ERROR_ACCESS:
		return "LIBUSB_ERROR_ACCESS";
	case LIBUSB_ERROR_NO_DEVICE:
		return "LIBUSB_ERROR_NO_DEVICE";
	case LIBUSB_ERROR_NOT_FOUND:
		return "LIBUSB_ERROR_NOT_FOUND";
	case LIBUSB_ERROR_BUSY:
		return "LIBUSB_ERROR_BUSY";
	case LIBUSB_ERROR_TIMEOUT:
		return "LIBUSB_ERROR_TIMEOUT";
	case LIBUSB_ERROR_OVERFLOW:
		return "LIBUSB_ERROR_OVERFLOW";
	case LIBUSB_ERROR_PIPE:
		return "LIBUSB_ERROR_PIPE";
	case LIBUSB_ERROR_INTERRUPTED:
		return "LIBUSB_ERROR_INTERRUPTED";
	case LIBUSB_ERROR_NO_MEM:
		return "LIBUSB_ERROR_NO_MEM";
	case LIBUSB_ERROR_NOT_SUPPORTED:
		return "LIBUSB_ERROR_NOT_SUPPORTED";
	case LIBUSB_ERROR_OTHER:
		return "LIBUSB_ERROR_OTHER";

	case LIBUSB_TRANSFER_ERROR:
		return "LIBUSB_TRANSFER_ERROR";
	case LIBUSB_TRANSFER_TIMED_OUT:
		return "LIBUSB_TRANSFER_TIMED_OUT";
	case LIBUSB_TRANSFER_CANCELLED:
		return "LIBUSB_TRANSFER_CANCELLED";
	case LIBUSB_TRANSFER_STALL:
		return "LIBUSB_TRANSFER_STALL";
	case LIBUSB_TRANSFER_NO_DEVICE:
		return "LIBUSB_TRANSFER_NO_DEVICE";
	case LIBUSB_TRANSFER_OVERFLOW:
		return "LIBUSB_TRANSFER_OVERFLOW";

	case 0:
		return "LIBUSB_SUCCESS / LIBUSB_TRANSFER_COMPLETED";
	default:
		return "**UNKNOWN**";
	}
}

DEFAULT_VISIBILITY
const struct libusb_version * LIBUSB_CALL libusb_get_version(void)
{
	return &libusb_version_internal;
}
