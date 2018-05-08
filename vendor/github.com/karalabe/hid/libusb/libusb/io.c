
#include <config.h>

#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#ifdef HAVE_SYS_TIME_H
#include <sys/time.h>
#endif
#ifdef USBI_TIMERFD_AVAILABLE
#include <sys/timerfd.h>
#endif

#include "libusbi.h"
#include "hotplug.h"

int usbi_io_init(struct libusb_context *ctx)
{
	int r;

	usbi_mutex_init(&ctx->flying_transfers_lock);
	usbi_mutex_init(&ctx->events_lock);
	usbi_mutex_init(&ctx->event_waiters_lock);
	usbi_cond_init(&ctx->event_waiters_cond);
	usbi_mutex_init(&ctx->event_data_lock);
	usbi_tls_key_create(&ctx->event_handling_key);
	list_init(&ctx->flying_transfers);
	list_init(&ctx->ipollfds);
	list_init(&ctx->hotplug_msgs);
	list_init(&ctx->completed_transfers);

	r = usbi_pipe(ctx->event_pipe);
	if (r < 0) {
		r = LIBUSB_ERROR_OTHER;
		goto err;
	}

	r = usbi_add_pollfd(ctx, ctx->event_pipe[0], POLLIN);
	if (r < 0)
		goto err_close_pipe;

#ifdef USBI_TIMERFD_AVAILABLE
	ctx->timerfd = timerfd_create(usbi_backend->get_timerfd_clockid(),
		TFD_NONBLOCK);
	if (ctx->timerfd >= 0) {
		usbi_dbg("using timerfd for timeouts");
		r = usbi_add_pollfd(ctx, ctx->timerfd, POLLIN);
		if (r < 0)
			goto err_close_timerfd;
	} else {
		usbi_dbg("timerfd not available (code %d error %d)", ctx->timerfd, errno);
		ctx->timerfd = -1;
	}
#endif

	return 0;

#ifdef USBI_TIMERFD_AVAILABLE
err_close_timerfd:
	close(ctx->timerfd);
	usbi_remove_pollfd(ctx, ctx->event_pipe[0]);
#endif
err_close_pipe:
	usbi_close(ctx->event_pipe[0]);
	usbi_close(ctx->event_pipe[1]);
err:
	usbi_mutex_destroy(&ctx->flying_transfers_lock);
	usbi_mutex_destroy(&ctx->events_lock);
	usbi_mutex_destroy(&ctx->event_waiters_lock);
	usbi_cond_destroy(&ctx->event_waiters_cond);
	usbi_mutex_destroy(&ctx->event_data_lock);
	usbi_tls_key_delete(ctx->event_handling_key);
	return r;
}

void usbi_io_exit(struct libusb_context *ctx)
{
	usbi_remove_pollfd(ctx, ctx->event_pipe[0]);
	usbi_close(ctx->event_pipe[0]);
	usbi_close(ctx->event_pipe[1]);
#ifdef USBI_TIMERFD_AVAILABLE
	if (usbi_using_timerfd(ctx)) {
		usbi_remove_pollfd(ctx, ctx->timerfd);
		close(ctx->timerfd);
	}
#endif
	usbi_mutex_destroy(&ctx->flying_transfers_lock);
	usbi_mutex_destroy(&ctx->events_lock);
	usbi_mutex_destroy(&ctx->event_waiters_lock);
	usbi_cond_destroy(&ctx->event_waiters_cond);
	usbi_mutex_destroy(&ctx->event_data_lock);
	usbi_tls_key_delete(ctx->event_handling_key);
	if (ctx->pollfds)
		free(ctx->pollfds);
}

static int calculate_timeout(struct usbi_transfer *transfer)
{
	int r;
	struct timespec current_time;
	unsigned int timeout =
		USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer)->timeout;

	if (!timeout)
		return 0;

	r = usbi_backend->clock_gettime(USBI_CLOCK_MONOTONIC, &current_time);
	if (r < 0) {
		usbi_err(ITRANSFER_CTX(transfer),
			"failed to read monotonic clock, errno=%d", errno);
		return r;
	}

	current_time.tv_sec += timeout / 1000;
	current_time.tv_nsec += (timeout % 1000) * 1000000;

	while (current_time.tv_nsec >= 1000000000) {
		current_time.tv_nsec -= 1000000000;
		current_time.tv_sec++;
	}

	TIMESPEC_TO_TIMEVAL(&transfer->timeout, &current_time);
	return 0;
}

DEFAULT_VISIBILITY
struct libusb_transfer * LIBUSB_CALL libusb_alloc_transfer(
	int iso_packets)
{
	struct libusb_transfer *transfer;
	size_t os_alloc_size = usbi_backend->transfer_priv_size;
	size_t alloc_size = sizeof(struct usbi_transfer)
		+ sizeof(struct libusb_transfer)
		+ (sizeof(struct libusb_iso_packet_descriptor) * iso_packets)
		+ os_alloc_size;
	struct usbi_transfer *itransfer = calloc(1, alloc_size);
	if (!itransfer)
		return NULL;

	itransfer->num_iso_packets = iso_packets;
	usbi_mutex_init(&itransfer->lock);
	transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	usbi_dbg("transfer %p", transfer);
	return transfer;
}

void API_EXPORTED libusb_free_transfer(struct libusb_transfer *transfer)
{
	struct usbi_transfer *itransfer;
	if (!transfer)
		return;

	usbi_dbg("transfer %p", transfer);
	if (transfer->flags & LIBUSB_TRANSFER_FREE_BUFFER && transfer->buffer)
		free(transfer->buffer);

	itransfer = LIBUSB_TRANSFER_TO_USBI_TRANSFER(transfer);
	usbi_mutex_destroy(&itransfer->lock);
	free(itransfer);
}

#ifdef USBI_TIMERFD_AVAILABLE
static int disarm_timerfd(struct libusb_context *ctx)
{
	const struct itimerspec disarm_timer = { { 0, 0 }, { 0, 0 } };
	int r;

	usbi_dbg("");
	r = timerfd_settime(ctx->timerfd, 0, &disarm_timer, NULL);
	if (r < 0)
		return LIBUSB_ERROR_OTHER;
	else
		return 0;
}

static int arm_timerfd_for_next_timeout(struct libusb_context *ctx)
{
	struct usbi_transfer *transfer;

	list_for_each_entry(transfer, &ctx->flying_transfers, list, struct usbi_transfer) {
		struct timeval *cur_tv = &transfer->timeout;

		if (!timerisset(cur_tv))
			goto disarm;

		if (!(transfer->timeout_flags & (USBI_TRANSFER_TIMEOUT_HANDLED | USBI_TRANSFER_OS_HANDLES_TIMEOUT))) {
			int r;
			const struct itimerspec it = { {0, 0},
				{ cur_tv->tv_sec, cur_tv->tv_usec * 1000 } };
			usbi_dbg("next timeout originally %dms", USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer)->timeout);
			r = timerfd_settime(ctx->timerfd, TFD_TIMER_ABSTIME, &it, NULL);
			if (r < 0)
				return LIBUSB_ERROR_OTHER;
			return 0;
		}
	}

disarm:
	return disarm_timerfd(ctx);
}
#else
static int arm_timerfd_for_next_timeout(struct libusb_context *ctx)
{
	UNUSED(ctx);
	return 0;
}
#endif

static int add_to_flying_list(struct usbi_transfer *transfer)
{
	struct usbi_transfer *cur;
	struct timeval *timeout = &transfer->timeout;
	struct libusb_context *ctx = ITRANSFER_CTX(transfer);
	int r;
	int first = 1;

	r = calculate_timeout(transfer);
	if (r)
		return r;

	if (list_empty(&ctx->flying_transfers)) {
		list_add(&transfer->list, &ctx->flying_transfers);
		goto out;
	}

	if (!timerisset(timeout)) {
		list_add_tail(&transfer->list, &ctx->flying_transfers);

		goto out;
	}

	list_for_each_entry(cur, &ctx->flying_transfers, list, struct usbi_transfer) {

		struct timeval *cur_tv = &cur->timeout;

		if (!timerisset(cur_tv) || (cur_tv->tv_sec > timeout->tv_sec) ||
				(cur_tv->tv_sec == timeout->tv_sec &&
					cur_tv->tv_usec > timeout->tv_usec)) {
			list_add_tail(&transfer->list, &cur->list);
			goto out;
		}
		first = 0;
	}

	list_add_tail(&transfer->list, &ctx->flying_transfers);
out:
#ifdef USBI_TIMERFD_AVAILABLE
	if (first && usbi_using_timerfd(ctx) && timerisset(timeout)) {

		const struct itimerspec it = { {0, 0},
			{ timeout->tv_sec, timeout->tv_usec * 1000 } };
		usbi_dbg("arm timerfd for timeout in %dms (first in line)",
			USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer)->timeout);
		r = timerfd_settime(ctx->timerfd, TFD_TIMER_ABSTIME, &it, NULL);
		if (r < 0) {
			usbi_warn(ctx, "failed to arm first timerfd (errno %d)", errno);
			r = LIBUSB_ERROR_OTHER;
		}
	}
#else
	UNUSED(first);
#endif

	if (r)
		list_del(&transfer->list);

	return r;
}

static int remove_from_flying_list(struct usbi_transfer *transfer)
{
	struct libusb_context *ctx = ITRANSFER_CTX(transfer);
	int rearm_timerfd;
	int r = 0;

	usbi_mutex_lock(&ctx->flying_transfers_lock);
	rearm_timerfd = (timerisset(&transfer->timeout) &&
		list_first_entry(&ctx->flying_transfers, struct usbi_transfer, list) == transfer);
	list_del(&transfer->list);
	if (usbi_using_timerfd(ctx) && rearm_timerfd)
		r = arm_timerfd_for_next_timeout(ctx);
	usbi_mutex_unlock(&ctx->flying_transfers_lock);

	return r;
}

int API_EXPORTED libusb_submit_transfer(struct libusb_transfer *transfer)
{
	struct usbi_transfer *itransfer =
		LIBUSB_TRANSFER_TO_USBI_TRANSFER(transfer);
	struct libusb_context *ctx = TRANSFER_CTX(transfer);
	int r;

	usbi_dbg("transfer %p", transfer);

	usbi_mutex_lock(&ctx->flying_transfers_lock);
	usbi_mutex_lock(&itransfer->lock);
	if (itransfer->state_flags & USBI_TRANSFER_IN_FLIGHT) {
		usbi_mutex_unlock(&ctx->flying_transfers_lock);
		usbi_mutex_unlock(&itransfer->lock);
		return LIBUSB_ERROR_BUSY;
	}
	itransfer->transferred = 0;
	itransfer->state_flags = 0;
	itransfer->timeout_flags = 0;
	r = add_to_flying_list(itransfer);
	if (r) {
		usbi_mutex_unlock(&ctx->flying_transfers_lock);
		usbi_mutex_unlock(&itransfer->lock);
		return r;
	}

	usbi_mutex_unlock(&ctx->flying_transfers_lock);

	r = usbi_backend->submit_transfer(itransfer);
	if (r == LIBUSB_SUCCESS) {
		itransfer->state_flags |= USBI_TRANSFER_IN_FLIGHT;

		libusb_ref_device(transfer->dev_handle->dev);
	}
	usbi_mutex_unlock(&itransfer->lock);

	if (r != LIBUSB_SUCCESS)
		remove_from_flying_list(itransfer);

	return r;
}

int API_EXPORTED libusb_cancel_transfer(struct libusb_transfer *transfer)
{
	struct usbi_transfer *itransfer =
		LIBUSB_TRANSFER_TO_USBI_TRANSFER(transfer);
	int r;

	usbi_dbg("transfer %p", transfer );
	usbi_mutex_lock(&itransfer->lock);
	if (!(itransfer->state_flags & USBI_TRANSFER_IN_FLIGHT)
			|| (itransfer->state_flags & USBI_TRANSFER_CANCELLING)) {
		r = LIBUSB_ERROR_NOT_FOUND;
		goto out;
	}
	r = usbi_backend->cancel_transfer(itransfer);
	if (r < 0) {
		if (r != LIBUSB_ERROR_NOT_FOUND &&
		    r != LIBUSB_ERROR_NO_DEVICE)
			usbi_err(TRANSFER_CTX(transfer),
				"cancel transfer failed error %d", r);
		else
			usbi_dbg("cancel transfer failed error %d", r);

		if (r == LIBUSB_ERROR_NO_DEVICE)
			itransfer->state_flags |= USBI_TRANSFER_DEVICE_DISAPPEARED;
	}

	itransfer->state_flags |= USBI_TRANSFER_CANCELLING;

out:
	usbi_mutex_unlock(&itransfer->lock);
	return r;
}

void API_EXPORTED libusb_transfer_set_stream_id(
	struct libusb_transfer *transfer, uint32_t stream_id)
{
	struct usbi_transfer *itransfer =
		LIBUSB_TRANSFER_TO_USBI_TRANSFER(transfer);

	itransfer->stream_id = stream_id;
}

uint32_t API_EXPORTED libusb_transfer_get_stream_id(
	struct libusb_transfer *transfer)
{
	struct usbi_transfer *itransfer =
		LIBUSB_TRANSFER_TO_USBI_TRANSFER(transfer);

	return itransfer->stream_id;
}

int usbi_handle_transfer_completion(struct usbi_transfer *itransfer,
	enum libusb_transfer_status status)
{
	struct libusb_transfer *transfer =
		USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct libusb_device_handle *dev_handle = transfer->dev_handle;
	uint8_t flags;
	int r;

	r = remove_from_flying_list(itransfer);
	if (r < 0)
		usbi_err(ITRANSFER_CTX(itransfer), "failed to set timer for next timeout, errno=%d", errno);

	usbi_mutex_lock(&itransfer->lock);
	itransfer->state_flags &= ~USBI_TRANSFER_IN_FLIGHT;
	usbi_mutex_unlock(&itransfer->lock);

	if (status == LIBUSB_TRANSFER_COMPLETED
			&& transfer->flags & LIBUSB_TRANSFER_SHORT_NOT_OK) {
		int rqlen = transfer->length;
		if (transfer->type == LIBUSB_TRANSFER_TYPE_CONTROL)
			rqlen -= LIBUSB_CONTROL_SETUP_SIZE;
		if (rqlen != itransfer->transferred) {
			usbi_dbg("interpreting short transfer as error");
			status = LIBUSB_TRANSFER_ERROR;
		}
	}

	flags = transfer->flags;
	transfer->status = status;
	transfer->actual_length = itransfer->transferred;
	usbi_dbg("transfer %p has callback %p", transfer, transfer->callback);
	if (transfer->callback)
		transfer->callback(transfer);

	if (flags & LIBUSB_TRANSFER_FREE_TRANSFER)
		libusb_free_transfer(transfer);
	libusb_unref_device(dev_handle->dev);
	return r;
}

int usbi_handle_transfer_cancellation(struct usbi_transfer *transfer)
{
	struct libusb_context *ctx = ITRANSFER_CTX(transfer);
	uint8_t timed_out;

	usbi_mutex_lock(&ctx->flying_transfers_lock);
	timed_out = transfer->timeout_flags & USBI_TRANSFER_TIMED_OUT;
	usbi_mutex_unlock(&ctx->flying_transfers_lock);

	if (timed_out) {
		usbi_dbg("detected timeout cancellation");
		return usbi_handle_transfer_completion(transfer, LIBUSB_TRANSFER_TIMED_OUT);
	}

	return usbi_handle_transfer_completion(transfer, LIBUSB_TRANSFER_CANCELLED);
}

void usbi_signal_transfer_completion(struct usbi_transfer *transfer)
{
	struct libusb_context *ctx = ITRANSFER_CTX(transfer);
	int pending_events;

	usbi_mutex_lock(&ctx->event_data_lock);
	pending_events = usbi_pending_events(ctx);
	list_add_tail(&transfer->completed_list, &ctx->completed_transfers);
	if (!pending_events)
		usbi_signal_event(ctx);
	usbi_mutex_unlock(&ctx->event_data_lock);
}

int API_EXPORTED libusb_try_lock_events(libusb_context *ctx)
{
	int r;
	unsigned int ru;
	USBI_GET_CONTEXT(ctx);

	usbi_mutex_lock(&ctx->event_data_lock);
	ru = ctx->device_close;
	usbi_mutex_unlock(&ctx->event_data_lock);
	if (ru) {
		usbi_dbg("someone else is closing a device");
		return 1;
	}

	r = usbi_mutex_trylock(&ctx->events_lock);
	if (r)
		return 1;

	ctx->event_handler_active = 1;
	return 0;
}

void API_EXPORTED libusb_lock_events(libusb_context *ctx)
{
	USBI_GET_CONTEXT(ctx);
	usbi_mutex_lock(&ctx->events_lock);
	ctx->event_handler_active = 1;
}

void API_EXPORTED libusb_unlock_events(libusb_context *ctx)
{
	USBI_GET_CONTEXT(ctx);
	ctx->event_handler_active = 0;
	usbi_mutex_unlock(&ctx->events_lock);

	usbi_mutex_lock(&ctx->event_waiters_lock);
	usbi_cond_broadcast(&ctx->event_waiters_cond);
	usbi_mutex_unlock(&ctx->event_waiters_lock);
}

int API_EXPORTED libusb_event_handling_ok(libusb_context *ctx)
{
	unsigned int r;
	USBI_GET_CONTEXT(ctx);

	usbi_mutex_lock(&ctx->event_data_lock);
	r = ctx->device_close;
	usbi_mutex_unlock(&ctx->event_data_lock);
	if (r) {
		usbi_dbg("someone else is closing a device");
		return 0;
	}

	return 1;
}

int API_EXPORTED libusb_event_handler_active(libusb_context *ctx)
{
	unsigned int r;
	USBI_GET_CONTEXT(ctx);

	usbi_mutex_lock(&ctx->event_data_lock);
	r = ctx->device_close;
	usbi_mutex_unlock(&ctx->event_data_lock);
	if (r) {
		usbi_dbg("someone else is closing a device");
		return 1;
	}

	return ctx->event_handler_active;
}

void API_EXPORTED libusb_interrupt_event_handler(libusb_context *ctx)
{
	int pending_events;
	USBI_GET_CONTEXT(ctx);

	usbi_dbg("");
	usbi_mutex_lock(&ctx->event_data_lock);

	pending_events = usbi_pending_events(ctx);
	ctx->event_flags |= USBI_EVENT_USER_INTERRUPT;
	if (!pending_events)
		usbi_signal_event(ctx);

	usbi_mutex_unlock(&ctx->event_data_lock);
}

void API_EXPORTED libusb_lock_event_waiters(libusb_context *ctx)
{
	USBI_GET_CONTEXT(ctx);
	usbi_mutex_lock(&ctx->event_waiters_lock);
}

void API_EXPORTED libusb_unlock_event_waiters(libusb_context *ctx)
{
	USBI_GET_CONTEXT(ctx);
	usbi_mutex_unlock(&ctx->event_waiters_lock);
}

int API_EXPORTED libusb_wait_for_event(libusb_context *ctx, struct timeval *tv)
{
	int r;

	USBI_GET_CONTEXT(ctx);
	if (tv == NULL) {
		usbi_cond_wait(&ctx->event_waiters_cond, &ctx->event_waiters_lock);
		return 0;
	}

	r = usbi_cond_timedwait(&ctx->event_waiters_cond,
		&ctx->event_waiters_lock, tv);

	if (r < 0)
		return r;
	else
		return (r == ETIMEDOUT);
}

static void handle_timeout(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer =
		USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	int r;

	itransfer->timeout_flags |= USBI_TRANSFER_TIMEOUT_HANDLED;
	r = libusb_cancel_transfer(transfer);
	if (r == LIBUSB_SUCCESS)
		itransfer->timeout_flags |= USBI_TRANSFER_TIMED_OUT;
	else
		usbi_warn(TRANSFER_CTX(transfer),
			"async cancel failed %d errno=%d", r, errno);
}

static int handle_timeouts_locked(struct libusb_context *ctx)
{
	int r;
	struct timespec systime_ts;
	struct timeval systime;
	struct usbi_transfer *transfer;

	if (list_empty(&ctx->flying_transfers))
		return 0;

	r = usbi_backend->clock_gettime(USBI_CLOCK_MONOTONIC, &systime_ts);
	if (r < 0)
		return r;

	TIMESPEC_TO_TIMEVAL(&systime, &systime_ts);

	list_for_each_entry(transfer, &ctx->flying_transfers, list, struct usbi_transfer) {
		struct timeval *cur_tv = &transfer->timeout;

		if (!timerisset(cur_tv))
			return 0;

		if (transfer->timeout_flags & (USBI_TRANSFER_TIMEOUT_HANDLED | USBI_TRANSFER_OS_HANDLES_TIMEOUT))
			continue;

		if ((cur_tv->tv_sec > systime.tv_sec) ||
				(cur_tv->tv_sec == systime.tv_sec &&
					cur_tv->tv_usec > systime.tv_usec))
			return 0;

		handle_timeout(transfer);
	}
	return 0;
}

static int handle_timeouts(struct libusb_context *ctx)
{
	int r;
	USBI_GET_CONTEXT(ctx);
	usbi_mutex_lock(&ctx->flying_transfers_lock);
	r = handle_timeouts_locked(ctx);
	usbi_mutex_unlock(&ctx->flying_transfers_lock);
	return r;
}

#ifdef USBI_TIMERFD_AVAILABLE
static int handle_timerfd_trigger(struct libusb_context *ctx)
{
	int r;

	usbi_mutex_lock(&ctx->flying_transfers_lock);

	r = handle_timeouts_locked(ctx);
	if (r < 0)
		goto out;

	r = arm_timerfd_for_next_timeout(ctx);

out:
	usbi_mutex_unlock(&ctx->flying_transfers_lock);
	return r;
}
#endif

static int handle_events(struct libusb_context *ctx, struct timeval *tv)
{
	int r;
	struct usbi_pollfd *ipollfd;
	POLL_NFDS_TYPE nfds = 0;
	POLL_NFDS_TYPE internal_nfds;
	struct pollfd *fds = NULL;
	int i = -1;
	int timeout_ms;
	int special_event;

	if (usbi_handling_events(ctx))
		return LIBUSB_ERROR_BUSY;
	usbi_start_event_handling(ctx);

	if (usbi_using_timerfd(ctx))
		internal_nfds = 2;
	else
		internal_nfds = 1;

	usbi_mutex_lock(&ctx->event_data_lock);
	if (ctx->event_flags & USBI_EVENT_POLLFDS_MODIFIED) {
		usbi_dbg("poll fds modified, reallocating");

		if (ctx->pollfds) {
			free(ctx->pollfds);
			ctx->pollfds = NULL;
		}

		assert(ctx->pollfds_cnt >= internal_nfds);

		ctx->pollfds = calloc(ctx->pollfds_cnt, sizeof(*ctx->pollfds));
		if (!ctx->pollfds) {
			usbi_mutex_unlock(&ctx->event_data_lock);
			r = LIBUSB_ERROR_NO_MEM;
			goto done;
		}

		list_for_each_entry(ipollfd, &ctx->ipollfds, list, struct usbi_pollfd) {
			struct libusb_pollfd *pollfd = &ipollfd->pollfd;
			i++;
			ctx->pollfds[i].fd = pollfd->fd;
			ctx->pollfds[i].events = pollfd->events;
		}

		ctx->event_flags &= ~USBI_EVENT_POLLFDS_MODIFIED;

		if (!usbi_pending_events(ctx))
			usbi_clear_event(ctx);
	}
	fds = ctx->pollfds;
	nfds = ctx->pollfds_cnt;
	usbi_mutex_unlock(&ctx->event_data_lock);

	timeout_ms = (int)(tv->tv_sec * 1000) + (tv->tv_usec / 1000);

	if (tv->tv_usec % 1000)
		timeout_ms++;

redo_poll:
	usbi_dbg("poll() %d fds with timeout in %dms", nfds, timeout_ms);
	r = usbi_poll(fds, nfds, timeout_ms);
	usbi_dbg("poll() returned %d", r);
	if (r == 0) {
		r = handle_timeouts(ctx);
		goto done;
	}
	else if (r == -1 && errno == EINTR) {
		r = LIBUSB_ERROR_INTERRUPTED;
		goto done;
	}
	else if (r < 0) {
		usbi_err(ctx, "poll failed %d err=%d", r, errno);
		r = LIBUSB_ERROR_IO;
		goto done;
	}

	special_event = 0;

	if (fds[0].revents) {
		libusb_hotplug_message *message = NULL;
		struct usbi_transfer *itransfer;
		int ret = 0;

		usbi_dbg("caught a fish on the event pipe");

		usbi_mutex_lock(&ctx->event_data_lock);

		if (ctx->event_flags & USBI_EVENT_POLLFDS_MODIFIED)
			usbi_dbg("someone updated the poll fds");

		if (ctx->event_flags & USBI_EVENT_USER_INTERRUPT) {
			usbi_dbg("someone purposely interrupted");
			ctx->event_flags &= ~USBI_EVENT_USER_INTERRUPT;
		}

		if (ctx->device_close)
			usbi_dbg("someone is closing a device");

		if (!list_empty(&ctx->hotplug_msgs)) {
			usbi_dbg("hotplug message received");
			special_event = 1;
			message = list_first_entry(&ctx->hotplug_msgs, libusb_hotplug_message, list);
			list_del(&message->list);
		}

		while (ret == 0 && !list_empty(&ctx->completed_transfers)) {
			itransfer = list_first_entry(&ctx->completed_transfers, struct usbi_transfer, completed_list);
			list_del(&itransfer->completed_list);
			usbi_mutex_unlock(&ctx->event_data_lock);
			ret = usbi_backend->handle_transfer_completion(itransfer);
			if (ret)
				usbi_err(ctx, "backend handle_transfer_completion failed with error %d", ret);
			usbi_mutex_lock(&ctx->event_data_lock);
		}

		if (!usbi_pending_events(ctx))
			usbi_clear_event(ctx);

		usbi_mutex_unlock(&ctx->event_data_lock);

		if (message) {
			usbi_hotplug_match(ctx, message->device, message->event);

			if (LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT == message->event)
				libusb_unref_device(message->device);

			free(message);
		}

		if (ret) {

			r = ret;
			goto done;
		}

		if (0 == --r)
			goto handled;
	}

#ifdef USBI_TIMERFD_AVAILABLE

	if (usbi_using_timerfd(ctx) && fds[1].revents) {

		int ret;
		usbi_dbg("timerfd triggered");
		special_event = 1;

		ret = handle_timerfd_trigger(ctx);
		if (ret < 0) {

			r = ret;
			goto done;
		}

		if (0 == --r)
			goto handled;
	}
#endif

	r = usbi_backend->handle_events(ctx, fds + internal_nfds, nfds - internal_nfds, r);
	if (r)
		usbi_err(ctx, "backend handle_events failed with error %d", r);

handled:
	if (r == 0 && special_event) {
		timeout_ms = 0;
		goto redo_poll;
	}

done:
	usbi_end_event_handling(ctx);
	return r;
}

static int get_next_timeout(libusb_context *ctx, struct timeval *tv,
	struct timeval *out)
{
	struct timeval timeout;
	int r = libusb_get_next_timeout(ctx, &timeout);
	if (r) {

		if (!timerisset(&timeout))
			return 1;

		if (timercmp(&timeout, tv, <))
			*out = timeout;
		else
			*out = *tv;
	} else {
		*out = *tv;
	}
	return 0;
}

int API_EXPORTED libusb_handle_events_timeout_completed(libusb_context *ctx,
	struct timeval *tv, int *completed)
{
	int r;
	struct timeval poll_timeout;

	USBI_GET_CONTEXT(ctx);
	r = get_next_timeout(ctx, tv, &poll_timeout);
	if (r) {

		return handle_timeouts(ctx);
	}

retry:
	if (libusb_try_lock_events(ctx) == 0) {
		if (completed == NULL || !*completed) {

			usbi_dbg("doing our own event handling");
			r = handle_events(ctx, &poll_timeout);
		}
		libusb_unlock_events(ctx);
		return r;
	}

	libusb_lock_event_waiters(ctx);

	if (completed && *completed)
		goto already_done;

	if (!libusb_event_handler_active(ctx)) {

		libusb_unlock_event_waiters(ctx);
		usbi_dbg("event handler was active but went away, retrying");
		goto retry;
	}

	usbi_dbg("another thread is doing event handling");
	r = libusb_wait_for_event(ctx, &poll_timeout);

already_done:
	libusb_unlock_event_waiters(ctx);

	if (r < 0)
		return r;
	else if (r == 1)
		return handle_timeouts(ctx);
	else
		return 0;
}

int API_EXPORTED libusb_handle_events_timeout(libusb_context *ctx,
	struct timeval *tv)
{
	return libusb_handle_events_timeout_completed(ctx, tv, NULL);
}

int API_EXPORTED libusb_handle_events(libusb_context *ctx)
{
	struct timeval tv;
	tv.tv_sec = 60;
	tv.tv_usec = 0;
	return libusb_handle_events_timeout_completed(ctx, &tv, NULL);
}

int API_EXPORTED libusb_handle_events_completed(libusb_context *ctx,
	int *completed)
{
	struct timeval tv;
	tv.tv_sec = 60;
	tv.tv_usec = 0;
	return libusb_handle_events_timeout_completed(ctx, &tv, completed);
}

int API_EXPORTED libusb_handle_events_locked(libusb_context *ctx,
	struct timeval *tv)
{
	int r;
	struct timeval poll_timeout;

	USBI_GET_CONTEXT(ctx);
	r = get_next_timeout(ctx, tv, &poll_timeout);
	if (r) {

		return handle_timeouts(ctx);
	}

	return handle_events(ctx, &poll_timeout);
}

int API_EXPORTED libusb_pollfds_handle_timeouts(libusb_context *ctx)
{
#if defined(USBI_TIMERFD_AVAILABLE)
	USBI_GET_CONTEXT(ctx);
	return usbi_using_timerfd(ctx);
#else
	UNUSED(ctx);
	return 0;
#endif
}

int API_EXPORTED libusb_get_next_timeout(libusb_context *ctx,
	struct timeval *tv)
{
	struct usbi_transfer *transfer;
	struct timespec cur_ts;
	struct timeval cur_tv;
	struct timeval next_timeout = { 0, 0 };
	int r;

	USBI_GET_CONTEXT(ctx);
	if (usbi_using_timerfd(ctx))
		return 0;

	usbi_mutex_lock(&ctx->flying_transfers_lock);
	if (list_empty(&ctx->flying_transfers)) {
		usbi_mutex_unlock(&ctx->flying_transfers_lock);
		usbi_dbg("no URBs, no timeout!");
		return 0;
	}

	list_for_each_entry(transfer, &ctx->flying_transfers, list, struct usbi_transfer) {
		if (transfer->timeout_flags & (USBI_TRANSFER_TIMEOUT_HANDLED | USBI_TRANSFER_OS_HANDLES_TIMEOUT))
			continue;

		if (!timerisset(&transfer->timeout))
			break;

		next_timeout = transfer->timeout;
		break;
	}
	usbi_mutex_unlock(&ctx->flying_transfers_lock);

	if (!timerisset(&next_timeout)) {
		usbi_dbg("no URB with timeout or all handled by OS; no timeout!");
		return 0;
	}

	r = usbi_backend->clock_gettime(USBI_CLOCK_MONOTONIC, &cur_ts);
	if (r < 0) {
		usbi_err(ctx, "failed to read monotonic clock, errno=%d", errno);
		return 0;
	}
	TIMESPEC_TO_TIMEVAL(&cur_tv, &cur_ts);

	if (!timercmp(&cur_tv, &next_timeout, <)) {
		usbi_dbg("first timeout already expired");
		timerclear(tv);
	} else {
		timersub(&next_timeout, &cur_tv, tv);
		usbi_dbg("next timeout in %d.%06ds", tv->tv_sec, tv->tv_usec);
	}

	return 1;
}

void API_EXPORTED libusb_set_pollfd_notifiers(libusb_context *ctx,
	libusb_pollfd_added_cb added_cb, libusb_pollfd_removed_cb removed_cb,
	void *user_data)
{
	USBI_GET_CONTEXT(ctx);
	ctx->fd_added_cb = added_cb;
	ctx->fd_removed_cb = removed_cb;
	ctx->fd_cb_user_data = user_data;
}

static void usbi_fd_notification(struct libusb_context *ctx)
{
	int pending_events;

	pending_events = usbi_pending_events(ctx);
	ctx->event_flags |= USBI_EVENT_POLLFDS_MODIFIED;
	if (!pending_events)
		usbi_signal_event(ctx);
}

int usbi_add_pollfd(struct libusb_context *ctx, int fd, short events)
{
	struct usbi_pollfd *ipollfd = malloc(sizeof(*ipollfd));
	if (!ipollfd)
		return LIBUSB_ERROR_NO_MEM;

	usbi_dbg("add fd %d events %d", fd, events);
	ipollfd->pollfd.fd = fd;
	ipollfd->pollfd.events = events;
	usbi_mutex_lock(&ctx->event_data_lock);
	list_add_tail(&ipollfd->list, &ctx->ipollfds);
	ctx->pollfds_cnt++;
	usbi_fd_notification(ctx);
	usbi_mutex_unlock(&ctx->event_data_lock);

	if (ctx->fd_added_cb)
		ctx->fd_added_cb(fd, events, ctx->fd_cb_user_data);
	return 0;
}

void usbi_remove_pollfd(struct libusb_context *ctx, int fd)
{
	struct usbi_pollfd *ipollfd;
	int found = 0;

	usbi_dbg("remove fd %d", fd);
	usbi_mutex_lock(&ctx->event_data_lock);
	list_for_each_entry(ipollfd, &ctx->ipollfds, list, struct usbi_pollfd)
		if (ipollfd->pollfd.fd == fd) {
			found = 1;
			break;
		}

	if (!found) {
		usbi_dbg("couldn't find fd %d to remove", fd);
		usbi_mutex_unlock(&ctx->event_data_lock);
		return;
	}

	list_del(&ipollfd->list);
	ctx->pollfds_cnt--;
	usbi_fd_notification(ctx);
	usbi_mutex_unlock(&ctx->event_data_lock);
	free(ipollfd);
	if (ctx->fd_removed_cb)
		ctx->fd_removed_cb(fd, ctx->fd_cb_user_data);
}

DEFAULT_VISIBILITY
const struct libusb_pollfd ** LIBUSB_CALL libusb_get_pollfds(
	libusb_context *ctx)
{
#ifndef OS_WINDOWS
	struct libusb_pollfd **ret = NULL;
	struct usbi_pollfd *ipollfd;
	size_t i = 0;
	USBI_GET_CONTEXT(ctx);

	usbi_mutex_lock(&ctx->event_data_lock);

	ret = calloc(ctx->pollfds_cnt + 1, sizeof(struct libusb_pollfd *));
	if (!ret)
		goto out;

	list_for_each_entry(ipollfd, &ctx->ipollfds, list, struct usbi_pollfd)
		ret[i++] = (struct libusb_pollfd *) ipollfd;
	ret[ctx->pollfds_cnt] = NULL;

out:
	usbi_mutex_unlock(&ctx->event_data_lock);
	return (const struct libusb_pollfd **) ret;
#else
	usbi_err(ctx, "external polling of libusb's internal descriptors "\
		"is not yet supported on Windows platforms");
	return NULL;
#endif
}

void API_EXPORTED libusb_free_pollfds(const struct libusb_pollfd **pollfds)
{
	if (!pollfds)
		return;

	free((void *)pollfds);
}

void usbi_handle_disconnect(struct libusb_device_handle *dev_handle)
{
	struct usbi_transfer *cur;
	struct usbi_transfer *to_cancel;

	usbi_dbg("device %d.%d",
		dev_handle->dev->bus_number, dev_handle->dev->device_address);

	while (1) {
		to_cancel = NULL;
		usbi_mutex_lock(&HANDLE_CTX(dev_handle)->flying_transfers_lock);
		list_for_each_entry(cur, &HANDLE_CTX(dev_handle)->flying_transfers, list, struct usbi_transfer)
			if (USBI_TRANSFER_TO_LIBUSB_TRANSFER(cur)->dev_handle == dev_handle) {
				usbi_mutex_lock(&cur->lock);
				if (cur->state_flags & USBI_TRANSFER_IN_FLIGHT)
					to_cancel = cur;
				usbi_mutex_unlock(&cur->lock);

				if (to_cancel)
					break;
			}
		usbi_mutex_unlock(&HANDLE_CTX(dev_handle)->flying_transfers_lock);

		if (!to_cancel)
			break;

		usbi_dbg("cancelling transfer %p from disconnect",
			 USBI_TRANSFER_TO_LIBUSB_TRANSFER(to_cancel));

		usbi_mutex_lock(&to_cancel->lock);
		usbi_backend->clear_transfer_priv(to_cancel);
		usbi_mutex_unlock(&to_cancel->lock);
		usbi_handle_transfer_completion(to_cancel, LIBUSB_TRANSFER_NO_DEVICE);
	}

}
