
#include <config.h>

#include <inttypes.h>
#include <process.h>
#include <stdio.h>

#include "libusbi.h"
#include "windows_common.h"
#include "windows_nt_common.h"

static uint64_t hires_ticks_to_ps;
static uint64_t hires_frequency;

#define TIMER_REQUEST_RETRY_MS	100
#define WM_TIMER_REQUEST	(WM_USER + 1)
#define WM_TIMER_EXIT		(WM_USER + 2)

struct timer_request {
	struct timespec *tp;
	HANDLE event;
};

static HANDLE timer_thread = NULL;
static DWORD timer_thread_id = 0;

DLL_DECLARE_HANDLE(User32);
DLL_DECLARE_FUNC_PREFIXED(WINAPI, BOOL, p, GetMessageA, (LPMSG, HWND, UINT, UINT));
DLL_DECLARE_FUNC_PREFIXED(WINAPI, BOOL, p, PeekMessageA, (LPMSG, HWND, UINT, UINT, UINT));
DLL_DECLARE_FUNC_PREFIXED(WINAPI, BOOL, p, PostThreadMessageA, (DWORD, UINT, WPARAM, LPARAM));

static unsigned __stdcall windows_clock_gettime_threaded(void *param);

#if defined(ENABLE_LOGGING)
const char *windows_error_str(DWORD error_code)
{
	static char err_string[ERR_BUFFER_SIZE];

	DWORD size;
	int len;

	if (error_code == 0)
		error_code = GetLastError();

	len = sprintf(err_string, "[%u] ", (unsigned int)error_code);

	switch (error_code & 0xE0000000) {
	case 0:
		error_code = HRESULT_FROM_WIN32(error_code); 
		break;
	case 0xE0000000:
		error_code = 0x80000000 | (FACILITY_SETUPAPI << 16) | (error_code & 0x0000FFFF);
		break;
	default:
		break;
	}

	size = FormatMessageA(FORMAT_MESSAGE_FROM_SYSTEM|FORMAT_MESSAGE_IGNORE_INSERTS,
			NULL, error_code, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT),
			&err_string[len], ERR_BUFFER_SIZE - len, NULL);
	if (size == 0) {
		DWORD format_error = GetLastError();
		if (format_error)
			snprintf(err_string, ERR_BUFFER_SIZE,
				"Windows error code %u (FormatMessage error code %u)",
				(unsigned int)error_code, (unsigned int)format_error);
		else
			snprintf(err_string, ERR_BUFFER_SIZE, "Unknown error code %u", (unsigned int)error_code);
	} else {

		size_t pos = len + size - 2;
		if (err_string[pos] == '\r')
			err_string[pos] = '\0';
	}

	return err_string;
}
#endif

#define HTAB_SIZE 1021UL	

typedef struct htab_entry {
	unsigned long used;
	char *str;
} htab_entry;

static htab_entry *htab_table = NULL;
static usbi_mutex_t htab_mutex = NULL;
static unsigned long htab_filled;

static bool htab_create(struct libusb_context *ctx)
{
	if (htab_table != NULL) {
		usbi_err(ctx, "hash table already allocated");
		return true;
	}

	usbi_mutex_init(&htab_mutex);

	usbi_dbg("using %lu entries hash table", HTAB_SIZE);
	htab_filled = 0;

	htab_table = calloc(HTAB_SIZE + 1, sizeof(htab_entry));
	if (htab_table == NULL) {
		usbi_err(ctx, "could not allocate space for hash table");
		return false;
	}

	return true;
}

static void htab_destroy(void)
{
	unsigned long i;

	if (htab_table == NULL)
		return;

	for (i = 0; i < HTAB_SIZE; i++)
		free(htab_table[i].str);

	safe_free(htab_table);

	usbi_mutex_destroy(&htab_mutex);
}

unsigned long htab_hash(const char *str)
{
	unsigned long hval, hval2;
	unsigned long idx;
	unsigned long r = 5381;
	int c;
	const char *sz = str;

	if (str == NULL)
		return 0;

	while ((c = *sz++) != 0)
		r = ((r << 5) + r) + c;
	if (r == 0)
		++r;

	hval = r % HTAB_SIZE;
	if (hval == 0)
		++hval;

	idx = hval;

	usbi_mutex_lock(&htab_mutex);

	if (htab_table[idx].used) {
		if ((htab_table[idx].used == hval) && (strcmp(str, htab_table[idx].str) == 0))
			goto out_unlock; 

		usbi_dbg("hash collision ('%s' vs '%s')", str, htab_table[idx].str);

		hval2 = 1 + hval % (HTAB_SIZE - 2);

		do {

			if (idx <= hval2)
				idx = HTAB_SIZE + idx - hval2;
			else
				idx -= hval2;

			if (idx == hval)
				break;

			if ((htab_table[idx].used == hval) && (strcmp(str, htab_table[idx].str) == 0))
				goto out_unlock;
		} while (htab_table[idx].used);
	}

	if (htab_filled >= HTAB_SIZE) {
		usbi_err(NULL, "hash table is full (%lu entries)", HTAB_SIZE);
		idx = 0;
		goto out_unlock;
	}

	htab_table[idx].str = _strdup(str);
	if (htab_table[idx].str == NULL) {
		usbi_err(NULL, "could not duplicate string for hash table");
		idx = 0;
		goto out_unlock;
	}

	htab_table[idx].used = hval;
	++htab_filled;

out_unlock:
	usbi_mutex_unlock(&htab_mutex);

	return idx;
}

static int windows_init_dlls(void)
{
	DLL_GET_HANDLE(User32);
	DLL_LOAD_FUNC_PREFIXED(User32, p, GetMessageA, TRUE);
	DLL_LOAD_FUNC_PREFIXED(User32, p, PeekMessageA, TRUE);
	DLL_LOAD_FUNC_PREFIXED(User32, p, PostThreadMessageA, TRUE);

	return LIBUSB_SUCCESS;
}

static void windows_exit_dlls(void)
{
	DLL_FREE_HANDLE(User32);
}

static bool windows_init_clock(struct libusb_context *ctx)
{
	DWORD_PTR affinity, dummy;
	HANDLE event = NULL;
	LARGE_INTEGER li_frequency;
	int i;

	if (QueryPerformanceFrequency(&li_frequency)) {

		if (windows_init_dlls() != LIBUSB_SUCCESS) {
			usbi_err(ctx, "could not resolve DLL functions");
			return false;
		}

		hires_frequency = li_frequency.QuadPart;
		hires_ticks_to_ps = UINT64_C(1000000000000) / hires_frequency;
		usbi_dbg("hires timer available (Frequency: %"PRIu64" Hz)", hires_frequency);

		if (!GetProcessAffinityMask(GetCurrentProcess(), &affinity, &dummy) || (affinity == 0)) {
			usbi_err(ctx, "could not get process affinity: %s", windows_error_str(0));
			return false;
		}

		for (i = 0; !(affinity & (DWORD_PTR)(1 << i)); i++);
		affinity = (DWORD_PTR)(1 << i);

		usbi_dbg("timer thread will run on core #%d", i);

		event = CreateEvent(NULL, FALSE, FALSE, NULL);
		if (event == NULL) {
			usbi_err(ctx, "could not create event: %s", windows_error_str(0));
			return false;
		}

		timer_thread = (HANDLE)_beginthreadex(NULL, 0, windows_clock_gettime_threaded, (void *)event,
				0, (unsigned int *)&timer_thread_id);
		if (timer_thread == NULL) {
			usbi_err(ctx, "unable to create timer thread - aborting");
			CloseHandle(event);
			return false;
		}

		if (!SetThreadAffinityMask(timer_thread, affinity))
			usbi_warn(ctx, "unable to set timer thread affinity, timer discrepancies may arise");

		if (WaitForSingleObject(event, INFINITE) != WAIT_OBJECT_0) {
			usbi_err(ctx, "failed to wait for timer thread to become ready - aborting");
			CloseHandle(event);
			return false;
		}

		CloseHandle(event);
	} else {
		usbi_dbg("no hires timer available on this platform");
		hires_frequency = 0;
		hires_ticks_to_ps = UINT64_C(0);
	}

	return true;
}

void windows_destroy_clock(void)
{
	if (timer_thread) {

		if (!pPostThreadMessageA(timer_thread_id, WM_TIMER_EXIT, 0, 0)
				|| (WaitForSingleObject(timer_thread, INFINITE) != WAIT_OBJECT_0)) {
			usbi_dbg("could not wait for timer thread to quit");
			TerminateThread(timer_thread, 1);

		}
		CloseHandle(timer_thread);
		timer_thread = NULL;
		timer_thread_id = 0;
	}
}

static unsigned __stdcall windows_clock_gettime_threaded(void *param)
{
	struct timer_request *request;
	LARGE_INTEGER hires_counter;
	MSG msg;

	pPeekMessageA(&msg, NULL, WM_USER, WM_USER, PM_NOREMOVE);

	if (!SetEvent((HANDLE)param))
		usbi_dbg("SetEvent failed for timer init event: %s", windows_error_str(0));
	param = NULL;

	while (1) {
		if (pGetMessageA(&msg, NULL, WM_TIMER_REQUEST, WM_TIMER_EXIT) == -1) {
			usbi_err(NULL, "GetMessage failed for timer thread: %s", windows_error_str(0));
			return 1;
		}

		switch (msg.message) {
		case WM_TIMER_REQUEST:

			request = (struct timer_request *)msg.lParam;
			QueryPerformanceCounter(&hires_counter);
			request->tp->tv_sec = (long)(hires_counter.QuadPart / hires_frequency);
			request->tp->tv_nsec = (long)(((hires_counter.QuadPart % hires_frequency) / 1000) * hires_ticks_to_ps);
			if (!SetEvent(request->event))
				usbi_err(NULL, "SetEvent failed for timer request: %s", windows_error_str(0));
			break;
		case WM_TIMER_EXIT:
			usbi_dbg("timer thread quitting");
			return 0;
		}
	}
}

int windows_clock_gettime(int clk_id, struct timespec *tp)
{
	struct timer_request request;
#if !defined(_MSC_VER) || (_MSC_VER < 1900)
	FILETIME filetime;
	ULARGE_INTEGER rtime;
#endif
	DWORD r;

	switch (clk_id) {
	case USBI_CLOCK_MONOTONIC:
		if (timer_thread) {
			request.tp = tp;
			request.event = CreateEvent(NULL, FALSE, FALSE, NULL);
			if (request.event == NULL)
				return LIBUSB_ERROR_NO_MEM;

			if (!pPostThreadMessageA(timer_thread_id, WM_TIMER_REQUEST, 0, (LPARAM)&request)) {
				usbi_err(NULL, "PostThreadMessage failed for timer thread: %s", windows_error_str(0));
				CloseHandle(request.event);
				return LIBUSB_ERROR_OTHER;
			}

			do {
				r = WaitForSingleObject(request.event, TIMER_REQUEST_RETRY_MS);
				if (r == WAIT_TIMEOUT)
					usbi_dbg("could not obtain a timer value within reasonable timeframe - too much load?");
				else if (r == WAIT_FAILED)
					usbi_err(NULL, "WaitForSingleObject failed: %s", windows_error_str(0));
			} while (r == WAIT_TIMEOUT);
			CloseHandle(request.event);

			if (r == WAIT_OBJECT_0)
				return LIBUSB_SUCCESS;
			else
				return LIBUSB_ERROR_OTHER;
		}

	case USBI_CLOCK_REALTIME:
#if defined(_MSC_VER) && (_MSC_VER >= 1900)
		timespec_get(tp, TIME_UTC);
#else

		GetSystemTimeAsFileTime(&filetime);
		rtime.LowPart = filetime.dwLowDateTime;
		rtime.HighPart = filetime.dwHighDateTime;
		rtime.QuadPart -= EPOCH_TIME;
		tp->tv_sec = (long)(rtime.QuadPart / 10000000);
		tp->tv_nsec = (long)((rtime.QuadPart % 10000000) * 100);
#endif
		return LIBUSB_SUCCESS;
	default:
		return LIBUSB_ERROR_INVALID_PARAM;
	}
}

static void windows_transfer_callback(struct usbi_transfer *itransfer, uint32_t io_result, uint32_t io_size)
{
	int status, istatus;

	usbi_dbg("handling I/O completion with errcode %u, size %u", io_result, io_size);

	switch (io_result) {
	case NO_ERROR:
		status = windows_copy_transfer_data(itransfer, io_size);
		break;
	case ERROR_GEN_FAILURE:
		usbi_dbg("detected endpoint stall");
		status = LIBUSB_TRANSFER_STALL;
		break;
	case ERROR_SEM_TIMEOUT:
		usbi_dbg("detected semaphore timeout");
		status = LIBUSB_TRANSFER_TIMED_OUT;
		break;
	case ERROR_OPERATION_ABORTED:
		istatus = windows_copy_transfer_data(itransfer, io_size);
		if (istatus != LIBUSB_TRANSFER_COMPLETED)
			usbi_dbg("Failed to copy partial data in aborted operation: %d", istatus);

		usbi_dbg("detected operation aborted");
		status = LIBUSB_TRANSFER_CANCELLED;
		break;
	default:
		usbi_err(ITRANSFER_CTX(itransfer), "detected I/O error %u: %s", io_result, windows_error_str(io_result));
		status = LIBUSB_TRANSFER_ERROR;
		break;
	}
	windows_clear_transfer_priv(itransfer);	
	if (status == LIBUSB_TRANSFER_CANCELLED)
		usbi_handle_transfer_cancellation(itransfer);
	else
		usbi_handle_transfer_completion(itransfer, (enum libusb_transfer_status)status);
}

void windows_handle_callback(struct usbi_transfer *itransfer, uint32_t io_result, uint32_t io_size)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);

	switch (transfer->type) {
	case LIBUSB_TRANSFER_TYPE_CONTROL:
	case LIBUSB_TRANSFER_TYPE_BULK:
	case LIBUSB_TRANSFER_TYPE_INTERRUPT:
	case LIBUSB_TRANSFER_TYPE_ISOCHRONOUS:
		windows_transfer_callback(itransfer, io_result, io_size);
		break;
	case LIBUSB_TRANSFER_TYPE_BULK_STREAM:
		usbi_warn(ITRANSFER_CTX(itransfer), "bulk stream transfers are not yet supported on this platform");
		break;
	default:
		usbi_err(ITRANSFER_CTX(itransfer), "unknown endpoint type %d", transfer->type);
	}
}

int windows_handle_events(struct libusb_context *ctx, struct pollfd *fds, POLL_NFDS_TYPE nfds, int num_ready)
{
	POLL_NFDS_TYPE i;
	bool found = false;
	struct usbi_transfer *transfer;
	struct winfd *pollable_fd = NULL;
	DWORD io_size, io_result;
	int r = LIBUSB_SUCCESS;

	usbi_mutex_lock(&ctx->open_devs_lock);
	for (i = 0; i < nfds && num_ready > 0; i++) {

		usbi_dbg("checking fd %d with revents = %04x", fds[i].fd, fds[i].revents);

		if (!fds[i].revents)
			continue;

		num_ready--;

		usbi_mutex_lock(&ctx->flying_transfers_lock);
		found = false;
		list_for_each_entry(transfer, &ctx->flying_transfers, list, struct usbi_transfer) {
			pollable_fd = windows_get_fd(transfer);
			if (pollable_fd->fd == fds[i].fd) {
				found = true;
				break;
			}
		}
		usbi_mutex_unlock(&ctx->flying_transfers_lock);

		if (found) {
			windows_get_overlapped_result(transfer, pollable_fd, &io_result, &io_size);

			usbi_remove_pollfd(ctx, pollable_fd->fd);

			windows_handle_callback(transfer, io_result, io_size);
		} else {
			usbi_err(ctx, "could not find a matching transfer for fd %d", fds[i]);
			r = LIBUSB_ERROR_NOT_FOUND;
			break;
		}
	}
	usbi_mutex_unlock(&ctx->open_devs_lock);

	return r;
}

int windows_common_init(struct libusb_context *ctx)
{
	if (!windows_init_clock(ctx))
		goto error_roll_back;

	if (!htab_create(ctx))
		goto error_roll_back;

	return LIBUSB_SUCCESS;

error_roll_back:
	windows_common_exit();
	return LIBUSB_ERROR_NO_MEM;
}

void windows_common_exit(void)
{
	htab_destroy();
	windows_destroy_clock();
	windows_exit_dlls();
}
