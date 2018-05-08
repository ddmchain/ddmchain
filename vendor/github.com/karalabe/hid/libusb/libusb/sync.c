
#include <config.h>

#include <errno.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "libusbi.h"

static void LIBUSB_CALL sync_transfer_cb(struct libusb_transfer *transfer)
{
	int *completed = transfer->user_data;
	*completed = 1;
	usbi_dbg("actual_length=%d", transfer->actual_length);

}

static void sync_transfer_wait_for_completion(struct libusb_transfer *transfer)
{
	int r, *completed = transfer->user_data;
	struct libusb_context *ctx = HANDLE_CTX(transfer->dev_handle);

	while (!*completed) {
		r = libusb_handle_events_completed(ctx, completed);
		if (r < 0) {
			if (r == LIBUSB_ERROR_INTERRUPTED)
				continue;
			usbi_err(ctx, "libusb_handle_events failed: %s, cancelling transfer and retrying",
				 libusb_error_name(r));
			libusb_cancel_transfer(transfer);
			continue;
		}
	}
}

int API_EXPORTED libusb_control_transfer(libusb_device_handle *dev_handle,
	uint8_t bmRequestType, uint8_t bRequest, uint16_t wValue, uint16_t wIndex,
	unsigned char *data, uint16_t wLength, unsigned int timeout)
{
	struct libusb_transfer *transfer;
	unsigned char *buffer;
	int completed = 0;
	int r;

	if (usbi_handling_events(HANDLE_CTX(dev_handle)))
		return LIBUSB_ERROR_BUSY;

	transfer = libusb_alloc_transfer(0);
	if (!transfer)
		return LIBUSB_ERROR_NO_MEM;

	buffer = (unsigned char*) malloc(LIBUSB_CONTROL_SETUP_SIZE + wLength);
	if (!buffer) {
		libusb_free_transfer(transfer);
		return LIBUSB_ERROR_NO_MEM;
	}

	libusb_fill_control_setup(buffer, bmRequestType, bRequest, wValue, wIndex,
		wLength);
	if ((bmRequestType & LIBUSB_ENDPOINT_DIR_MASK) == LIBUSB_ENDPOINT_OUT)
		memcpy(buffer + LIBUSB_CONTROL_SETUP_SIZE, data, wLength);

	libusb_fill_control_transfer(transfer, dev_handle, buffer,
		sync_transfer_cb, &completed, timeout);
	transfer->flags = LIBUSB_TRANSFER_FREE_BUFFER;
	r = libusb_submit_transfer(transfer);
	if (r < 0) {
		libusb_free_transfer(transfer);
		return r;
	}

	sync_transfer_wait_for_completion(transfer);

	if ((bmRequestType & LIBUSB_ENDPOINT_DIR_MASK) == LIBUSB_ENDPOINT_IN)
		memcpy(data, libusb_control_transfer_get_data(transfer),
			transfer->actual_length);

	switch (transfer->status) {
	case LIBUSB_TRANSFER_COMPLETED:
		r = transfer->actual_length;
		break;
	case LIBUSB_TRANSFER_TIMED_OUT:
		r = LIBUSB_ERROR_TIMEOUT;
		break;
	case LIBUSB_TRANSFER_STALL:
		r = LIBUSB_ERROR_PIPE;
		break;
	case LIBUSB_TRANSFER_NO_DEVICE:
		r = LIBUSB_ERROR_NO_DEVICE;
		break;
	case LIBUSB_TRANSFER_OVERFLOW:
		r = LIBUSB_ERROR_OVERFLOW;
		break;
	case LIBUSB_TRANSFER_ERROR:
	case LIBUSB_TRANSFER_CANCELLED:
		r = LIBUSB_ERROR_IO;
		break;
	default:
		usbi_warn(HANDLE_CTX(dev_handle),
			"unrecognised status code %d", transfer->status);
		r = LIBUSB_ERROR_OTHER;
	}

	libusb_free_transfer(transfer);
	return r;
}

static int do_sync_bulk_transfer(struct libusb_device_handle *dev_handle,
	unsigned char endpoint, unsigned char *buffer, int length,
	int *transferred, unsigned int timeout, unsigned char type)
{
	struct libusb_transfer *transfer;
	int completed = 0;
	int r;

	if (usbi_handling_events(HANDLE_CTX(dev_handle)))
		return LIBUSB_ERROR_BUSY;

	transfer = libusb_alloc_transfer(0);
	if (!transfer)
		return LIBUSB_ERROR_NO_MEM;

	libusb_fill_bulk_transfer(transfer, dev_handle, endpoint, buffer, length,
		sync_transfer_cb, &completed, timeout);
	transfer->type = type;

	r = libusb_submit_transfer(transfer);
	if (r < 0) {
		libusb_free_transfer(transfer);
		return r;
	}

	sync_transfer_wait_for_completion(transfer);

	if (transferred)
		*transferred = transfer->actual_length;

	switch (transfer->status) {
	case LIBUSB_TRANSFER_COMPLETED:
		r = 0;
		break;
	case LIBUSB_TRANSFER_TIMED_OUT:
		r = LIBUSB_ERROR_TIMEOUT;
		break;
	case LIBUSB_TRANSFER_STALL:
		r = LIBUSB_ERROR_PIPE;
		break;
	case LIBUSB_TRANSFER_OVERFLOW:
		r = LIBUSB_ERROR_OVERFLOW;
		break;
	case LIBUSB_TRANSFER_NO_DEVICE:
		r = LIBUSB_ERROR_NO_DEVICE;
		break;
	case LIBUSB_TRANSFER_ERROR:
	case LIBUSB_TRANSFER_CANCELLED:
		r = LIBUSB_ERROR_IO;
		break;
	default:
		usbi_warn(HANDLE_CTX(dev_handle),
			"unrecognised status code %d", transfer->status);
		r = LIBUSB_ERROR_OTHER;
	}

	libusb_free_transfer(transfer);
	return r;
}

int API_EXPORTED libusb_bulk_transfer(struct libusb_device_handle *dev_handle,
	unsigned char endpoint, unsigned char *data, int length, int *transferred,
	unsigned int timeout)
{
	return do_sync_bulk_transfer(dev_handle, endpoint, data, length,
		transferred, timeout, LIBUSB_TRANSFER_TYPE_BULK);
}

int API_EXPORTED libusb_interrupt_transfer(
	struct libusb_device_handle *dev_handle, unsigned char endpoint,
	unsigned char *data, int length, int *transferred, unsigned int timeout)
{
	return do_sync_bulk_transfer(dev_handle, endpoint, data, length,
		transferred, timeout, LIBUSB_TRANSFER_TYPE_INTERRUPT);
}
