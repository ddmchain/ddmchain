
#ifndef	LIBUSB_SUNOS_H
#define	LIBUSB_SUNOS_H

#include <libdevinfo.h>
#include <pthread.h>
#include "libusbi.h"

#define	READ	0
#define	WRITE	1

typedef struct sunos_device_priv {
	uint8_t	cfgvalue;		
	uint8_t	*raw_cfgdescr;		
	struct libusb_device_descriptor	dev_descr;	
	char	*ugenpath;		
	char	*phypath;		
} sunos_dev_priv_t;

typedef	struct endpoint {
	int datafd;	
	int statfd;	
} sunos_ep_priv_t;

typedef struct sunos_device_handle_priv {
	uint8_t			altsetting[USB_MAXINTERFACES];	
	uint8_t			config_index;
	sunos_ep_priv_t		eps[USB_MAXENDPOINTS];
	sunos_dev_priv_t	*dpriv; 
} sunos_dev_handle_priv_t;

typedef	struct sunos_transfer_priv {
	struct aiocb		aiocb;
	struct libusb_transfer	*transfer;
} sunos_xfer_priv_t;

struct node_args {
	struct libusb_context	*ctx;
	struct discovered_devs	**discdevs;
	const char		*last_ugenpath;
	di_devlink_handle_t	dlink_hdl;
};

struct devlink_cbarg {
	struct node_args	*nargs;	
	di_node_t		myself;	
	di_minor_t		minor;
};

struct aio_callback_args{
	struct libusb_transfer *transfer;
	struct aiocb aiocb;
};

#endif 
