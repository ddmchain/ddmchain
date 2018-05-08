
#if !defined(USBI_HOTPLUG_H)
#define USBI_HOTPLUG_H

#ifndef LIBUSBI_H
#include "libusbi.h"
#endif

struct libusb_hotplug_callback {

	struct libusb_context *ctx;

	int vendor_id;

	int product_id;

	int dev_class;

	libusb_hotplug_flag flags;

	libusb_hotplug_event events;

	libusb_hotplug_callback_fn cb;

	libusb_hotplug_callback_handle handle;

	void *user_data;

	int needs_free;

	struct list_head list;
};

typedef struct libusb_hotplug_callback libusb_hotplug_callback;

struct libusb_hotplug_message {

	libusb_hotplug_event event;

	struct libusb_device *device;

	struct list_head list;
};

typedef struct libusb_hotplug_message libusb_hotplug_message;

void usbi_hotplug_deregister_all(struct libusb_context *ctx);
void usbi_hotplug_match(struct libusb_context *ctx, struct libusb_device *dev,
			libusb_hotplug_event event);
void usbi_hotplug_notification(struct libusb_context *ctx, struct libusb_device *dev,
			libusb_hotplug_event event);

#endif
