
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <ctype.h>
#include <locale.h>
#include <errno.h>

#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/ioctl.h>
#include <sys/utsname.h>
#include <fcntl.h>
#include <pthread.h>
#include <wchar.h>

#include <libusb.h>
#ifndef __ANDROID__
#include <iconv.h>
#endif

#include "hidapi.h"

#ifdef __ANDROID__

typedef int pthread_barrierattr_t;
typedef struct pthread_barrier {
    pthread_mutex_t mutex;
    pthread_cond_t cond;
    int count;
    int trip_count;
} pthread_barrier_t;

static int pthread_barrier_init(pthread_barrier_t *barrier, const pthread_barrierattr_t *attr, unsigned int count)
{
	if(count == 0) {
		errno = EINVAL;
		return -1;
	}

	if(pthread_mutex_init(&barrier->mutex, 0) < 0) {
		return -1;
	}
	if(pthread_cond_init(&barrier->cond, 0) < 0) {
		pthread_mutex_destroy(&barrier->mutex);
		return -1;
	}
	barrier->trip_count = count;
	barrier->count = 0;

	return 0;
}

static int pthread_barrier_destroy(pthread_barrier_t *barrier)
{
	pthread_cond_destroy(&barrier->cond);
	pthread_mutex_destroy(&barrier->mutex);
	return 0;
}

static int pthread_barrier_wait(pthread_barrier_t *barrier)
{
	pthread_mutex_lock(&barrier->mutex);
	++(barrier->count);
	if(barrier->count >= barrier->trip_count)
	{
		barrier->count = 0;
		pthread_cond_broadcast(&barrier->cond);
		pthread_mutex_unlock(&barrier->mutex);
		return 1;
	}
	else
	{
		pthread_cond_wait(&barrier->cond, &(barrier->mutex));
		pthread_mutex_unlock(&barrier->mutex);
		return 0;
	}
}

#endif

#ifdef __cplusplus
extern "C" {
#endif

#ifdef DEBUG_PRINTF
#define LOG(...) fprintf(stderr, __VA_ARGS__)
#else
#define LOG(...) do {} while (0)
#endif

#ifndef __FreeBSD__
#define DETACH_KERNEL_DRIVER
#endif

struct input_report {
	uint8_t *data;
	size_t len;
	struct input_report *next;
};

struct hid_device_ {

	libusb_device_handle *device_handle;

	int input_endpoint;
	int output_endpoint;
	int input_ep_max_packet_size;

	int interface;

	int manufacturer_index;
	int product_index;
	int serial_index;

	int blocking; 

	pthread_t thread;
	pthread_mutex_t mutex; 
	pthread_cond_t condition;
	pthread_barrier_t barrier; 
	int shutdown_thread;
	int cancelled;
	struct libusb_transfer *transfer;

	struct input_report *input_reports;
};

static libusb_context *usb_context = NULL;

uint16_t get_usb_code_for_current_locale(void);
static int return_data(hid_device *dev, unsigned char *data, size_t length);

static hid_device *new_hid_device(void)
{
	hid_device *dev = calloc(1, sizeof(hid_device));
	dev->blocking = 1;

	pthread_mutex_init(&dev->mutex, NULL);
	pthread_cond_init(&dev->condition, NULL);
	pthread_barrier_init(&dev->barrier, NULL, 2);

	return dev;
}

static void free_hid_device(hid_device *dev)
{

	pthread_barrier_destroy(&dev->barrier);
	pthread_cond_destroy(&dev->condition);
	pthread_mutex_destroy(&dev->mutex);

	free(dev);
}

#if 0

static void register_error(hid_device *device, const char *op)
{

}
#endif

#ifdef INVASIVE_GET_USAGE

static uint32_t get_bytes(uint8_t *rpt, size_t len, size_t num_bytes, size_t cur)
{

	if (cur + num_bytes >= len)
		return 0;

	if (num_bytes == 0)
		return 0;
	else if (num_bytes == 1) {
		return rpt[cur+1];
	}
	else if (num_bytes == 2) {
		return (rpt[cur+2] * 256 + rpt[cur+1]);
	}
	else if (num_bytes == 4) {
		return (rpt[cur+4] * 0x01000000 +
		        rpt[cur+3] * 0x00010000 +
		        rpt[cur+2] * 0x00000100 +
		        rpt[cur+1] * 0x00000001);
	}
	else
		return 0;
}

static int get_usage(uint8_t *report_descriptor, size_t size,
                     unsigned short *usage_page, unsigned short *usage)
{
	unsigned int i = 0;
	int size_code;
	int data_len, key_size;
	int usage_found = 0, usage_page_found = 0;

	while (i < size) {
		int key = report_descriptor[i];
		int key_cmd = key & 0xfc;

		if ((key & 0xf0) == 0xf0) {

			if (i+1 < size)
				data_len = report_descriptor[i+1];
			else
				data_len = 0; 
			key_size = 3;
		}
		else {

			size_code = key & 0x3;
			switch (size_code) {
			case 0:
			case 1:
			case 2:
				data_len = size_code;
				break;
			case 3:
				data_len = 4;
				break;
			default:

				data_len = 0;
				break;
			};
			key_size = 1;
		}

		if (key_cmd == 0x4) {
			*usage_page  = get_bytes(report_descriptor, size, data_len, i);
			usage_page_found = 1;

		}
		if (key_cmd == 0x8) {
			*usage = get_bytes(report_descriptor, size, data_len, i);
			usage_found = 1;

		}

		if (usage_page_found && usage_found)
			return 0; 

		i += data_len + key_size;
	}

	return -1; 
}
#endif 

#if defined(__FreeBSD__) && __FreeBSD__ < 10

static inline int libusb_get_string_descriptor(libusb_device_handle *dev,
	uint8_t descriptor_index, uint16_t lang_id,
	unsigned char *data, int length)
{
	return libusb_control_transfer(dev,
		LIBUSB_ENDPOINT_IN | 0x0, 
		LIBUSB_REQUEST_GET_DESCRIPTOR,
		(LIBUSB_DT_STRING << 8) | descriptor_index,
		lang_id, data, (uint16_t) length, 1000);
}

#endif

static uint16_t get_first_language(libusb_device_handle *dev)
{
	uint16_t buf[32];
	int len;

	len = libusb_get_string_descriptor(dev,
			0x0, 
			0x0, 
			(unsigned char*)buf,
			sizeof(buf));
	if (len < 4)
		return 0x0;

	return buf[1]; 
}

static int is_language_supported(libusb_device_handle *dev, uint16_t lang)
{
	uint16_t buf[32];
	int len;
	int i;

	len = libusb_get_string_descriptor(dev,
			0x0, 
			0x0, 
			(unsigned char*)buf,
			sizeof(buf));
	if (len < 4)
		return 0x0;

	len /= 2; 

	for (i = 1; i < len; i++) {
		if (buf[i] == lang)
			return 1;
	}

	return 0;
}

static wchar_t *get_usb_string(libusb_device_handle *dev, uint8_t idx)
{
	char buf[512];
	int len;
	wchar_t *str = NULL;

#ifndef __ANDROID__ 
	wchar_t wbuf[256];

	iconv_t ic;
	size_t inbytes;
	size_t outbytes;
	size_t res;
#ifdef __FreeBSD__
	const char *inptr;
#else
	char *inptr;
#endif
	char *outptr;
#endif

	uint16_t lang;
	lang = get_usb_code_for_current_locale();
	if (!is_language_supported(dev, lang))
		lang = get_first_language(dev);

	len = libusb_get_string_descriptor(dev,
			idx,
			lang,
			(unsigned char*)buf,
			sizeof(buf));
	if (len < 0)
		return NULL;

#ifdef __ANDROID__

	len -= 2;
	str = malloc((len / 2 + 1) * sizeof(wchar_t));
	int i;
	for (i = 0; i < len / 2; i++) {
		str[i] = buf[i * 2 + 2] | (buf[i * 2 + 3] << 8);
	}
	str[len / 2] = 0x00000000;

#else

	ic = iconv_open("WCHAR_T", "UTF-16LE");
	if (ic == (iconv_t)-1) {
		LOG("iconv_open() failed\n");
		return NULL;
	}

	inptr = buf+2;
	inbytes = len-2;
	outptr = (char*) wbuf;
	outbytes = sizeof(wbuf);
	res = iconv(ic, &inptr, &inbytes, &outptr, &outbytes);
	if (res == (size_t)-1) {
		LOG("iconv() failed\n");
		goto err;
	}

	wbuf[sizeof(wbuf)/sizeof(wbuf[0])-1] = 0x00000000;
	if (outbytes >= sizeof(wbuf[0]))
		*((wchar_t*)outptr) = 0x00000000;

	str = wcsdup(wbuf);

err:
	iconv_close(ic);

#endif

	return str;
}

static char *make_path(libusb_device *dev, int interface_number)
{
	char str[64];
	snprintf(str, sizeof(str), "%04x:%04x:%02x",
		libusb_get_bus_number(dev),
		libusb_get_device_address(dev),
		interface_number);
	str[sizeof(str)-1] = '\0';

	return strdup(str);
}

int HID_API_EXPORT hid_init(void)
{
	if (!usb_context) {
		const char *locale;

		if (libusb_init(&usb_context))
			return -1;

		locale = setlocale(LC_CTYPE, NULL);
		if (!locale)
			setlocale(LC_CTYPE, "");
	}

	return 0;
}

int HID_API_EXPORT hid_exit(void)
{
	if (usb_context) {
		libusb_exit(usb_context);
		usb_context = NULL;
	}

	return 0;
}

struct hid_device_info  HID_API_EXPORT *hid_enumerate(unsigned short vendor_id, unsigned short product_id)
{
	libusb_device **devs;
	libusb_device *dev;
	libusb_device_handle *handle;
	ssize_t num_devs;
	int i = 0;

	struct hid_device_info *root = NULL; 
	struct hid_device_info *cur_dev = NULL;

	if(hid_init() < 0)
		return NULL;

	num_devs = libusb_get_device_list(usb_context, &devs);
	if (num_devs < 0)
		return NULL;
	while ((dev = devs[i++]) != NULL) {
		struct libusb_device_descriptor desc;
		struct libusb_config_descriptor *conf_desc = NULL;
		int j, k;
		int interface_num = 0;

		int res = libusb_get_device_descriptor(dev, &desc);
		unsigned short dev_vid = desc.idVendor;
		unsigned short dev_pid = desc.idProduct;

		res = libusb_get_active_config_descriptor(dev, &conf_desc);
		if (res < 0)
			libusb_get_config_descriptor(dev, 0, &conf_desc);
		if (conf_desc) {
			for (j = 0; j < conf_desc->bNumInterfaces; j++) {
				const struct libusb_interface *intf = &conf_desc->interface[j];
				for (k = 0; k < intf->num_altsetting; k++) {
					const struct libusb_interface_descriptor *intf_desc;
					intf_desc = &intf->altsetting[k];
					if (intf_desc->bInterfaceClass == LIBUSB_CLASS_HID) {
						interface_num = intf_desc->bInterfaceNumber;

						if ((vendor_id == 0x0 || vendor_id == dev_vid) &&
						    (product_id == 0x0 || product_id == dev_pid)) {
							struct hid_device_info *tmp;

							tmp = calloc(1, sizeof(struct hid_device_info));
							if (cur_dev) {
								cur_dev->next = tmp;
							}
							else {
								root = tmp;
							}
							cur_dev = tmp;

							cur_dev->next = NULL;
							cur_dev->path = make_path(dev, interface_num);

							res = libusb_open(dev, &handle);

							if (res >= 0) {

								if (desc.iSerialNumber > 0)
									cur_dev->serial_number =
										get_usb_string(handle, desc.iSerialNumber);

								if (desc.iManufacturer > 0)
									cur_dev->manufacturer_string =
										get_usb_string(handle, desc.iManufacturer);
								if (desc.iProduct > 0)
									cur_dev->product_string =
										get_usb_string(handle, desc.iProduct);

#ifdef INVASIVE_GET_USAGE
{

								unsigned char data[256];
#ifdef DETACH_KERNEL_DRIVER
								int detached = 0;

								res = libusb_kernel_driver_active(handle, interface_num);
								if (res == 1) {
									res = libusb_detach_kernel_driver(handle, interface_num);
									if (res < 0)
										LOG("Couldn't detach kernel driver, even though a kernel driver was attached.");
									else
										detached = 1;
								}
#endif
								res = libusb_claim_interface(handle, interface_num);
								if (res >= 0) {

									res = libusb_control_transfer(handle, LIBUSB_ENDPOINT_IN|LIBUSB_RECIPIENT_INTERFACE, LIBUSB_REQUEST_GET_DESCRIPTOR, (LIBUSB_DT_REPORT << 8)|interface_num, 0, data, sizeof(data), 5000);
									if (res >= 0) {
										unsigned short page=0, usage=0;

										get_usage(data, res,  &page, &usage);
										cur_dev->usage_page = page;
										cur_dev->usage = usage;
									}
									else
										LOG("libusb_control_transfer() for getting the HID report failed with %d\n", res);

									res = libusb_release_interface(handle, interface_num);
									if (res < 0)
										LOG("Can't release the interface.\n");
								}
								else
									LOG("Can't claim interface %d\n", res);
#ifdef DETACH_KERNEL_DRIVER

								if (detached) {
									res = libusb_attach_kernel_driver(handle, interface_num);
									if (res < 0)
										LOG("Couldn't re-attach kernel driver.\n");
								}
#endif
}
#endif 

								libusb_close(handle);
							}

							cur_dev->vendor_id = dev_vid;
							cur_dev->product_id = dev_pid;

							cur_dev->release_number = desc.bcdDevice;

							cur_dev->interface_number = interface_num;
						}
					}
				} 
			} 
			libusb_free_config_descriptor(conf_desc);
		}
	}

	libusb_free_device_list(devs, 1);

	return root;
}

void  HID_API_EXPORT hid_free_enumeration(struct hid_device_info *devs)
{
	struct hid_device_info *d = devs;
	while (d) {
		struct hid_device_info *next = d->next;
		free(d->path);
		free(d->serial_number);
		free(d->manufacturer_string);
		free(d->product_string);
		free(d);
		d = next;
	}
}

hid_device * hid_open(unsigned short vendor_id, unsigned short product_id, const wchar_t *serial_number)
{
	struct hid_device_info *devs, *cur_dev;
	const char *path_to_open = NULL;
	hid_device *handle = NULL;

	devs = hid_enumerate(vendor_id, product_id);
	cur_dev = devs;
	while (cur_dev) {
		if (cur_dev->vendor_id == vendor_id &&
		    cur_dev->product_id == product_id) {
			if (serial_number) {
				if (cur_dev->serial_number &&
				    wcscmp(serial_number, cur_dev->serial_number) == 0) {
					path_to_open = cur_dev->path;
					break;
				}
			}
			else {
				path_to_open = cur_dev->path;
				break;
			}
		}
		cur_dev = cur_dev->next;
	}

	if (path_to_open) {

		handle = hid_open_path(path_to_open);
	}

	hid_free_enumeration(devs);

	return handle;
}

static void read_callback(struct libusb_transfer *transfer)
{
	hid_device *dev = transfer->user_data;
	int res;

	if (transfer->status == LIBUSB_TRANSFER_COMPLETED) {

		struct input_report *rpt = malloc(sizeof(*rpt));
		rpt->data = malloc(transfer->actual_length);
		memcpy(rpt->data, transfer->buffer, transfer->actual_length);
		rpt->len = transfer->actual_length;
		rpt->next = NULL;

		pthread_mutex_lock(&dev->mutex);

		if (dev->input_reports == NULL) {

			dev->input_reports = rpt;
			pthread_cond_signal(&dev->condition);
		}
		else {

			struct input_report *cur = dev->input_reports;
			int num_queued = 0;
			while (cur->next != NULL) {
				cur = cur->next;
				num_queued++;
			}
			cur->next = rpt;

			if (num_queued > 30) {
				return_data(dev, NULL, 0);
			}
		}
		pthread_mutex_unlock(&dev->mutex);
	}
	else if (transfer->status == LIBUSB_TRANSFER_CANCELLED) {
		dev->shutdown_thread = 1;
		dev->cancelled = 1;
		return;
	}
	else if (transfer->status == LIBUSB_TRANSFER_NO_DEVICE) {
		dev->shutdown_thread = 1;
		dev->cancelled = 1;
		return;
	}
	else if (transfer->status == LIBUSB_TRANSFER_TIMED_OUT) {

	}
	else {
		LOG("Unknown transfer code: %d\n", transfer->status);
	}

	res = libusb_submit_transfer(transfer);
	if (res != 0) {
		LOG("Unable to submit URB. libusb error code: %d\n", res);
		dev->shutdown_thread = 1;
		dev->cancelled = 1;
	}
}

static void *read_thread(void *param)
{
	hid_device *dev = param;
	unsigned char *buf;
	const size_t length = dev->input_ep_max_packet_size;

	buf = malloc(length);
	dev->transfer = libusb_alloc_transfer(0);
	libusb_fill_interrupt_transfer(dev->transfer,
		dev->device_handle,
		dev->input_endpoint,
		buf,
		length,
		read_callback,
		dev,
		5000);

	libusb_submit_transfer(dev->transfer);

	pthread_barrier_wait(&dev->barrier);

	while (!dev->shutdown_thread) {
		int res;
		res = libusb_handle_events(usb_context);
		if (res < 0) {

			LOG("read_thread(): libusb reports error # %d\n", res);

			if (res != LIBUSB_ERROR_BUSY &&
			    res != LIBUSB_ERROR_TIMEOUT &&
			    res != LIBUSB_ERROR_OVERFLOW &&
			    res != LIBUSB_ERROR_INTERRUPTED) {
				break;
			}
		}
	}

	libusb_cancel_transfer(dev->transfer);

	while (!dev->cancelled)
		libusb_handle_events_completed(usb_context, &dev->cancelled);

	pthread_mutex_lock(&dev->mutex);
	pthread_cond_broadcast(&dev->condition);
	pthread_mutex_unlock(&dev->mutex);

	return NULL;
}

hid_device * HID_API_EXPORT hid_open_path(const char *path)
{
	hid_device *dev = NULL;

	libusb_device **devs;
	libusb_device *usb_dev;
	int res;
	int d = 0;
	int good_open = 0;

	if(hid_init() < 0)
		return NULL;

	dev = new_hid_device();

	libusb_get_device_list(usb_context, &devs);
	while ((usb_dev = devs[d++]) != NULL) {
		struct libusb_device_descriptor desc;
		struct libusb_config_descriptor *conf_desc = NULL;
		int i,j,k;
		libusb_get_device_descriptor(usb_dev, &desc);

		if (libusb_get_active_config_descriptor(usb_dev, &conf_desc) < 0)
			continue;
		for (j = 0; j < conf_desc->bNumInterfaces; j++) {
			const struct libusb_interface *intf = &conf_desc->interface[j];
			for (k = 0; k < intf->num_altsetting; k++) {
				const struct libusb_interface_descriptor *intf_desc;
				intf_desc = &intf->altsetting[k];
				if (intf_desc->bInterfaceClass == LIBUSB_CLASS_HID) {
					char *dev_path = make_path(usb_dev, intf_desc->bInterfaceNumber);
					if (!strcmp(dev_path, path)) {

						res = libusb_open(usb_dev, &dev->device_handle);
						if (res < 0) {
							LOG("can't open device\n");
							free(dev_path);
							break;
						}
						good_open = 1;
#ifdef DETACH_KERNEL_DRIVER

						if (libusb_kernel_driver_active(dev->device_handle, intf_desc->bInterfaceNumber) == 1) {
							res = libusb_detach_kernel_driver(dev->device_handle, intf_desc->bInterfaceNumber);
							if (res < 0) {
								libusb_close(dev->device_handle);
								LOG("Unable to detach Kernel Driver\n");
								free(dev_path);
								good_open = 0;
								break;
							}
						}
#endif
						res = libusb_claim_interface(dev->device_handle, intf_desc->bInterfaceNumber);
						if (res < 0) {
							LOG("can't claim interface %d: %d\n", intf_desc->bInterfaceNumber, res);
							free(dev_path);
							libusb_close(dev->device_handle);
							good_open = 0;
							break;
						}

						dev->manufacturer_index = desc.iManufacturer;
						dev->product_index      = desc.iProduct;
						dev->serial_index       = desc.iSerialNumber;

						dev->interface = intf_desc->bInterfaceNumber;

						for (i = 0; i < intf_desc->bNumEndpoints; i++) {
							const struct libusb_endpoint_descriptor *ep
								= &intf_desc->endpoint[i];

							int is_interrupt =
								(ep->bmAttributes & LIBUSB_TRANSFER_TYPE_MASK)
							      == LIBUSB_TRANSFER_TYPE_INTERRUPT;
							int is_output =
								(ep->bEndpointAddress & LIBUSB_ENDPOINT_DIR_MASK)
							      == LIBUSB_ENDPOINT_OUT;
							int is_input =
								(ep->bEndpointAddress & LIBUSB_ENDPOINT_DIR_MASK)
							      == LIBUSB_ENDPOINT_IN;

							if (dev->input_endpoint == 0 &&
							    is_interrupt && is_input) {

								dev->input_endpoint = ep->bEndpointAddress;
								dev->input_ep_max_packet_size = ep->wMaxPacketSize;
							}
							if (dev->output_endpoint == 0 &&
							    is_interrupt && is_output) {

								dev->output_endpoint = ep->bEndpointAddress;
							}
						}

						pthread_create(&dev->thread, NULL, read_thread, dev);

						pthread_barrier_wait(&dev->barrier);

					}
					free(dev_path);
				}
			}
		}
		libusb_free_config_descriptor(conf_desc);

	}

	libusb_free_device_list(devs, 1);

	if (good_open) {
		return dev;
	}
	else {

		free_hid_device(dev);
		return NULL;
	}
}

int HID_API_EXPORT hid_write(hid_device *dev, const unsigned char *data, size_t length)
{
	int res;
	int report_number = data[0];
	int skipped_report_id = 0;

	if (report_number == 0x0) {
		data++;
		length--;
		skipped_report_id = 1;
	}

	if (dev->output_endpoint <= 0) {

		res = libusb_control_transfer(dev->device_handle,
			LIBUSB_REQUEST_TYPE_CLASS|LIBUSB_RECIPIENT_INTERFACE|LIBUSB_ENDPOINT_OUT,
			0x09,
			(2 << 8) | report_number,
			dev->interface,
			(unsigned char *)data, length,
			1000);

		if (res < 0)
			return -1;

		if (skipped_report_id)
			length++;

		return length;
	}
	else {

		int actual_length;
		res = libusb_interrupt_transfer(dev->device_handle,
			dev->output_endpoint,
			(unsigned char*)data,
			length,
			&actual_length, 1000);

		if (res < 0)
			return -1;

		if (skipped_report_id)
			actual_length++;

		return actual_length;
	}
}

static int return_data(hid_device *dev, unsigned char *data, size_t length)
{

	struct input_report *rpt = dev->input_reports;
	size_t len = (length < rpt->len)? length: rpt->len;
	if (len > 0)
		memcpy(data, rpt->data, len);
	dev->input_reports = rpt->next;
	free(rpt->data);
	free(rpt);
	return len;
}

static void cleanup_mutex(void *param)
{
	hid_device *dev = param;
	pthread_mutex_unlock(&dev->mutex);
}

int HID_API_EXPORT hid_read_timeout(hid_device *dev, unsigned char *data, size_t length, int milliseconds)
{
	int bytes_read = -1;

#if 0
	int transferred;
	int res = libusb_interrupt_transfer(dev->device_handle, dev->input_endpoint, data, length, &transferred, 5000);
	LOG("transferred: %d\n", transferred);
	return transferred;
#endif

	pthread_mutex_lock(&dev->mutex);
	pthread_cleanup_push(&cleanup_mutex, dev);

	if (dev->input_reports) {

		bytes_read = return_data(dev, data, length);
		goto ret;
	}

	if (dev->shutdown_thread) {

		bytes_read = -1;
		goto ret;
	}

	if (milliseconds == -1) {

		while (!dev->input_reports && !dev->shutdown_thread) {
			pthread_cond_wait(&dev->condition, &dev->mutex);
		}
		if (dev->input_reports) {
			bytes_read = return_data(dev, data, length);
		}
	}
	else if (milliseconds > 0) {

		int res;
		struct timespec ts;
		clock_gettime(CLOCK_REALTIME, &ts);
		ts.tv_sec += milliseconds / 1000;
		ts.tv_nsec += (milliseconds % 1000) * 1000000;
		if (ts.tv_nsec >= 1000000000L) {
			ts.tv_sec++;
			ts.tv_nsec -= 1000000000L;
		}

		while (!dev->input_reports && !dev->shutdown_thread) {
			res = pthread_cond_timedwait(&dev->condition, &dev->mutex, &ts);
			if (res == 0) {
				if (dev->input_reports) {
					bytes_read = return_data(dev, data, length);
					break;
				}

			}
			else if (res == ETIMEDOUT) {

				bytes_read = 0;
				break;
			}
			else {

				bytes_read = -1;
				break;
			}
		}
	}
	else {

		bytes_read = 0;
	}

ret:
	pthread_mutex_unlock(&dev->mutex);
	pthread_cleanup_pop(0);

	return bytes_read;
}

int HID_API_EXPORT hid_read(hid_device *dev, unsigned char *data, size_t length)
{
	return hid_read_timeout(dev, data, length, dev->blocking ? -1 : 0);
}

int HID_API_EXPORT hid_set_nonblocking(hid_device *dev, int nonblock)
{
	dev->blocking = !nonblock;

	return 0;
}

int HID_API_EXPORT hid_send_feature_report(hid_device *dev, const unsigned char *data, size_t length)
{
	int res = -1;
	int skipped_report_id = 0;
	int report_number = data[0];

	if (report_number == 0x0) {
		data++;
		length--;
		skipped_report_id = 1;
	}

	res = libusb_control_transfer(dev->device_handle,
		LIBUSB_REQUEST_TYPE_CLASS|LIBUSB_RECIPIENT_INTERFACE|LIBUSB_ENDPOINT_OUT,
		0x09,
		(3 << 8) | report_number,
		dev->interface,
		(unsigned char *)data, length,
		1000);

	if (res < 0)
		return -1;

	if (skipped_report_id)
		length++;

	return length;
}

int HID_API_EXPORT hid_get_feature_report(hid_device *dev, unsigned char *data, size_t length)
{
	int res = -1;
	int skipped_report_id = 0;
	int report_number = data[0];

	if (report_number == 0x0) {

		data++;
		length--;
		skipped_report_id = 1;
	}
	res = libusb_control_transfer(dev->device_handle,
		LIBUSB_REQUEST_TYPE_CLASS|LIBUSB_RECIPIENT_INTERFACE|LIBUSB_ENDPOINT_IN,
		0x01,
		(3 << 8) | report_number,
		dev->interface,
		(unsigned char *)data, length,
		1000);

	if (res < 0)
		return -1;

	if (skipped_report_id)
		res++;

	return res;
}

void HID_API_EXPORT hid_close(hid_device *dev)
{
	if (!dev)
		return;

	dev->shutdown_thread = 1;
	libusb_cancel_transfer(dev->transfer);

	pthread_join(dev->thread, NULL);

	free(dev->transfer->buffer);
	libusb_free_transfer(dev->transfer);

	libusb_release_interface(dev->device_handle, dev->interface);

	libusb_close(dev->device_handle);

	pthread_mutex_lock(&dev->mutex);
	while (dev->input_reports) {
		return_data(dev, NULL, 0);
	}
	pthread_mutex_unlock(&dev->mutex);

	free_hid_device(dev);
}

int HID_API_EXPORT_CALL hid_get_manufacturer_string(hid_device *dev, wchar_t *string, size_t maxlen)
{
	return hid_get_indexed_string(dev, dev->manufacturer_index, string, maxlen);
}

int HID_API_EXPORT_CALL hid_get_product_string(hid_device *dev, wchar_t *string, size_t maxlen)
{
	return hid_get_indexed_string(dev, dev->product_index, string, maxlen);
}

int HID_API_EXPORT_CALL hid_get_serial_number_string(hid_device *dev, wchar_t *string, size_t maxlen)
{
	return hid_get_indexed_string(dev, dev->serial_index, string, maxlen);
}

int HID_API_EXPORT_CALL hid_get_indexed_string(hid_device *dev, int string_index, wchar_t *string, size_t maxlen)
{
	wchar_t *str;

	str = get_usb_string(dev->device_handle, string_index);
	if (str) {
		wcsncpy(string, str, maxlen);
		string[maxlen-1] = L'\0';
		free(str);
		return 0;
	}
	else
		return -1;
}

HID_API_EXPORT const wchar_t * HID_API_CALL  hid_error(hid_device *dev)
{
	return NULL;
}

struct lang_map_entry {
	const char *name;
	const char *string_code;
	uint16_t usb_code;
};

#define LANG(name,code,usb_code) { name, code, usb_code }
static struct lang_map_entry lang_map[] = {
	LANG("Afrikaans", "af", 0x0436),
	LANG("Albanian", "sq", 0x041C),
	LANG("Arabic - United Arab Emirates", "ar_ae", 0x3801),
	LANG("Arabic - Bahrain", "ar_bh", 0x3C01),
	LANG("Arabic - Algeria", "ar_dz", 0x1401),
	LANG("Arabic - Egypt", "ar_eg", 0x0C01),
	LANG("Arabic - Iraq", "ar_iq", 0x0801),
	LANG("Arabic - Jordan", "ar_jo", 0x2C01),
	LANG("Arabic - Kuwait", "ar_kw", 0x3401),
	LANG("Arabic - Lebanon", "ar_lb", 0x3001),
	LANG("Arabic - Libya", "ar_ly", 0x1001),
	LANG("Arabic - Morocco", "ar_ma", 0x1801),
	LANG("Arabic - Oman", "ar_om", 0x2001),
	LANG("Arabic - Qatar", "ar_qa", 0x4001),
	LANG("Arabic - Saudi Arabia", "ar_sa", 0x0401),
	LANG("Arabic - Syria", "ar_sy", 0x2801),
	LANG("Arabic - Tunisia", "ar_tn", 0x1C01),
	LANG("Arabic - Yemen", "ar_ye", 0x2401),
	LANG("Armenian", "hy", 0x042B),
	LANG("Azeri - Latin", "az_az", 0x042C),
	LANG("Azeri - Cyrillic", "az_az", 0x082C),
	LANG("Basque", "eu", 0x042D),
	LANG("Belarusian", "be", 0x0423),
	LANG("Bulgarian", "bg", 0x0402),
	LANG("Catalan", "ca", 0x0403),
	LANG("Chinese - China", "zh_cn", 0x0804),
	LANG("Chinese - Hong Kong SAR", "zh_hk", 0x0C04),
	LANG("Chinese - Macau SAR", "zh_mo", 0x1404),
	LANG("Chinese - Singapore", "zh_sg", 0x1004),
	LANG("Chinese - Taiwan", "zh_tw", 0x0404),
	LANG("Croatian", "hr", 0x041A),
	LANG("Czech", "cs", 0x0405),
	LANG("Danish", "da", 0x0406),
	LANG("Dutch - Netherlands", "nl_nl", 0x0413),
	LANG("Dutch - Belgium", "nl_be", 0x0813),
	LANG("English - Australia", "en_au", 0x0C09),
	LANG("English - Belize", "en_bz", 0x2809),
	LANG("English - Canada", "en_ca", 0x1009),
	LANG("English - Caribbean", "en_cb", 0x2409),
	LANG("English - Ireland", "en_ie", 0x1809),
	LANG("English - Jamaica", "en_jm", 0x2009),
	LANG("English - New Zealand", "en_nz", 0x1409),
	LANG("English - Phillippines", "en_ph", 0x3409),
	LANG("English - Southern Africa", "en_za", 0x1C09),
	LANG("English - Trinidad", "en_tt", 0x2C09),
	LANG("English - Great Britain", "en_gb", 0x0809),
	LANG("English - United States", "en_us", 0x0409),
	LANG("Estonian", "et", 0x0425),
	LANG("Farsi", "fa", 0x0429),
	LANG("Finnish", "fi", 0x040B),
	LANG("Faroese", "fo", 0x0438),
	LANG("French - France", "fr_fr", 0x040C),
	LANG("French - Belgium", "fr_be", 0x080C),
	LANG("French - Canada", "fr_ca", 0x0C0C),
	LANG("French - Luxembourg", "fr_lu", 0x140C),
	LANG("French - Switzerland", "fr_ch", 0x100C),
	LANG("Gaelic - Ireland", "gd_ie", 0x083C),
	LANG("Gaelic - Scotland", "gd", 0x043C),
	LANG("German - Germany", "de_de", 0x0407),
	LANG("German - Austria", "de_at", 0x0C07),
	LANG("German - Liechtenstein", "de_li", 0x1407),
	LANG("German - Luxembourg", "de_lu", 0x1007),
	LANG("German - Switzerland", "de_ch", 0x0807),
	LANG("Greek", "el", 0x0408),
	LANG("Hebrew", "he", 0x040D),
	LANG("Hindi", "hi", 0x0439),
	LANG("Hungarian", "hu", 0x040E),
	LANG("Icelandic", "is", 0x040F),
	LANG("Indonesian", "id", 0x0421),
	LANG("Italian - Italy", "it_it", 0x0410),
	LANG("Italian - Switzerland", "it_ch", 0x0810),
	LANG("Japanese", "ja", 0x0411),
	LANG("Korean", "ko", 0x0412),
	LANG("Latvian", "lv", 0x0426),
	LANG("Lithuanian", "lt", 0x0427),
	LANG("F.Y.R.O. Macedonia", "mk", 0x042F),
	LANG("Malay - Malaysia", "ms_my", 0x043E),
	LANG("Malay – Brunei", "ms_bn", 0x083E),
	LANG("Maltese", "mt", 0x043A),
	LANG("Marathi", "mr", 0x044E),
	LANG("Norwegian - Bokml", "no_no", 0x0414),
	LANG("Norwegian - Nynorsk", "no_no", 0x0814),
	LANG("Polish", "pl", 0x0415),
	LANG("Portuguese - Portugal", "pt_pt", 0x0816),
	LANG("Portuguese - Brazil", "pt_br", 0x0416),
	LANG("Raeto-Romance", "rm", 0x0417),
	LANG("Romanian - Romania", "ro", 0x0418),
	LANG("Romanian - Republic of Moldova", "ro_mo", 0x0818),
	LANG("Russian", "ru", 0x0419),
	LANG("Russian - Republic of Moldova", "ru_mo", 0x0819),
	LANG("Sanskrit", "sa", 0x044F),
	LANG("Serbian - Cyrillic", "sr_sp", 0x0C1A),
	LANG("Serbian - Latin", "sr_sp", 0x081A),
	LANG("Setsuana", "tn", 0x0432),
	LANG("Slovenian", "sl", 0x0424),
	LANG("Slovak", "sk", 0x041B),
	LANG("Sorbian", "sb", 0x042E),
	LANG("Spanish - Spain (Traditional)", "es_es", 0x040A),
	LANG("Spanish - Argentina", "es_ar", 0x2C0A),
	LANG("Spanish - Bolivia", "es_bo", 0x400A),
	LANG("Spanish - Chile", "es_cl", 0x340A),
	LANG("Spanish - Colombia", "es_co", 0x240A),
	LANG("Spanish - Costa Rica", "es_cr", 0x140A),
	LANG("Spanish - Dominican Republic", "es_do", 0x1C0A),
	LANG("Spanish - Ecuador", "es_ec", 0x300A),
	LANG("Spanish - Guatemala", "es_gt", 0x100A),
	LANG("Spanish - Honduras", "es_hn", 0x480A),
	LANG("Spanish - Mexico", "es_mx", 0x080A),
	LANG("Spanish - Nicaragua", "es_ni", 0x4C0A),
	LANG("Spanish - Panama", "es_pa", 0x180A),
	LANG("Spanish - Peru", "es_pe", 0x280A),
	LANG("Spanish - Puerto Rico", "es_pr", 0x500A),
	LANG("Spanish - Paraguay", "es_py", 0x3C0A),
	LANG("Spanish - El Salvador", "es_sv", 0x440A),
	LANG("Spanish - Uruguay", "es_uy", 0x380A),
	LANG("Spanish - Venezuela", "es_ve", 0x200A),
	LANG("Southern Sotho", "st", 0x0430),
	LANG("Swahili", "sw", 0x0441),
	LANG("Swedish - Sweden", "sv_se", 0x041D),
	LANG("Swedish - Finland", "sv_fi", 0x081D),
	LANG("Tamil", "ta", 0x0449),
	LANG("Tatar", "tt", 0X0444),
	LANG("Thai", "th", 0x041E),
	LANG("Turkish", "tr", 0x041F),
	LANG("Tsonga", "ts", 0x0431),
	LANG("Ukrainian", "uk", 0x0422),
	LANG("Urdu", "ur", 0x0420),
	LANG("Uzbek - Cyrillic", "uz_uz", 0x0843),
	LANG("Uzbek – Latin", "uz_uz", 0x0443),
	LANG("Vietnamese", "vi", 0x042A),
	LANG("Xhosa", "xh", 0x0434),
	LANG("Yiddish", "yi", 0x043D),
	LANG("Zulu", "zu", 0x0435),
	LANG(NULL, NULL, 0x0),
};

uint16_t get_usb_code_for_current_locale(void)
{
	char *locale;
	char search_string[64];
	char *ptr;
	struct lang_map_entry *lang;

	locale = setlocale(0, NULL);
	if (!locale)
		return 0x0;

	strncpy(search_string, locale, sizeof(search_string));
	search_string[sizeof(search_string)-1] = '\0';

	ptr = search_string;
	while (*ptr) {
		*ptr = tolower(*ptr);
		if (*ptr == '.') {
			*ptr = '\0';
			break;
		}
		ptr++;
	}

	lang = lang_map;
	while (lang->string_code) {
		if (!strcmp(lang->string_code, search_string)) {
			return lang->usb_code;
		}
		lang++;
	}

	ptr = search_string;
	while (*ptr) {
		*ptr = tolower(*ptr);
		if (*ptr == '_') {
			*ptr = '\0';
			break;
		}
		ptr++;
	}

#if 0 

	lang = lang_map;
	while (lang->string_code) {
		if (!strcmp(lang->string_code, search_string)) {
			return lang->usb_code;
		}
		lang++;
	}
#endif

	return 0x0;
}

#ifdef __cplusplus
}
#endif
