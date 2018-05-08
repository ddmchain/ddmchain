
#include <config.h>

#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#ifdef HAVE_SYS_TYPES_H
#include <sys/types.h>
#endif
#include <assert.h>

#include "libusbi.h"
#include "hotplug.h"

static int usbi_hotplug_match_cb (struct libusb_context *ctx,
	struct libusb_device *dev, libusb_hotplug_event event,
	struct libusb_hotplug_callback *hotplug_cb)
{

	if (hotplug_cb->needs_free) {

		return 1;
	}

	if (!(hotplug_cb->events & event)) {
		return 0;
	}

	if (LIBUSB_HOTPLUG_MATCH_ANY != hotplug_cb->vendor_id &&
	    hotplug_cb->vendor_id != dev->device_descriptor.idVendor) {
		return 0;
	}

	if (LIBUSB_HOTPLUG_MATCH_ANY != hotplug_cb->product_id &&
	    hotplug_cb->product_id != dev->device_descriptor.idProduct) {
		return 0;
	}

	if (LIBUSB_HOTPLUG_MATCH_ANY != hotplug_cb->dev_class &&
	    hotplug_cb->dev_class != dev->device_descriptor.bDeviceClass) {
		return 0;
	}

	return hotplug_cb->cb (ctx, dev, event, hotplug_cb->user_data);
}

void usbi_hotplug_match(struct libusb_context *ctx, struct libusb_device *dev,
	libusb_hotplug_event event)
{
	struct libusb_hotplug_callback *hotplug_cb, *next;
	int ret;

	usbi_mutex_lock(&ctx->hotplug_cbs_lock);

	list_for_each_entry_safe(hotplug_cb, next, &ctx->hotplug_cbs, list, struct libusb_hotplug_callback) {
		usbi_mutex_unlock(&ctx->hotplug_cbs_lock);
		ret = usbi_hotplug_match_cb (ctx, dev, event, hotplug_cb);
		usbi_mutex_lock(&ctx->hotplug_cbs_lock);

		if (ret) {
			list_del(&hotplug_cb->list);
			free(hotplug_cb);
		}
	}

	usbi_mutex_unlock(&ctx->hotplug_cbs_lock);

}

void usbi_hotplug_notification(struct libusb_context *ctx, struct libusb_device *dev,
	libusb_hotplug_event event)
{
	int pending_events;
	libusb_hotplug_message *message = calloc(1, sizeof(*message));

	if (!message) {
		usbi_err(ctx, "error allocating hotplug message");
		return;
	}

	message->event = event;
	message->device = dev;

	usbi_mutex_lock(&ctx->event_data_lock);
	pending_events = usbi_pending_events(ctx);
	list_add_tail(&message->list, &ctx->hotplug_msgs);
	if (!pending_events)
		usbi_signal_event(ctx);
	usbi_mutex_unlock(&ctx->event_data_lock);
}

int API_EXPORTED libusb_hotplug_register_callback(libusb_context *ctx,
	libusb_hotplug_event events, libusb_hotplug_flag flags,
	int vendor_id, int product_id, int dev_class,
	libusb_hotplug_callback_fn cb_fn, void *user_data,
	libusb_hotplug_callback_handle *callback_handle)
{
	libusb_hotplug_callback *new_callback;
	static int handle_id = 1;

	if (!libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {
		return LIBUSB_ERROR_NOT_SUPPORTED;
	}

	if ((LIBUSB_HOTPLUG_MATCH_ANY != vendor_id && (~0xffff & vendor_id)) ||
	    (LIBUSB_HOTPLUG_MATCH_ANY != product_id && (~0xffff & product_id)) ||
	    (LIBUSB_HOTPLUG_MATCH_ANY != dev_class && (~0xff & dev_class)) ||
	    !cb_fn) {
		return LIBUSB_ERROR_INVALID_PARAM;
	}

	USBI_GET_CONTEXT(ctx);

	new_callback = (libusb_hotplug_callback *)calloc(1, sizeof (*new_callback));
	if (!new_callback) {
		return LIBUSB_ERROR_NO_MEM;
	}

	new_callback->ctx = ctx;
	new_callback->vendor_id = vendor_id;
	new_callback->product_id = product_id;
	new_callback->dev_class = dev_class;
	new_callback->flags = flags;
	new_callback->events = events;
	new_callback->cb = cb_fn;
	new_callback->user_data = user_data;
	new_callback->needs_free = 0;

	usbi_mutex_lock(&ctx->hotplug_cbs_lock);

	new_callback->handle = handle_id++;

	list_add(&new_callback->list, &ctx->hotplug_cbs);

	usbi_mutex_unlock(&ctx->hotplug_cbs_lock);

	if (flags & LIBUSB_HOTPLUG_ENUMERATE) {
		int i, len;
		struct libusb_device **devs;

		len = (int) libusb_get_device_list(ctx, &devs);
		if (len < 0) {
			libusb_hotplug_deregister_callback(ctx,
							new_callback->handle);
			return len;
		}

		for (i = 0; i < len; i++) {
			usbi_hotplug_match_cb(ctx, devs[i],
					LIBUSB_HOTPLUG_EVENT_DEVICE_ARRIVED,
					new_callback);
		}

		libusb_free_device_list(devs, 1);
	}

	if (callback_handle)
		*callback_handle = new_callback->handle;

	return LIBUSB_SUCCESS;
}

void API_EXPORTED libusb_hotplug_deregister_callback (struct libusb_context *ctx,
	libusb_hotplug_callback_handle callback_handle)
{
	struct libusb_hotplug_callback *hotplug_cb;

	if (!libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {
		return;
	}

	USBI_GET_CONTEXT(ctx);

	usbi_mutex_lock(&ctx->hotplug_cbs_lock);
	list_for_each_entry(hotplug_cb, &ctx->hotplug_cbs, list,
			    struct libusb_hotplug_callback) {
		if (callback_handle == hotplug_cb->handle) {

			hotplug_cb->needs_free = 1;
		}
	}
	usbi_mutex_unlock(&ctx->hotplug_cbs_lock);

	usbi_hotplug_notification(ctx, NULL, 0);
}

void usbi_hotplug_deregister_all(struct libusb_context *ctx) {
	struct libusb_hotplug_callback *hotplug_cb, *next;

	usbi_mutex_lock(&ctx->hotplug_cbs_lock);
	list_for_each_entry_safe(hotplug_cb, next, &ctx->hotplug_cbs, list,
				 struct libusb_hotplug_callback) {
		list_del(&hotplug_cb->list);
		free(hotplug_cb);
	}

	usbi_mutex_unlock(&ctx->hotplug_cbs_lock);
}
