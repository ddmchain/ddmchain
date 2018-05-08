
#include <config.h>

#include <stdint.h>
#include <inttypes.h>

#include "libusbi.h"
#include "wince_usb.h"

int windows_version = WINDOWS_CE;
static uint64_t hires_frequency, hires_ticks_to_ps;
static HANDLE driver_handle = INVALID_HANDLE_VALUE;
static int concurrent_usage = -1;

#if defined(ENABLE_LOGGING)
static const char *windows_error_str(DWORD error_code)
{
	static TCHAR wErr_string[ERR_BUFFER_SIZE];
	static char err_string[ERR_BUFFER_SIZE];

	DWORD size;
	int len;

	if (error_code == 0)
		error_code = GetLastError();

	len = sprintf(err_string, "[%u] ", (unsigned int)error_code);

	size = FormatMessage(FORMAT_MESSAGE_FROM_SYSTEM|FORMAT_MESSAGE_IGNORE_INSERTS,
		NULL, error_code, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT),
		wErr_string, ERR_BUFFER_SIZE, NULL);
	if (size == 0) {
		DWORD format_error = GetLastError();
		if (format_error)
			snprintf(err_string, ERR_BUFFER_SIZE,
				"Windows error code %u (FormatMessage error code %u)",
				(unsigned int)error_code, (unsigned int)format_error);
		else
			snprintf(err_string, ERR_BUFFER_SIZE, "Unknown error code %u", (unsigned int)error_code);
	} else {

		size_t pos = size - 2;
		if (wErr_string[pos] == 0x0D)
			wErr_string[pos] = 0;

		if (!WideCharToMultiByte(CP_ACP, 0, wErr_string, -1, &err_string[len], ERR_BUFFER_SIZE - len, NULL, NULL))
			strcpy(err_string, "Unable to convert error string");
	}

	return err_string;
}
#endif

static struct wince_device_priv *_device_priv(struct libusb_device *dev)
{
	return (struct wince_device_priv *)dev->os_priv;
}

static int translate_driver_error(DWORD error)
{
	switch (error) {
	case ERROR_INVALID_PARAMETER:
		return LIBUSB_ERROR_INVALID_PARAM;
	case ERROR_CALL_NOT_IMPLEMENTED:
	case ERROR_NOT_SUPPORTED:
		return LIBUSB_ERROR_NOT_SUPPORTED;
	case ERROR_NOT_ENOUGH_MEMORY:
		return LIBUSB_ERROR_NO_MEM;
	case ERROR_INVALID_HANDLE:
		return LIBUSB_ERROR_NO_DEVICE;
	case ERROR_BUSY:
		return LIBUSB_ERROR_BUSY;

	case ERROR_CANCELLED:
	case ERROR_INTERNAL_ERROR:
	default:
		return LIBUSB_ERROR_OTHER;
	}
}

static int init_dllimports(void)
{
	DLL_GET_HANDLE(ceusbkwrapper);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwOpenDriver, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwGetDeviceList, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwReleaseDeviceList, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwGetDeviceAddress, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwGetDeviceDescriptor, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwGetConfigDescriptor, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwCloseDriver, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwCancelTransfer, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwIssueControlTransfer, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwClaimInterface, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwReleaseInterface, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwSetInterfaceAlternateSetting, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwClearHaltHost, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwClearHaltDevice, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwGetConfig, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwSetConfig, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwResetDevice, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwKernelDriverActive, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwAttachKernelDriver, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwDetachKernelDriver, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwIssueBulkTransfer, TRUE);
	DLL_LOAD_FUNC(ceusbkwrapper, UkwIsPipeHalted, TRUE);

	return LIBUSB_SUCCESS;
}

static void exit_dllimports(void)
{
	DLL_FREE_HANDLE(ceusbkwrapper);
}

static int init_device(
	struct libusb_device *dev, UKW_DEVICE drv_dev,
	unsigned char bus_addr, unsigned char dev_addr)
{
	struct wince_device_priv *priv = _device_priv(dev);
	int r = LIBUSB_SUCCESS;

	dev->bus_number = bus_addr;
	dev->device_address = dev_addr;
	priv->dev = drv_dev;

	if (!UkwGetDeviceDescriptor(priv->dev, &(priv->desc)))
		r = translate_driver_error(GetLastError());

	return r;
}

static int wince_init(struct libusb_context *ctx)
{
	int r = LIBUSB_ERROR_OTHER;
	HANDLE semaphore;
	LARGE_INTEGER li_frequency;
	TCHAR sem_name[11 + 8 + 1]; 

	_stprintf(sem_name, _T("libusb_init%08X"), (unsigned int)(GetCurrentProcessId() & 0xFFFFFFFF));
	semaphore = CreateSemaphore(NULL, 1, 1, sem_name);
	if (semaphore == NULL) {
		usbi_err(ctx, "could not create semaphore: %s", windows_error_str(0));
		return LIBUSB_ERROR_NO_MEM;
	}

	if (WaitForSingleObject(semaphore, INFINITE) != WAIT_OBJECT_0) {
		usbi_err(ctx, "failure to access semaphore: %s", windows_error_str(0));
		CloseHandle(semaphore);
		return LIBUSB_ERROR_NO_MEM;
	}

	if ( ++concurrent_usage == 0 ) {	

		init_polling();

		if (init_dllimports() != LIBUSB_SUCCESS) {
			usbi_err(ctx, "could not resolve DLL functions");
			r = LIBUSB_ERROR_NOT_SUPPORTED;
			goto init_exit;
		}

		driver_handle = UkwOpenDriver();
		if (driver_handle == INVALID_HANDLE_VALUE) {
			usbi_err(ctx, "could not connect to driver");
			r = LIBUSB_ERROR_NOT_SUPPORTED;
			goto init_exit;
		}

		if (QueryPerformanceFrequency(&li_frequency)) {
			hires_frequency = li_frequency.QuadPart;

			hires_ticks_to_ps = UINT64_C(1000000000000) / hires_frequency;
			usbi_dbg("hires timer available (Frequency: %"PRIu64" Hz)", hires_frequency);
		} else {
			usbi_dbg("no hires timer available on this platform");
			hires_frequency = 0;
			hires_ticks_to_ps = UINT64_C(0);
		}
	}

	r = LIBUSB_SUCCESS;

init_exit: 
	if (!concurrent_usage && r != LIBUSB_SUCCESS) { 
		exit_dllimports();
		exit_polling();

		if (driver_handle != INVALID_HANDLE_VALUE) {
			UkwCloseDriver(driver_handle);
			driver_handle = INVALID_HANDLE_VALUE;
		}
	}

	if (r != LIBUSB_SUCCESS)
		--concurrent_usage; 

	ReleaseSemaphore(semaphore, 1, NULL);	
	CloseHandle(semaphore);
	return r;
}

static void wince_exit(void)
{
	HANDLE semaphore;
	TCHAR sem_name[11 + 8 + 1]; 

	_stprintf(sem_name, _T("libusb_init%08X"), (unsigned int)(GetCurrentProcessId() & 0xFFFFFFFF));
	semaphore = CreateSemaphore(NULL, 1, 1, sem_name);
	if (semaphore == NULL)
		return;

	if (WaitForSingleObject(semaphore, INFINITE) != WAIT_OBJECT_0) {
		CloseHandle(semaphore);
		return;
	}

	if (--concurrent_usage < 0) {	
		exit_dllimports();
		exit_polling();

		if (driver_handle != INVALID_HANDLE_VALUE) {
			UkwCloseDriver(driver_handle);
			driver_handle = INVALID_HANDLE_VALUE;
		}
	}

	ReleaseSemaphore(semaphore, 1, NULL);	
	CloseHandle(semaphore);
}

static int wince_get_device_list(
	struct libusb_context *ctx,
	struct discovered_devs **discdevs)
{
	UKW_DEVICE devices[MAX_DEVICE_COUNT];
	struct discovered_devs *new_devices = *discdevs;
	DWORD count = 0, i;
	struct libusb_device *dev = NULL;
	unsigned char bus_addr, dev_addr;
	unsigned long session_id;
	BOOL success;
	DWORD release_list_offset = 0;
	int r = LIBUSB_SUCCESS;

	success = UkwGetDeviceList(driver_handle, devices, MAX_DEVICE_COUNT, &count);
	if (!success) {
		int libusbErr = translate_driver_error(GetLastError());
		usbi_err(ctx, "could not get devices: %s", windows_error_str(0));
		return libusbErr;
	}

	for (i = 0; i < count; ++i) {
		release_list_offset = i;
		success = UkwGetDeviceAddress(devices[i], &bus_addr, &dev_addr, &session_id);
		if (!success) {
			r = translate_driver_error(GetLastError());
			usbi_err(ctx, "could not get device address for %u: %s", (unsigned int)i, windows_error_str(0));
			goto err_out;
		}

		dev = usbi_get_device_by_session_id(ctx, session_id);
		if (dev) {
			usbi_dbg("using existing device for %u/%u (session %lu)",
					bus_addr, dev_addr, session_id);

			UkwReleaseDeviceList(driver_handle, &devices[i], 1);
			release_list_offset++;
		} else {
			usbi_dbg("allocating new device for %u/%u (session %lu)",
					bus_addr, dev_addr, session_id);
			dev = usbi_alloc_device(ctx, session_id);
			if (!dev) {
				r = LIBUSB_ERROR_NO_MEM;
				goto err_out;
			}

			r = init_device(dev, devices[i], bus_addr, dev_addr);
			if (r < 0)
				goto err_out;

			r = usbi_sanitize_device(dev);
			if (r < 0)
				goto err_out;
		}

		new_devices = discovered_devs_append(new_devices, dev);
		if (!discdevs) {
			r = LIBUSB_ERROR_NO_MEM;
			goto err_out;
		}

		libusb_unref_device(dev);
	}

	*discdevs = new_devices;
	return r;
err_out:
	*discdevs = new_devices;
	libusb_unref_device(dev);

	UkwReleaseDeviceList(driver_handle, &devices[release_list_offset], count - release_list_offset);
	return r;
}

static int wince_open(struct libusb_device_handle *handle)
{

	return LIBUSB_SUCCESS;
}

static void wince_close(struct libusb_device_handle *handle)
{

}

static int wince_get_device_descriptor(
	struct libusb_device *device,
	unsigned char *buffer, int *host_endian)
{
	struct wince_device_priv *priv = _device_priv(device);

	*host_endian = 1;
	memcpy(buffer, &priv->desc, DEVICE_DESC_LENGTH);
	return LIBUSB_SUCCESS;
}

static int wince_get_active_config_descriptor(
	struct libusb_device *device,
	unsigned char *buffer, size_t len, int *host_endian)
{
	struct wince_device_priv *priv = _device_priv(device);
	DWORD actualSize = len;

	*host_endian = 0;
	if (!UkwGetConfigDescriptor(priv->dev, UKW_ACTIVE_CONFIGURATION, buffer, len, &actualSize))
		return translate_driver_error(GetLastError());

	return actualSize;
}

static int wince_get_config_descriptor(
	struct libusb_device *device,
	uint8_t config_index,
	unsigned char *buffer, size_t len, int *host_endian)
{
	struct wince_device_priv *priv = _device_priv(device);
	DWORD actualSize = len;

	*host_endian = 0;
	if (!UkwGetConfigDescriptor(priv->dev, config_index, buffer, len, &actualSize))
		return translate_driver_error(GetLastError());

	return actualSize;
}

static int wince_get_configuration(
	struct libusb_device_handle *handle,
	int *config)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);
	UCHAR cv = 0;

	if (!UkwGetConfig(priv->dev, &cv))
		return translate_driver_error(GetLastError());

	(*config) = cv;
	return LIBUSB_SUCCESS;
}

static int wince_set_configuration(
	struct libusb_device_handle *handle,
	int config)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);

	UCHAR cv = (config < 0) ? 0 : config;
	if (!UkwSetConfig(priv->dev, cv))
		return translate_driver_error(GetLastError());

	return LIBUSB_SUCCESS;
}

static int wince_claim_interface(
	struct libusb_device_handle *handle,
	int interface_number)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);

	if (!UkwClaimInterface(priv->dev, interface_number))
		return translate_driver_error(GetLastError());

	return LIBUSB_SUCCESS;
}

static int wince_release_interface(
	struct libusb_device_handle *handle,
	int interface_number)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);

	if (!UkwSetInterfaceAlternateSetting(priv->dev, interface_number, 0))
		return translate_driver_error(GetLastError());

	if (!UkwReleaseInterface(priv->dev, interface_number))
		return translate_driver_error(GetLastError());

	return LIBUSB_SUCCESS;
}

static int wince_set_interface_altsetting(
	struct libusb_device_handle *handle,
	int interface_number, int altsetting)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);

	if (!UkwSetInterfaceAlternateSetting(priv->dev, interface_number, altsetting))
		return translate_driver_error(GetLastError());

	return LIBUSB_SUCCESS;
}

static int wince_clear_halt(
	struct libusb_device_handle *handle,
	unsigned char endpoint)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);

	if (!UkwClearHaltHost(priv->dev, endpoint))
		return translate_driver_error(GetLastError());

	if (!UkwClearHaltDevice(priv->dev, endpoint))
		return translate_driver_error(GetLastError());

	return LIBUSB_SUCCESS;
}

static int wince_reset_device(
	struct libusb_device_handle *handle)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);

	if (!UkwResetDevice(priv->dev))
		return translate_driver_error(GetLastError());

	return LIBUSB_SUCCESS;
}

static int wince_kernel_driver_active(
	struct libusb_device_handle *handle,
	int interface_number)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);
	BOOL result = FALSE;

	if (!UkwKernelDriverActive(priv->dev, interface_number, &result))
		return translate_driver_error(GetLastError());

	return result ? 1 : 0;
}

static int wince_detach_kernel_driver(
	struct libusb_device_handle *handle,
	int interface_number)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);

	if (!UkwDetachKernelDriver(priv->dev, interface_number))
		return translate_driver_error(GetLastError());

	return LIBUSB_SUCCESS;
}

static int wince_attach_kernel_driver(
	struct libusb_device_handle *handle,
	int interface_number)
{
	struct wince_device_priv *priv = _device_priv(handle->dev);

	if (!UkwAttachKernelDriver(priv->dev, interface_number))
		return translate_driver_error(GetLastError());

	return LIBUSB_SUCCESS;
}

static void wince_destroy_device(struct libusb_device *dev)
{
	struct wince_device_priv *priv = _device_priv(dev);

	UkwReleaseDeviceList(driver_handle, &priv->dev, 1);
}

static void wince_clear_transfer_priv(struct usbi_transfer *itransfer)
{
	struct wince_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winfd wfd = fd_to_winfd(transfer_priv->pollable_fd.fd);

	wfd.itransfer = NULL;
	CloseHandle(wfd.handle);
	usbi_free_fd(&transfer_priv->pollable_fd);
}

static int wince_cancel_transfer(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct wince_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	struct wince_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);

	if (!UkwCancelTransfer(priv->dev, transfer_priv->pollable_fd.overlapped, UKW_TF_NO_WAIT))
		return translate_driver_error(GetLastError());

	return LIBUSB_SUCCESS;
}

static int wince_submit_control_or_bulk_transfer(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct libusb_context *ctx = DEVICE_CTX(transfer->dev_handle->dev);
	struct wince_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct wince_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	BOOL direction_in, ret;
	struct winfd wfd;
	DWORD flags;
	HANDLE eventHandle;
	PUKW_CONTROL_HEADER setup = NULL;
	const BOOL control_transfer = transfer->type == LIBUSB_TRANSFER_TYPE_CONTROL;

	transfer_priv->pollable_fd = INVALID_WINFD;
	if (control_transfer) {
		setup = (PUKW_CONTROL_HEADER) transfer->buffer;
		direction_in = setup->bmRequestType & LIBUSB_ENDPOINT_IN;
	} else {
		direction_in = transfer->endpoint & LIBUSB_ENDPOINT_IN;
	}
	flags = direction_in ? UKW_TF_IN_TRANSFER : UKW_TF_OUT_TRANSFER;
	flags |= UKW_TF_SHORT_TRANSFER_OK;

	eventHandle = CreateEvent(NULL, FALSE, FALSE, NULL);
	if (eventHandle == NULL) {
		usbi_err(ctx, "Failed to create event for async transfer");
		return LIBUSB_ERROR_NO_MEM;
	}

	wfd = usbi_create_fd(eventHandle, direction_in ? RW_READ : RW_WRITE, itransfer, &wince_cancel_transfer);
	if (wfd.fd < 0) {
		CloseHandle(eventHandle);
		return LIBUSB_ERROR_NO_MEM;
	}

	transfer_priv->pollable_fd = wfd;
	if (control_transfer) {

		DWORD bufLen = transfer->length - sizeof(UKW_CONTROL_HEADER);
		PVOID buf = (PVOID) &transfer->buffer[sizeof(UKW_CONTROL_HEADER)];

		ret = UkwIssueControlTransfer(priv->dev, flags, setup, buf, bufLen, &transfer->actual_length, wfd.overlapped);
	} else {
		ret = UkwIssueBulkTransfer(priv->dev, flags, transfer->endpoint, transfer->buffer,
			transfer->length, &transfer->actual_length, wfd.overlapped);
	}

	if (!ret) {
		int libusbErr = translate_driver_error(GetLastError());
		usbi_err(ctx, "UkwIssue%sTransfer failed: error %u",
			control_transfer ? "Control" : "Bulk", (unsigned int)GetLastError());
		wince_clear_transfer_priv(itransfer);
		return libusbErr;
	}
	usbi_add_pollfd(ctx, transfer_priv->pollable_fd.fd, direction_in ? POLLIN : POLLOUT);

	return LIBUSB_SUCCESS;
}

static int wince_submit_iso_transfer(struct usbi_transfer *itransfer)
{
	return LIBUSB_ERROR_NOT_SUPPORTED;
}

static int wince_submit_transfer(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);

	switch (transfer->type) {
	case LIBUSB_TRANSFER_TYPE_CONTROL:
	case LIBUSB_TRANSFER_TYPE_BULK:
	case LIBUSB_TRANSFER_TYPE_INTERRUPT:
		return wince_submit_control_or_bulk_transfer(itransfer);
	case LIBUSB_TRANSFER_TYPE_ISOCHRONOUS:
		return wince_submit_iso_transfer(itransfer);
	case LIBUSB_TRANSFER_TYPE_BULK_STREAM:
		return LIBUSB_ERROR_NOT_SUPPORTED;
	default:
		usbi_err(TRANSFER_CTX(transfer), "unknown endpoint type %d", transfer->type);
		return LIBUSB_ERROR_INVALID_PARAM;
	}
}

static void wince_transfer_callback(
	struct usbi_transfer *itransfer,
	uint32_t io_result, uint32_t io_size)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct wince_transfer_priv *transfer_priv = (struct wince_transfer_priv*)usbi_transfer_get_os_priv(itransfer);
	struct wince_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	int status;

	usbi_dbg("handling I/O completion with errcode %u", io_result);

	if (io_result == ERROR_NOT_SUPPORTED &&
		transfer->type != LIBUSB_TRANSFER_TYPE_CONTROL) {

		BOOL halted = FALSE;
		usbi_dbg("checking I/O completion with errcode ERROR_NOT_SUPPORTED is really a stall");
		if (UkwIsPipeHalted(priv->dev, transfer->endpoint, &halted)) {

			WORD wStatus = 0;
			DWORD written = 0;
			UKW_CONTROL_HEADER ctrlHeader;
			ctrlHeader.bmRequestType = LIBUSB_REQUEST_TYPE_STANDARD |
				LIBUSB_ENDPOINT_IN | LIBUSB_RECIPIENT_ENDPOINT;
			ctrlHeader.bRequest = LIBUSB_REQUEST_GET_STATUS;
			ctrlHeader.wValue = 0;
			ctrlHeader.wIndex = transfer->endpoint;
			ctrlHeader.wLength = sizeof(wStatus);
			if (UkwIssueControlTransfer(priv->dev,
					UKW_TF_IN_TRANSFER | UKW_TF_SEND_TO_ENDPOINT,
					&ctrlHeader, &wStatus, sizeof(wStatus), &written, NULL)) {
				if (written == sizeof(wStatus) &&
						(wStatus & STATUS_HALT_FLAG) == 0) {
					if (!halted || UkwClearHaltHost(priv->dev, transfer->endpoint)) {
						usbi_dbg("Endpoint doesn't appear to be stalled, overriding error with success");
						io_result = ERROR_SUCCESS;
					} else {
						usbi_dbg("Endpoint doesn't appear to be stalled, but the host is halted, changing error");
						io_result = ERROR_IO_DEVICE;
					}
				}
			}
		}
	}

	switch(io_result) {
	case ERROR_SUCCESS:
		itransfer->transferred += io_size;
		status = LIBUSB_TRANSFER_COMPLETED;
		break;
	case ERROR_CANCELLED:
		usbi_dbg("detected transfer cancel");
		status = LIBUSB_TRANSFER_CANCELLED;
		break;
	case ERROR_NOT_SUPPORTED:
	case ERROR_GEN_FAILURE:
		usbi_dbg("detected endpoint stall");
		status = LIBUSB_TRANSFER_STALL;
		break;
	case ERROR_SEM_TIMEOUT:
		usbi_dbg("detected semaphore timeout");
		status = LIBUSB_TRANSFER_TIMED_OUT;
		break;
	case ERROR_OPERATION_ABORTED:
		usbi_dbg("detected operation aborted");
		status = LIBUSB_TRANSFER_CANCELLED;
		break;
	default:
		usbi_err(ITRANSFER_CTX(itransfer), "detected I/O error: %s", windows_error_str(io_result));
		status = LIBUSB_TRANSFER_ERROR;
		break;
	}

	wince_clear_transfer_priv(itransfer);
	if (status == LIBUSB_TRANSFER_CANCELLED)
		usbi_handle_transfer_cancellation(itransfer);
	else
		usbi_handle_transfer_completion(itransfer, (enum libusb_transfer_status)status);
}

static void wince_handle_callback(
	struct usbi_transfer *itransfer,
	uint32_t io_result, uint32_t io_size)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);

	switch (transfer->type) {
	case LIBUSB_TRANSFER_TYPE_CONTROL:
	case LIBUSB_TRANSFER_TYPE_BULK:
	case LIBUSB_TRANSFER_TYPE_INTERRUPT:
	case LIBUSB_TRANSFER_TYPE_ISOCHRONOUS:
		wince_transfer_callback (itransfer, io_result, io_size);
		break;
	case LIBUSB_TRANSFER_TYPE_BULK_STREAM:
		break;
	default:
		usbi_err(ITRANSFER_CTX(itransfer), "unknown endpoint type %d", transfer->type);
	}
}

static int wince_handle_events(
	struct libusb_context *ctx,
	struct pollfd *fds, POLL_NFDS_TYPE nfds, int num_ready)
{
	struct wince_transfer_priv* transfer_priv = NULL;
	POLL_NFDS_TYPE i = 0;
	BOOL found = FALSE;
	struct usbi_transfer *transfer;
	DWORD io_size, io_result;
	int r = LIBUSB_SUCCESS;

	usbi_mutex_lock(&ctx->open_devs_lock);
	for (i = 0; i < nfds && num_ready > 0; i++) {

		usbi_dbg("checking fd %d with revents = %04x", fds[i].fd, fds[i].revents);

		if (!fds[i].revents)
			continue;

		num_ready--;

		usbi_mutex_lock(&ctx->flying_transfers_lock);
		list_for_each_entry(transfer, &ctx->flying_transfers, list, struct usbi_transfer) {
			transfer_priv = usbi_transfer_get_os_priv(transfer);
			if (transfer_priv->pollable_fd.fd == fds[i].fd) {
				found = TRUE;
				break;
			}
		}
		usbi_mutex_unlock(&ctx->flying_transfers_lock);

		if (found && HasOverlappedIoCompleted(transfer_priv->pollable_fd.overlapped)) {
			io_result = (DWORD)transfer_priv->pollable_fd.overlapped->Internal;
			io_size = (DWORD)transfer_priv->pollable_fd.overlapped->InternalHigh;
			usbi_remove_pollfd(ctx, transfer_priv->pollable_fd.fd);

			wince_handle_callback(transfer, io_result, io_size);
		} else if (found) {
			usbi_err(ctx, "matching transfer for fd %d has not completed", fds[i]);
			r = LIBUSB_ERROR_OTHER;
			break;
		} else {
			usbi_err(ctx, "could not find a matching transfer for fd %d", fds[i]);
			r = LIBUSB_ERROR_NOT_FOUND;
			break;
		}
	}
	usbi_mutex_unlock(&ctx->open_devs_lock);

	return r;
}

static int wince_clock_gettime(int clk_id, struct timespec *tp)
{
	LARGE_INTEGER hires_counter;
	ULARGE_INTEGER rtime;
	FILETIME filetime;
	SYSTEMTIME st;

	switch(clk_id) {
	case USBI_CLOCK_MONOTONIC:
		if (hires_frequency != 0 && QueryPerformanceCounter(&hires_counter)) {
			tp->tv_sec = (long)(hires_counter.QuadPart / hires_frequency);
			tp->tv_nsec = (long)(((hires_counter.QuadPart % hires_frequency) / 1000) * hires_ticks_to_ps);
			return LIBUSB_SUCCESS;
		}

	case USBI_CLOCK_REALTIME:

		GetSystemTime(&st);
		SystemTimeToFileTime(&st, &filetime);
		rtime.LowPart = filetime.dwLowDateTime;
		rtime.HighPart = filetime.dwHighDateTime;
		rtime.QuadPart -= EPOCH_TIME;
		tp->tv_sec = (long)(rtime.QuadPart / 10000000);
		tp->tv_nsec = (long)((rtime.QuadPart % 10000000)*100);
		return LIBUSB_SUCCESS;
	default:
		return LIBUSB_ERROR_INVALID_PARAM;
	}
}

const struct usbi_os_backend wince_backend = {
	"Windows CE",
	0,
	wince_init,
	wince_exit,

	wince_get_device_list,
	NULL,				
	wince_open,
	wince_close,

	wince_get_device_descriptor,
	wince_get_active_config_descriptor,
	wince_get_config_descriptor,
	NULL,				

	wince_get_configuration,
	wince_set_configuration,
	wince_claim_interface,
	wince_release_interface,

	wince_set_interface_altsetting,
	wince_clear_halt,
	wince_reset_device,

	NULL,				
	NULL,				

	NULL,				
	NULL,				

	wince_kernel_driver_active,
	wince_detach_kernel_driver,
	wince_attach_kernel_driver,

	wince_destroy_device,

	wince_submit_transfer,
	wince_cancel_transfer,
	wince_clear_transfer_priv,

	wince_handle_events,
	NULL,				

	wince_clock_gettime,
	sizeof(struct wince_device_priv),
	0,
	sizeof(struct wince_transfer_priv),
};
