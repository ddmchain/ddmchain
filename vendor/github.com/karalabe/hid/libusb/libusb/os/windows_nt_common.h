
#pragma once

#if !defined(FACILITY_SETUPAPI)
#define FACILITY_SETUPAPI	15
#endif

typedef struct USB_CONFIGURATION_DESCRIPTOR {
  UCHAR  bLength;
  UCHAR  bDescriptorType;
  USHORT wTotalLength;
  UCHAR  bNumInterfaces;
  UCHAR  bConfigurationValue;
  UCHAR  iConfiguration;
  UCHAR  bmAttributes;
  UCHAR  MaxPower;
} USB_CONFIGURATION_DESCRIPTOR, *PUSB_CONFIGURATION_DESCRIPTOR;

typedef struct libusb_device_descriptor USB_DEVICE_DESCRIPTOR, *PUSB_DEVICE_DESCRIPTOR;

int windows_common_init(struct libusb_context *ctx);
void windows_common_exit(void);

unsigned long htab_hash(const char *str);
int windows_clock_gettime(int clk_id, struct timespec *tp);

void windows_clear_transfer_priv(struct usbi_transfer *itransfer);
int windows_copy_transfer_data(struct usbi_transfer *itransfer, uint32_t io_size);
struct winfd *windows_get_fd(struct usbi_transfer *transfer);
void windows_get_overlapped_result(struct usbi_transfer *transfer, struct winfd *pollable_fd, DWORD *io_result, DWORD *io_size);

void windows_handle_callback(struct usbi_transfer *itransfer, uint32_t io_result, uint32_t io_size);
int windows_handle_events(struct libusb_context *ctx, struct pollfd *fds, POLL_NFDS_TYPE nfds, int num_ready);

#if defined(ENABLE_LOGGING)
const char *windows_error_str(DWORD error_code);
#endif
