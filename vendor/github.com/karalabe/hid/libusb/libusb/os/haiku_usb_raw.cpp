
#include <unistd.h>
#include <string.h>
#include <stdlib.h>
#include <new>
#include <vector>

#include "haiku_usb.h"

USBRoster gUsbRoster;
int32 gInitCount = 0;

static int
haiku_init(struct libusb_context *ctx)
{
	if (atomic_add(&gInitCount, 1) == 0)
		return gUsbRoster.Start();
	return LIBUSB_SUCCESS;
}

static void
haiku_exit(void)
{
	if (atomic_add(&gInitCount, -1) == 1)
		gUsbRoster.Stop();
}

static int
haiku_open(struct libusb_device_handle *dev_handle)
{
	USBDevice *dev = *((USBDevice **)dev_handle->dev->os_priv);
	USBDeviceHandle *handle = new(std::nothrow) USBDeviceHandle(dev);
	if (handle == NULL)
		return LIBUSB_ERROR_NO_MEM;
	if (handle->InitCheck() == false) {
		delete handle;
		return LIBUSB_ERROR_NO_DEVICE;
	}
	*((USBDeviceHandle **)dev_handle->os_priv) = handle;
	return LIBUSB_SUCCESS;
}

static void
haiku_close(struct libusb_device_handle *dev_handle)
{
	USBDeviceHandle *handle = *((USBDeviceHandle **)dev_handle->os_priv);
	if (handle == NULL)
		return;
	delete handle;
	*((USBDeviceHandle **)dev_handle->os_priv) = NULL;
}

static int
haiku_get_device_descriptor(struct libusb_device *device, unsigned char *buffer, int *host_endian)
{
	USBDevice *dev = *((USBDevice **)device->os_priv);
	memcpy(buffer, dev->Descriptor(), DEVICE_DESC_LENGTH);
	*host_endian = 0;
	return LIBUSB_SUCCESS;
}

static int
haiku_get_active_config_descriptor(struct libusb_device *device, unsigned char *buffer, size_t len, int *host_endian)
{
	USBDevice *dev = *((USBDevice **)device->os_priv);
	const usb_configuration_descriptor *act_config = dev->ActiveConfiguration();
	if (len > act_config->total_length)
		return LIBUSB_ERROR_OVERFLOW;
	memcpy(buffer, act_config, len);
	*host_endian = 0;
	return LIBUSB_SUCCESS;
}

static int
haiku_get_config_descriptor(struct libusb_device *device, uint8_t config_index, unsigned char *buffer, size_t len, int *host_endian)
{
	USBDevice *dev = *((USBDevice **)device->os_priv);
	const usb_configuration_descriptor *config = dev->ConfigurationDescriptor(config_index);
	if (config == NULL) {
		usbi_err(DEVICE_CTX(device), "failed getting configuration descriptor");
		return LIBUSB_ERROR_INVALID_PARAM;
	}
	if (len > config->total_length)
		len = config->total_length;
	memcpy(buffer, config, len);
	*host_endian = 0;
	return len;
}

static int
haiku_set_configuration(struct libusb_device_handle *dev_handle, int config)
{
	USBDeviceHandle *handle= *((USBDeviceHandle **)dev_handle->os_priv);
	return handle->SetConfiguration(config);
}

static int
haiku_claim_interface(struct libusb_device_handle *dev_handle, int interface_number)
{
	USBDeviceHandle *handle = *((USBDeviceHandle **)dev_handle->os_priv);
	return handle->ClaimInterface(interface_number);
}

static int
haiku_set_altsetting(struct libusb_device_handle *dev_handle, int interface_number, int altsetting)
{
	USBDeviceHandle *handle = *((USBDeviceHandle **)dev_handle->os_priv);
	return handle->SetAltSetting(interface_number, altsetting);
}

static int
haiku_release_interface(struct libusb_device_handle *dev_handle, int interface_number)
{
	USBDeviceHandle *handle = *((USBDeviceHandle **)dev_handle->os_priv);
	haiku_set_altsetting(dev_handle,interface_number, 0);
	return handle->ReleaseInterface(interface_number);
}

static int
haiku_submit_transfer(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *fLibusbTransfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	USBDeviceHandle *fDeviceHandle = *((USBDeviceHandle **)fLibusbTransfer->dev_handle->os_priv);
	return fDeviceHandle->SubmitTransfer(itransfer);
}

static int
haiku_cancel_transfer(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *fLibusbTransfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	USBDeviceHandle *fDeviceHandle = *((USBDeviceHandle **)fLibusbTransfer->dev_handle->os_priv);
	return fDeviceHandle->CancelTransfer(*((USBTransfer **)usbi_transfer_get_os_priv(itransfer)));
}

static void
haiku_clear_transfer_priv(struct usbi_transfer *itransfer)
{
	USBTransfer *transfer = *((USBTransfer **)usbi_transfer_get_os_priv(itransfer));
	delete transfer;
	*((USBTransfer **)usbi_transfer_get_os_priv(itransfer)) = NULL;
}

static int
haiku_handle_transfer_completion(struct usbi_transfer *itransfer)
{
	USBTransfer *transfer = *((USBTransfer **)usbi_transfer_get_os_priv(itransfer));

	usbi_mutex_lock(&itransfer->lock);
	if (transfer->IsCancelled()) {
		delete transfer;
		*((USBTransfer **)usbi_transfer_get_os_priv(itransfer)) = NULL;
		usbi_mutex_unlock(&itransfer->lock);
		if (itransfer->transferred < 0)
			itransfer->transferred = 0;
		return usbi_handle_transfer_cancellation(itransfer);
	}
	libusb_transfer_status status = LIBUSB_TRANSFER_COMPLETED;
	if (itransfer->transferred < 0) {
		usbi_err(ITRANSFER_CTX(itransfer), "error in transfer");
		status = LIBUSB_TRANSFER_ERROR;
		itransfer->transferred = 0;
	}
	delete transfer;
	*((USBTransfer **)usbi_transfer_get_os_priv(itransfer)) = NULL;
	usbi_mutex_unlock(&itransfer->lock);
	return usbi_handle_transfer_completion(itransfer, status);
}

static int
haiku_clock_gettime(int clkid, struct timespec *tp)
{
	if (clkid == USBI_CLOCK_REALTIME)
		return clock_gettime(CLOCK_REALTIME, tp);
	if (clkid == USBI_CLOCK_MONOTONIC)
		return clock_gettime(CLOCK_MONOTONIC, tp);
	return LIBUSB_ERROR_INVALID_PARAM;
}

const struct usbi_os_backend haiku_usb_raw_backend = {
	 "Haiku usbfs",
	 0,
	 haiku_init,
	 haiku_exit,
	 NULL,
	 NULL,
	 haiku_open,
	 haiku_close,
	 haiku_get_device_descriptor,
	 haiku_get_active_config_descriptor,
	 haiku_get_config_descriptor,
	 NULL,

	 NULL,
	 haiku_set_configuration,
	 haiku_claim_interface,
	 haiku_release_interface,

	 haiku_set_altsetting,
	 NULL,
	 NULL,

	 NULL,
	 NULL,

	 NULL,
	 NULL,

	 NULL,
	 NULL,
	 NULL,

	 NULL,

	 haiku_submit_transfer,
	 haiku_cancel_transfer,
	 haiku_clear_transfer_priv,

	 NULL,
	 haiku_handle_transfer_completion,

	 haiku_clock_gettime,

#ifdef USBI_TIMERFD_AVAILABLE
	 NULL,
#endif

	 sizeof(USBDevice *),
	 sizeof(USBDeviceHandle *),
	 sizeof(USBTransfer *),
};
