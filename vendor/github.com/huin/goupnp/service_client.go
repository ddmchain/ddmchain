package goupnp

import (
	"fmt"
	"net/url"

	"github.com/huin/goupnp/soap"
)

type ServiceClient struct {
	SOAPClient *soap.SOAPClient
	RootDevice *RootDevice
	Location   *url.URL
	Service    *Service
}

func NewServiceClients(searchTarget string) (clients []ServiceClient, errors []error, err error) {
	var maybeRootDevices []MaybeRootDevice
	if maybeRootDevices, err = DiscoverDevices(searchTarget); err != nil {
		return
	}

	clients = make([]ServiceClient, 0, len(maybeRootDevices))

	for _, maybeRootDevice := range maybeRootDevices {
		if maybeRootDevice.Err != nil {
			errors = append(errors, maybeRootDevice.Err)
			continue
		}

		deviceClients, err := NewServiceClientsFromRootDevice(maybeRootDevice.Root, maybeRootDevice.Location, searchTarget)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		clients = append(clients, deviceClients...)
	}

	return
}

func NewServiceClientsByURL(loc *url.URL, searchTarget string) ([]ServiceClient, error) {
	rootDevice, err := DeviceByURL(loc)
	if err != nil {
		return nil, err
	}
	return NewServiceClientsFromRootDevice(rootDevice, loc, searchTarget)
}

func NewServiceClientsFromRootDevice(rootDevice *RootDevice, loc *url.URL, searchTarget string) ([]ServiceClient, error) {
	device := &rootDevice.Device
	srvs := device.FindService(searchTarget)
	if len(srvs) == 0 {
		return nil, fmt.Errorf("goupnp: service %q not found within device %q (UDN=%q)",
			searchTarget, device.FriendlyName, device.UDN)
	}

	clients := make([]ServiceClient, 0, len(srvs))
	for _, srv := range srvs {
		clients = append(clients, ServiceClient{
			SOAPClient: srv.NewSOAPClient(),
			RootDevice: rootDevice,
			Location:   loc,
			Service:    srv,
		})
	}
	return clients, nil
}

func (client *ServiceClient) GetServiceClient() *ServiceClient {
	return client
}
