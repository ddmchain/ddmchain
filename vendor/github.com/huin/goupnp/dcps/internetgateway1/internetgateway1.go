
package internetgateway1

import (
	"net/url"
	"time"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/soap"
)

var _ time.Time

const (
	URN_LANDevice_1           = "urn:schemas-upnp-org:device:LANDevice:1"
	URN_WANConnectionDevice_1 = "urn:schemas-upnp-org:device:WANConnectionDevice:1"
	URN_WANDevice_1           = "urn:schemas-upnp-org:device:WANDevice:1"
)

const (
	URN_LANHostConfigManagement_1  = "urn:schemas-upnp-org:service:LANHostConfigManagement:1"
	URN_Layer3Forwarding_1         = "urn:schemas-upnp-org:service:Layer3Forwarding:1"
	URN_WANCableLinkConfig_1       = "urn:schemas-upnp-org:service:WANCableLinkConfig:1"
	URN_WANCommonInterfaceConfig_1 = "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1"
	URN_WANDSLLinkConfig_1         = "urn:schemas-upnp-org:service:WANDSLLinkConfig:1"
	URN_WANEthernetLinkConfig_1    = "urn:schemas-upnp-org:service:WANEthernetLinkConfig:1"
	URN_WANIPConnection_1          = "urn:schemas-upnp-org:service:WANIPConnection:1"
	URN_WANPOTSLinkConfig_1        = "urn:schemas-upnp-org:service:WANPOTSLinkConfig:1"
	URN_WANPPPConnection_1         = "urn:schemas-upnp-org:service:WANPPPConnection:1"
)

type LANHostConfigManagement1 struct {
	goupnp.ServiceClient
}

func NewLANHostConfigManagement1Clients() (clients []*LANHostConfigManagement1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients(URN_LANHostConfigManagement_1); err != nil {
		return
	}
	clients = newLANHostConfigManagement1ClientsFromGenericClients(genericClients)
	return
}

func NewLANHostConfigManagement1ClientsByURL(loc *url.URL) ([]*LANHostConfigManagement1, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, URN_LANHostConfigManagement_1)
	if err != nil {
		return nil, err
	}
	return newLANHostConfigManagement1ClientsFromGenericClients(genericClients), nil
}

func NewLANHostConfigManagement1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*LANHostConfigManagement1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_LANHostConfigManagement_1)
	if err != nil {
		return nil, err
	}
	return newLANHostConfigManagement1ClientsFromGenericClients(genericClients), nil
}

func newLANHostConfigManagement1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*LANHostConfigManagement1 {
	clients := make([]*LANHostConfigManagement1, len(genericClients))
	for i := range genericClients {
		clients[i] = &LANHostConfigManagement1{genericClients[i]}
	}
	return clients
}

func (client *LANHostConfigManagement1) SetDHCPServerConfigurable(NewDHCPServerConfigurable bool) (err error) {

	request := &struct {
		NewDHCPServerConfigurable string
	}{}

	if request.NewDHCPServerConfigurable, err = soap.MarshalBoolean(NewDHCPServerConfigurable); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "SetDHCPServerConfigurable", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) GetDHCPServerConfigurable() (NewDHCPServerConfigurable bool, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDHCPServerConfigurable string
	}{}

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "GetDHCPServerConfigurable", request, response); err != nil {
		return
	}

	if NewDHCPServerConfigurable, err = soap.UnmarshalBoolean(response.NewDHCPServerConfigurable); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) SetDHCPRelay(NewDHCPRelay bool) (err error) {

	request := &struct {
		NewDHCPRelay string
	}{}

	if request.NewDHCPRelay, err = soap.MarshalBoolean(NewDHCPRelay); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "SetDHCPRelay", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) GetDHCPRelay() (NewDHCPRelay bool, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDHCPRelay string
	}{}

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "GetDHCPRelay", request, response); err != nil {
		return
	}

	if NewDHCPRelay, err = soap.UnmarshalBoolean(response.NewDHCPRelay); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) SetSubnetMask(NewSubnetMask string) (err error) {

	request := &struct {
		NewSubnetMask string
	}{}

	if request.NewSubnetMask, err = soap.MarshalString(NewSubnetMask); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "SetSubnetMask", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) GetSubnetMask() (NewSubnetMask string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewSubnetMask string
	}{}

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "GetSubnetMask", request, response); err != nil {
		return
	}

	if NewSubnetMask, err = soap.UnmarshalString(response.NewSubnetMask); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) SetIPRouter(NewIPRouters string) (err error) {

	request := &struct {
		NewIPRouters string
	}{}

	if request.NewIPRouters, err = soap.MarshalString(NewIPRouters); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "SetIPRouter", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) DeleteIPRouter(NewIPRouters string) (err error) {

	request := &struct {
		NewIPRouters string
	}{}

	if request.NewIPRouters, err = soap.MarshalString(NewIPRouters); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "DeleteIPRouter", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) GetIPRoutersList() (NewIPRouters string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewIPRouters string
	}{}

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "GetIPRoutersList", request, response); err != nil {
		return
	}

	if NewIPRouters, err = soap.UnmarshalString(response.NewIPRouters); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) SetDomainName(NewDomainName string) (err error) {

	request := &struct {
		NewDomainName string
	}{}

	if request.NewDomainName, err = soap.MarshalString(NewDomainName); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "SetDomainName", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) GetDomainName() (NewDomainName string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDomainName string
	}{}

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "GetDomainName", request, response); err != nil {
		return
	}

	if NewDomainName, err = soap.UnmarshalString(response.NewDomainName); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) SetAddressRange(NewMinAddress string, NewMaxAddress string) (err error) {

	request := &struct {
		NewMinAddress string

		NewMaxAddress string
	}{}

	if request.NewMinAddress, err = soap.MarshalString(NewMinAddress); err != nil {
		return
	}
	if request.NewMaxAddress, err = soap.MarshalString(NewMaxAddress); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "SetAddressRange", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) GetAddressRange() (NewMinAddress string, NewMaxAddress string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewMinAddress string

		NewMaxAddress string
	}{}

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "GetAddressRange", request, response); err != nil {
		return
	}

	if NewMinAddress, err = soap.UnmarshalString(response.NewMinAddress); err != nil {
		return
	}
	if NewMaxAddress, err = soap.UnmarshalString(response.NewMaxAddress); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) SetReservedAddress(NewReservedAddresses string) (err error) {

	request := &struct {
		NewReservedAddresses string
	}{}

	if request.NewReservedAddresses, err = soap.MarshalString(NewReservedAddresses); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "SetReservedAddress", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) DeleteReservedAddress(NewReservedAddresses string) (err error) {

	request := &struct {
		NewReservedAddresses string
	}{}

	if request.NewReservedAddresses, err = soap.MarshalString(NewReservedAddresses); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "DeleteReservedAddress", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) GetReservedAddresses() (NewReservedAddresses string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewReservedAddresses string
	}{}

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "GetReservedAddresses", request, response); err != nil {
		return
	}

	if NewReservedAddresses, err = soap.UnmarshalString(response.NewReservedAddresses); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) SetDNSServer(NewDNSServers string) (err error) {

	request := &struct {
		NewDNSServers string
	}{}

	if request.NewDNSServers, err = soap.MarshalString(NewDNSServers); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "SetDNSServer", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) DeleteDNSServer(NewDNSServers string) (err error) {

	request := &struct {
		NewDNSServers string
	}{}

	if request.NewDNSServers, err = soap.MarshalString(NewDNSServers); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "DeleteDNSServer", request, response); err != nil {
		return
	}

	return
}

func (client *LANHostConfigManagement1) GetDNSServers() (NewDNSServers string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDNSServers string
	}{}

	if err = client.SOAPClient.PerformAction(URN_LANHostConfigManagement_1, "GetDNSServers", request, response); err != nil {
		return
	}

	if NewDNSServers, err = soap.UnmarshalString(response.NewDNSServers); err != nil {
		return
	}

	return
}

type Layer3Forwarding1 struct {
	goupnp.ServiceClient
}

func NewLayer3Forwarding1Clients() (clients []*Layer3Forwarding1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients(URN_Layer3Forwarding_1); err != nil {
		return
	}
	clients = newLayer3Forwarding1ClientsFromGenericClients(genericClients)
	return
}

func NewLayer3Forwarding1ClientsByURL(loc *url.URL) ([]*Layer3Forwarding1, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, URN_Layer3Forwarding_1)
	if err != nil {
		return nil, err
	}
	return newLayer3Forwarding1ClientsFromGenericClients(genericClients), nil
}

func NewLayer3Forwarding1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*Layer3Forwarding1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_Layer3Forwarding_1)
	if err != nil {
		return nil, err
	}
	return newLayer3Forwarding1ClientsFromGenericClients(genericClients), nil
}

func newLayer3Forwarding1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*Layer3Forwarding1 {
	clients := make([]*Layer3Forwarding1, len(genericClients))
	for i := range genericClients {
		clients[i] = &Layer3Forwarding1{genericClients[i]}
	}
	return clients
}

func (client *Layer3Forwarding1) SetDefaultConnectionService(NewDefaultConnectionService string) (err error) {

	request := &struct {
		NewDefaultConnectionService string
	}{}

	if request.NewDefaultConnectionService, err = soap.MarshalString(NewDefaultConnectionService); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_Layer3Forwarding_1, "SetDefaultConnectionService", request, response); err != nil {
		return
	}

	return
}

func (client *Layer3Forwarding1) GetDefaultConnectionService() (NewDefaultConnectionService string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDefaultConnectionService string
	}{}

	if err = client.SOAPClient.PerformAction(URN_Layer3Forwarding_1, "GetDefaultConnectionService", request, response); err != nil {
		return
	}

	if NewDefaultConnectionService, err = soap.UnmarshalString(response.NewDefaultConnectionService); err != nil {
		return
	}

	return
}

type WANCableLinkConfig1 struct {
	goupnp.ServiceClient
}

func NewWANCableLinkConfig1Clients() (clients []*WANCableLinkConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients(URN_WANCableLinkConfig_1); err != nil {
		return
	}
	clients = newWANCableLinkConfig1ClientsFromGenericClients(genericClients)
	return
}

func NewWANCableLinkConfig1ClientsByURL(loc *url.URL) ([]*WANCableLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, URN_WANCableLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANCableLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func NewWANCableLinkConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANCableLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANCableLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANCableLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANCableLinkConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANCableLinkConfig1 {
	clients := make([]*WANCableLinkConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANCableLinkConfig1{genericClients[i]}
	}
	return clients
}

func (client *WANCableLinkConfig1) GetCableLinkConfigInfo() (NewCableLinkConfigState string, NewLinkType string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewCableLinkConfigState string

		NewLinkType string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetCableLinkConfigInfo", request, response); err != nil {
		return
	}

	if NewCableLinkConfigState, err = soap.UnmarshalString(response.NewCableLinkConfigState); err != nil {
		return
	}
	if NewLinkType, err = soap.UnmarshalString(response.NewLinkType); err != nil {
		return
	}

	return
}

func (client *WANCableLinkConfig1) GetDownstreamFrequency() (NewDownstreamFrequency uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDownstreamFrequency string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetDownstreamFrequency", request, response); err != nil {
		return
	}

	if NewDownstreamFrequency, err = soap.UnmarshalUi4(response.NewDownstreamFrequency); err != nil {
		return
	}

	return
}

func (client *WANCableLinkConfig1) GetDownstreamModulation() (NewDownstreamModulation string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDownstreamModulation string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetDownstreamModulation", request, response); err != nil {
		return
	}

	if NewDownstreamModulation, err = soap.UnmarshalString(response.NewDownstreamModulation); err != nil {
		return
	}

	return
}

func (client *WANCableLinkConfig1) GetUpstreamFrequency() (NewUpstreamFrequency uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewUpstreamFrequency string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetUpstreamFrequency", request, response); err != nil {
		return
	}

	if NewUpstreamFrequency, err = soap.UnmarshalUi4(response.NewUpstreamFrequency); err != nil {
		return
	}

	return
}

func (client *WANCableLinkConfig1) GetUpstreamModulation() (NewUpstreamModulation string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewUpstreamModulation string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetUpstreamModulation", request, response); err != nil {
		return
	}

	if NewUpstreamModulation, err = soap.UnmarshalString(response.NewUpstreamModulation); err != nil {
		return
	}

	return
}

func (client *WANCableLinkConfig1) GetUpstreamChannelID() (NewUpstreamChannelID uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewUpstreamChannelID string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetUpstreamChannelID", request, response); err != nil {
		return
	}

	if NewUpstreamChannelID, err = soap.UnmarshalUi4(response.NewUpstreamChannelID); err != nil {
		return
	}

	return
}

func (client *WANCableLinkConfig1) GetUpstreamPowerLevel() (NewUpstreamPowerLevel uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewUpstreamPowerLevel string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetUpstreamPowerLevel", request, response); err != nil {
		return
	}

	if NewUpstreamPowerLevel, err = soap.UnmarshalUi4(response.NewUpstreamPowerLevel); err != nil {
		return
	}

	return
}

func (client *WANCableLinkConfig1) GetBPIEncryptionEnabled() (NewBPIEncryptionEnabled bool, err error) {

	request := interface{}(nil)

	response := &struct {
		NewBPIEncryptionEnabled string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetBPIEncryptionEnabled", request, response); err != nil {
		return
	}

	if NewBPIEncryptionEnabled, err = soap.UnmarshalBoolean(response.NewBPIEncryptionEnabled); err != nil {
		return
	}

	return
}

func (client *WANCableLinkConfig1) GetConfigFile() (NewConfigFile string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewConfigFile string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetConfigFile", request, response); err != nil {
		return
	}

	if NewConfigFile, err = soap.UnmarshalString(response.NewConfigFile); err != nil {
		return
	}

	return
}

func (client *WANCableLinkConfig1) GetTFTPServer() (NewTFTPServer string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewTFTPServer string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCableLinkConfig_1, "GetTFTPServer", request, response); err != nil {
		return
	}

	if NewTFTPServer, err = soap.UnmarshalString(response.NewTFTPServer); err != nil {
		return
	}

	return
}

type WANCommonInterfaceConfig1 struct {
	goupnp.ServiceClient
}

func NewWANCommonInterfaceConfig1Clients() (clients []*WANCommonInterfaceConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients(URN_WANCommonInterfaceConfig_1); err != nil {
		return
	}
	clients = newWANCommonInterfaceConfig1ClientsFromGenericClients(genericClients)
	return
}

func NewWANCommonInterfaceConfig1ClientsByURL(loc *url.URL) ([]*WANCommonInterfaceConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, URN_WANCommonInterfaceConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANCommonInterfaceConfig1ClientsFromGenericClients(genericClients), nil
}

func NewWANCommonInterfaceConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANCommonInterfaceConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANCommonInterfaceConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANCommonInterfaceConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANCommonInterfaceConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANCommonInterfaceConfig1 {
	clients := make([]*WANCommonInterfaceConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANCommonInterfaceConfig1{genericClients[i]}
	}
	return clients
}

func (client *WANCommonInterfaceConfig1) SetEnabledForInternet(NewEnabledForInternet bool) (err error) {

	request := &struct {
		NewEnabledForInternet string
	}{}

	if request.NewEnabledForInternet, err = soap.MarshalBoolean(NewEnabledForInternet); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "SetEnabledForInternet", request, response); err != nil {
		return
	}

	return
}

func (client *WANCommonInterfaceConfig1) GetEnabledForInternet() (NewEnabledForInternet bool, err error) {

	request := interface{}(nil)

	response := &struct {
		NewEnabledForInternet string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "GetEnabledForInternet", request, response); err != nil {
		return
	}

	if NewEnabledForInternet, err = soap.UnmarshalBoolean(response.NewEnabledForInternet); err != nil {
		return
	}

	return
}

func (client *WANCommonInterfaceConfig1) GetCommonLinkProperties() (NewWANAccessType string, NewLayer1UpstreamMaxBitRate uint32, NewLayer1DownstreamMaxBitRate uint32, NewPhysicalLinkStatus string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewWANAccessType string

		NewLayer1UpstreamMaxBitRate string

		NewLayer1DownstreamMaxBitRate string

		NewPhysicalLinkStatus string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "GetCommonLinkProperties", request, response); err != nil {
		return
	}

	if NewWANAccessType, err = soap.UnmarshalString(response.NewWANAccessType); err != nil {
		return
	}
	if NewLayer1UpstreamMaxBitRate, err = soap.UnmarshalUi4(response.NewLayer1UpstreamMaxBitRate); err != nil {
		return
	}
	if NewLayer1DownstreamMaxBitRate, err = soap.UnmarshalUi4(response.NewLayer1DownstreamMaxBitRate); err != nil {
		return
	}
	if NewPhysicalLinkStatus, err = soap.UnmarshalString(response.NewPhysicalLinkStatus); err != nil {
		return
	}

	return
}

func (client *WANCommonInterfaceConfig1) GetWANAccessProvider() (NewWANAccessProvider string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewWANAccessProvider string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "GetWANAccessProvider", request, response); err != nil {
		return
	}

	if NewWANAccessProvider, err = soap.UnmarshalString(response.NewWANAccessProvider); err != nil {
		return
	}

	return
}

func (client *WANCommonInterfaceConfig1) GetMaximumActiveConnections() (NewMaximumActiveConnections uint16, err error) {

	request := interface{}(nil)

	response := &struct {
		NewMaximumActiveConnections string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "GetMaximumActiveConnections", request, response); err != nil {
		return
	}

	if NewMaximumActiveConnections, err = soap.UnmarshalUi2(response.NewMaximumActiveConnections); err != nil {
		return
	}

	return
}

func (client *WANCommonInterfaceConfig1) GetTotalBytesSent() (NewTotalBytesSent uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewTotalBytesSent string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "GetTotalBytesSent", request, response); err != nil {
		return
	}

	if NewTotalBytesSent, err = soap.UnmarshalUi4(response.NewTotalBytesSent); err != nil {
		return
	}

	return
}

func (client *WANCommonInterfaceConfig1) GetTotalBytesReceived() (NewTotalBytesReceived uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewTotalBytesReceived string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "GetTotalBytesReceived", request, response); err != nil {
		return
	}

	if NewTotalBytesReceived, err = soap.UnmarshalUi4(response.NewTotalBytesReceived); err != nil {
		return
	}

	return
}

func (client *WANCommonInterfaceConfig1) GetTotalPacketsSent() (NewTotalPacketsSent uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewTotalPacketsSent string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "GetTotalPacketsSent", request, response); err != nil {
		return
	}

	if NewTotalPacketsSent, err = soap.UnmarshalUi4(response.NewTotalPacketsSent); err != nil {
		return
	}

	return
}

func (client *WANCommonInterfaceConfig1) GetTotalPacketsReceived() (NewTotalPacketsReceived uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewTotalPacketsReceived string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "GetTotalPacketsReceived", request, response); err != nil {
		return
	}

	if NewTotalPacketsReceived, err = soap.UnmarshalUi4(response.NewTotalPacketsReceived); err != nil {
		return
	}

	return
}

func (client *WANCommonInterfaceConfig1) GetActiveConnection(NewActiveConnectionIndex uint16) (NewActiveConnDeviceContainer string, NewActiveConnectionServiceID string, err error) {

	request := &struct {
		NewActiveConnectionIndex string
	}{}

	if request.NewActiveConnectionIndex, err = soap.MarshalUi2(NewActiveConnectionIndex); err != nil {
		return
	}

	response := &struct {
		NewActiveConnDeviceContainer string

		NewActiveConnectionServiceID string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANCommonInterfaceConfig_1, "GetActiveConnection", request, response); err != nil {
		return
	}

	if NewActiveConnDeviceContainer, err = soap.UnmarshalString(response.NewActiveConnDeviceContainer); err != nil {
		return
	}
	if NewActiveConnectionServiceID, err = soap.UnmarshalString(response.NewActiveConnectionServiceID); err != nil {
		return
	}

	return
}

type WANDSLLinkConfig1 struct {
	goupnp.ServiceClient
}

func NewWANDSLLinkConfig1Clients() (clients []*WANDSLLinkConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients(URN_WANDSLLinkConfig_1); err != nil {
		return
	}
	clients = newWANDSLLinkConfig1ClientsFromGenericClients(genericClients)
	return
}

func NewWANDSLLinkConfig1ClientsByURL(loc *url.URL) ([]*WANDSLLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, URN_WANDSLLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANDSLLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func NewWANDSLLinkConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANDSLLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANDSLLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANDSLLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANDSLLinkConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANDSLLinkConfig1 {
	clients := make([]*WANDSLLinkConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANDSLLinkConfig1{genericClients[i]}
	}
	return clients
}

func (client *WANDSLLinkConfig1) SetDSLLinkType(NewLinkType string) (err error) {

	request := &struct {
		NewLinkType string
	}{}

	if request.NewLinkType, err = soap.MarshalString(NewLinkType); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "SetDSLLinkType", request, response); err != nil {
		return
	}

	return
}

func (client *WANDSLLinkConfig1) GetDSLLinkInfo() (NewLinkType string, NewLinkStatus string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewLinkType string

		NewLinkStatus string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "GetDSLLinkInfo", request, response); err != nil {
		return
	}

	if NewLinkType, err = soap.UnmarshalString(response.NewLinkType); err != nil {
		return
	}
	if NewLinkStatus, err = soap.UnmarshalString(response.NewLinkStatus); err != nil {
		return
	}

	return
}

func (client *WANDSLLinkConfig1) GetAutoConfig() (NewAutoConfig bool, err error) {

	request := interface{}(nil)

	response := &struct {
		NewAutoConfig string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "GetAutoConfig", request, response); err != nil {
		return
	}

	if NewAutoConfig, err = soap.UnmarshalBoolean(response.NewAutoConfig); err != nil {
		return
	}

	return
}

func (client *WANDSLLinkConfig1) GetModulationType() (NewModulationType string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewModulationType string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "GetModulationType", request, response); err != nil {
		return
	}

	if NewModulationType, err = soap.UnmarshalString(response.NewModulationType); err != nil {
		return
	}

	return
}

func (client *WANDSLLinkConfig1) SetDestinationAddress(NewDestinationAddress string) (err error) {

	request := &struct {
		NewDestinationAddress string
	}{}

	if request.NewDestinationAddress, err = soap.MarshalString(NewDestinationAddress); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "SetDestinationAddress", request, response); err != nil {
		return
	}

	return
}

func (client *WANDSLLinkConfig1) GetDestinationAddress() (NewDestinationAddress string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDestinationAddress string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "GetDestinationAddress", request, response); err != nil {
		return
	}

	if NewDestinationAddress, err = soap.UnmarshalString(response.NewDestinationAddress); err != nil {
		return
	}

	return
}

func (client *WANDSLLinkConfig1) SetATMEncapsulation(NewATMEncapsulation string) (err error) {

	request := &struct {
		NewATMEncapsulation string
	}{}

	if request.NewATMEncapsulation, err = soap.MarshalString(NewATMEncapsulation); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "SetATMEncapsulation", request, response); err != nil {
		return
	}

	return
}

func (client *WANDSLLinkConfig1) GetATMEncapsulation() (NewATMEncapsulation string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewATMEncapsulation string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "GetATMEncapsulation", request, response); err != nil {
		return
	}

	if NewATMEncapsulation, err = soap.UnmarshalString(response.NewATMEncapsulation); err != nil {
		return
	}

	return
}

func (client *WANDSLLinkConfig1) SetFCSPreserved(NewFCSPreserved bool) (err error) {

	request := &struct {
		NewFCSPreserved string
	}{}

	if request.NewFCSPreserved, err = soap.MarshalBoolean(NewFCSPreserved); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "SetFCSPreserved", request, response); err != nil {
		return
	}

	return
}

func (client *WANDSLLinkConfig1) GetFCSPreserved() (NewFCSPreserved bool, err error) {

	request := interface{}(nil)

	response := &struct {
		NewFCSPreserved string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANDSLLinkConfig_1, "GetFCSPreserved", request, response); err != nil {
		return
	}

	if NewFCSPreserved, err = soap.UnmarshalBoolean(response.NewFCSPreserved); err != nil {
		return
	}

	return
}

type WANEthernetLinkConfig1 struct {
	goupnp.ServiceClient
}

func NewWANEthernetLinkConfig1Clients() (clients []*WANEthernetLinkConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients(URN_WANEthernetLinkConfig_1); err != nil {
		return
	}
	clients = newWANEthernetLinkConfig1ClientsFromGenericClients(genericClients)
	return
}

func NewWANEthernetLinkConfig1ClientsByURL(loc *url.URL) ([]*WANEthernetLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, URN_WANEthernetLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANEthernetLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func NewWANEthernetLinkConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANEthernetLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANEthernetLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANEthernetLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANEthernetLinkConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANEthernetLinkConfig1 {
	clients := make([]*WANEthernetLinkConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANEthernetLinkConfig1{genericClients[i]}
	}
	return clients
}

func (client *WANEthernetLinkConfig1) GetEthernetLinkStatus() (NewEthernetLinkStatus string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewEthernetLinkStatus string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANEthernetLinkConfig_1, "GetEthernetLinkStatus", request, response); err != nil {
		return
	}

	if NewEthernetLinkStatus, err = soap.UnmarshalString(response.NewEthernetLinkStatus); err != nil {
		return
	}

	return
}

type WANIPConnection1 struct {
	goupnp.ServiceClient
}

func NewWANIPConnection1Clients() (clients []*WANIPConnection1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients(URN_WANIPConnection_1); err != nil {
		return
	}
	clients = newWANIPConnection1ClientsFromGenericClients(genericClients)
	return
}

func NewWANIPConnection1ClientsByURL(loc *url.URL) ([]*WANIPConnection1, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, URN_WANIPConnection_1)
	if err != nil {
		return nil, err
	}
	return newWANIPConnection1ClientsFromGenericClients(genericClients), nil
}

func NewWANIPConnection1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANIPConnection1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANIPConnection_1)
	if err != nil {
		return nil, err
	}
	return newWANIPConnection1ClientsFromGenericClients(genericClients), nil
}

func newWANIPConnection1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANIPConnection1 {
	clients := make([]*WANIPConnection1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANIPConnection1{genericClients[i]}
	}
	return clients
}

func (client *WANIPConnection1) SetConnectionType(NewConnectionType string) (err error) {

	request := &struct {
		NewConnectionType string
	}{}

	if request.NewConnectionType, err = soap.MarshalString(NewConnectionType); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "SetConnectionType", request, response); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) GetConnectionTypeInfo() (NewConnectionType string, NewPossibleConnectionTypes string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewConnectionType string

		NewPossibleConnectionTypes string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "GetConnectionTypeInfo", request, response); err != nil {
		return
	}

	if NewConnectionType, err = soap.UnmarshalString(response.NewConnectionType); err != nil {
		return
	}
	if NewPossibleConnectionTypes, err = soap.UnmarshalString(response.NewPossibleConnectionTypes); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) RequestConnection() (err error) {

	request := interface{}(nil)

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "RequestConnection", request, response); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) RequestTermination() (err error) {

	request := interface{}(nil)

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "RequestTermination", request, response); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) ForceTermination() (err error) {

	request := interface{}(nil)

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "ForceTermination", request, response); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) SetAutoDisconnectTime(NewAutoDisconnectTime uint32) (err error) {

	request := &struct {
		NewAutoDisconnectTime string
	}{}

	if request.NewAutoDisconnectTime, err = soap.MarshalUi4(NewAutoDisconnectTime); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "SetAutoDisconnectTime", request, response); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) SetIdleDisconnectTime(NewIdleDisconnectTime uint32) (err error) {

	request := &struct {
		NewIdleDisconnectTime string
	}{}

	if request.NewIdleDisconnectTime, err = soap.MarshalUi4(NewIdleDisconnectTime); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "SetIdleDisconnectTime", request, response); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) SetWarnDisconnectDelay(NewWarnDisconnectDelay uint32) (err error) {

	request := &struct {
		NewWarnDisconnectDelay string
	}{}

	if request.NewWarnDisconnectDelay, err = soap.MarshalUi4(NewWarnDisconnectDelay); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "SetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) GetStatusInfo() (NewConnectionStatus string, NewLastConnectionError string, NewUptime uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewConnectionStatus string

		NewLastConnectionError string

		NewUptime string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "GetStatusInfo", request, response); err != nil {
		return
	}

	if NewConnectionStatus, err = soap.UnmarshalString(response.NewConnectionStatus); err != nil {
		return
	}
	if NewLastConnectionError, err = soap.UnmarshalString(response.NewLastConnectionError); err != nil {
		return
	}
	if NewUptime, err = soap.UnmarshalUi4(response.NewUptime); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) GetAutoDisconnectTime() (NewAutoDisconnectTime uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewAutoDisconnectTime string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "GetAutoDisconnectTime", request, response); err != nil {
		return
	}

	if NewAutoDisconnectTime, err = soap.UnmarshalUi4(response.NewAutoDisconnectTime); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) GetIdleDisconnectTime() (NewIdleDisconnectTime uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewIdleDisconnectTime string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "GetIdleDisconnectTime", request, response); err != nil {
		return
	}

	if NewIdleDisconnectTime, err = soap.UnmarshalUi4(response.NewIdleDisconnectTime); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) GetWarnDisconnectDelay() (NewWarnDisconnectDelay uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewWarnDisconnectDelay string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "GetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	if NewWarnDisconnectDelay, err = soap.UnmarshalUi4(response.NewWarnDisconnectDelay); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) GetNATRSIPStatus() (NewRSIPAvailable bool, NewNATEnabled bool, err error) {

	request := interface{}(nil)

	response := &struct {
		NewRSIPAvailable string

		NewNATEnabled string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "GetNATRSIPStatus", request, response); err != nil {
		return
	}

	if NewRSIPAvailable, err = soap.UnmarshalBoolean(response.NewRSIPAvailable); err != nil {
		return
	}
	if NewNATEnabled, err = soap.UnmarshalBoolean(response.NewNATEnabled); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) GetGenericPortMappingEntry(NewPortMappingIndex uint16) (NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {

	request := &struct {
		NewPortMappingIndex string
	}{}

	if request.NewPortMappingIndex, err = soap.MarshalUi2(NewPortMappingIndex); err != nil {
		return
	}

	response := &struct {
		NewRemoteHost string

		NewExternalPort string

		NewProtocol string

		NewInternalPort string

		NewInternalClient string

		NewEnabled string

		NewPortMappingDescription string

		NewLeaseDuration string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "GetGenericPortMappingEntry", request, response); err != nil {
		return
	}

	if NewRemoteHost, err = soap.UnmarshalString(response.NewRemoteHost); err != nil {
		return
	}
	if NewExternalPort, err = soap.UnmarshalUi2(response.NewExternalPort); err != nil {
		return
	}
	if NewProtocol, err = soap.UnmarshalString(response.NewProtocol); err != nil {
		return
	}
	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) GetSpecificPortMappingEntry(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {

	request := &struct {
		NewRemoteHost string

		NewExternalPort string

		NewProtocol string
	}{}

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}

	response := &struct {
		NewInternalPort string

		NewInternalClient string

		NewEnabled string

		NewPortMappingDescription string

		NewLeaseDuration string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "GetSpecificPortMappingEntry", request, response); err != nil {
		return
	}

	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) AddPortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32) (err error) {

	request := &struct {
		NewRemoteHost string

		NewExternalPort string

		NewProtocol string

		NewInternalPort string

		NewInternalClient string

		NewEnabled string

		NewPortMappingDescription string

		NewLeaseDuration string
	}{}

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	if request.NewInternalPort, err = soap.MarshalUi2(NewInternalPort); err != nil {
		return
	}
	if request.NewInternalClient, err = soap.MarshalString(NewInternalClient); err != nil {
		return
	}
	if request.NewEnabled, err = soap.MarshalBoolean(NewEnabled); err != nil {
		return
	}
	if request.NewPortMappingDescription, err = soap.MarshalString(NewPortMappingDescription); err != nil {
		return
	}
	if request.NewLeaseDuration, err = soap.MarshalUi4(NewLeaseDuration); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "AddPortMapping", request, response); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) DeletePortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (err error) {

	request := &struct {
		NewRemoteHost string

		NewExternalPort string

		NewProtocol string
	}{}

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "DeletePortMapping", request, response); err != nil {
		return
	}

	return
}

func (client *WANIPConnection1) GetExternalIPAddress() (NewExternalIPAddress string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewExternalIPAddress string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANIPConnection_1, "GetExternalIPAddress", request, response); err != nil {
		return
	}

	if NewExternalIPAddress, err = soap.UnmarshalString(response.NewExternalIPAddress); err != nil {
		return
	}

	return
}

type WANPOTSLinkConfig1 struct {
	goupnp.ServiceClient
}

func NewWANPOTSLinkConfig1Clients() (clients []*WANPOTSLinkConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients(URN_WANPOTSLinkConfig_1); err != nil {
		return
	}
	clients = newWANPOTSLinkConfig1ClientsFromGenericClients(genericClients)
	return
}

func NewWANPOTSLinkConfig1ClientsByURL(loc *url.URL) ([]*WANPOTSLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, URN_WANPOTSLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANPOTSLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func NewWANPOTSLinkConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANPOTSLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANPOTSLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANPOTSLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANPOTSLinkConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANPOTSLinkConfig1 {
	clients := make([]*WANPOTSLinkConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANPOTSLinkConfig1{genericClients[i]}
	}
	return clients
}

func (client *WANPOTSLinkConfig1) SetISPInfo(NewISPPhoneNumber string, NewISPInfo string, NewLinkType string) (err error) {

	request := &struct {
		NewISPPhoneNumber string

		NewISPInfo string

		NewLinkType string
	}{}

	if request.NewISPPhoneNumber, err = soap.MarshalString(NewISPPhoneNumber); err != nil {
		return
	}
	if request.NewISPInfo, err = soap.MarshalString(NewISPInfo); err != nil {
		return
	}
	if request.NewLinkType, err = soap.MarshalString(NewLinkType); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPOTSLinkConfig_1, "SetISPInfo", request, response); err != nil {
		return
	}

	return
}

func (client *WANPOTSLinkConfig1) SetCallRetryInfo(NewNumberOfRetries uint32, NewDelayBetweenRetries uint32) (err error) {

	request := &struct {
		NewNumberOfRetries string

		NewDelayBetweenRetries string
	}{}

	if request.NewNumberOfRetries, err = soap.MarshalUi4(NewNumberOfRetries); err != nil {
		return
	}
	if request.NewDelayBetweenRetries, err = soap.MarshalUi4(NewDelayBetweenRetries); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPOTSLinkConfig_1, "SetCallRetryInfo", request, response); err != nil {
		return
	}

	return
}

func (client *WANPOTSLinkConfig1) GetISPInfo() (NewISPPhoneNumber string, NewISPInfo string, NewLinkType string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewISPPhoneNumber string

		NewISPInfo string

		NewLinkType string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPOTSLinkConfig_1, "GetISPInfo", request, response); err != nil {
		return
	}

	if NewISPPhoneNumber, err = soap.UnmarshalString(response.NewISPPhoneNumber); err != nil {
		return
	}
	if NewISPInfo, err = soap.UnmarshalString(response.NewISPInfo); err != nil {
		return
	}
	if NewLinkType, err = soap.UnmarshalString(response.NewLinkType); err != nil {
		return
	}

	return
}

func (client *WANPOTSLinkConfig1) GetCallRetryInfo() (NewNumberOfRetries uint32, NewDelayBetweenRetries uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewNumberOfRetries string

		NewDelayBetweenRetries string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPOTSLinkConfig_1, "GetCallRetryInfo", request, response); err != nil {
		return
	}

	if NewNumberOfRetries, err = soap.UnmarshalUi4(response.NewNumberOfRetries); err != nil {
		return
	}
	if NewDelayBetweenRetries, err = soap.UnmarshalUi4(response.NewDelayBetweenRetries); err != nil {
		return
	}

	return
}

func (client *WANPOTSLinkConfig1) GetFclass() (NewFclass string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewFclass string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPOTSLinkConfig_1, "GetFclass", request, response); err != nil {
		return
	}

	if NewFclass, err = soap.UnmarshalString(response.NewFclass); err != nil {
		return
	}

	return
}

func (client *WANPOTSLinkConfig1) GetDataModulationSupported() (NewDataModulationSupported string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDataModulationSupported string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPOTSLinkConfig_1, "GetDataModulationSupported", request, response); err != nil {
		return
	}

	if NewDataModulationSupported, err = soap.UnmarshalString(response.NewDataModulationSupported); err != nil {
		return
	}

	return
}

func (client *WANPOTSLinkConfig1) GetDataProtocol() (NewDataProtocol string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDataProtocol string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPOTSLinkConfig_1, "GetDataProtocol", request, response); err != nil {
		return
	}

	if NewDataProtocol, err = soap.UnmarshalString(response.NewDataProtocol); err != nil {
		return
	}

	return
}

func (client *WANPOTSLinkConfig1) GetDataCompression() (NewDataCompression string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewDataCompression string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPOTSLinkConfig_1, "GetDataCompression", request, response); err != nil {
		return
	}

	if NewDataCompression, err = soap.UnmarshalString(response.NewDataCompression); err != nil {
		return
	}

	return
}

func (client *WANPOTSLinkConfig1) GetPlusVTRCommandSupported() (NewPlusVTRCommandSupported bool, err error) {

	request := interface{}(nil)

	response := &struct {
		NewPlusVTRCommandSupported string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPOTSLinkConfig_1, "GetPlusVTRCommandSupported", request, response); err != nil {
		return
	}

	if NewPlusVTRCommandSupported, err = soap.UnmarshalBoolean(response.NewPlusVTRCommandSupported); err != nil {
		return
	}

	return
}

type WANPPPConnection1 struct {
	goupnp.ServiceClient
}

func NewWANPPPConnection1Clients() (clients []*WANPPPConnection1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients(URN_WANPPPConnection_1); err != nil {
		return
	}
	clients = newWANPPPConnection1ClientsFromGenericClients(genericClients)
	return
}

func NewWANPPPConnection1ClientsByURL(loc *url.URL) ([]*WANPPPConnection1, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, URN_WANPPPConnection_1)
	if err != nil {
		return nil, err
	}
	return newWANPPPConnection1ClientsFromGenericClients(genericClients), nil
}

func NewWANPPPConnection1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANPPPConnection1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANPPPConnection_1)
	if err != nil {
		return nil, err
	}
	return newWANPPPConnection1ClientsFromGenericClients(genericClients), nil
}

func newWANPPPConnection1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANPPPConnection1 {
	clients := make([]*WANPPPConnection1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANPPPConnection1{genericClients[i]}
	}
	return clients
}

func (client *WANPPPConnection1) SetConnectionType(NewConnectionType string) (err error) {

	request := &struct {
		NewConnectionType string
	}{}

	if request.NewConnectionType, err = soap.MarshalString(NewConnectionType); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "SetConnectionType", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetConnectionTypeInfo() (NewConnectionType string, NewPossibleConnectionTypes string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewConnectionType string

		NewPossibleConnectionTypes string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetConnectionTypeInfo", request, response); err != nil {
		return
	}

	if NewConnectionType, err = soap.UnmarshalString(response.NewConnectionType); err != nil {
		return
	}
	if NewPossibleConnectionTypes, err = soap.UnmarshalString(response.NewPossibleConnectionTypes); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) ConfigureConnection(NewUserName string, NewPassword string) (err error) {

	request := &struct {
		NewUserName string

		NewPassword string
	}{}

	if request.NewUserName, err = soap.MarshalString(NewUserName); err != nil {
		return
	}
	if request.NewPassword, err = soap.MarshalString(NewPassword); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "ConfigureConnection", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) RequestConnection() (err error) {

	request := interface{}(nil)

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "RequestConnection", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) RequestTermination() (err error) {

	request := interface{}(nil)

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "RequestTermination", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) ForceTermination() (err error) {

	request := interface{}(nil)

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "ForceTermination", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) SetAutoDisconnectTime(NewAutoDisconnectTime uint32) (err error) {

	request := &struct {
		NewAutoDisconnectTime string
	}{}

	if request.NewAutoDisconnectTime, err = soap.MarshalUi4(NewAutoDisconnectTime); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "SetAutoDisconnectTime", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) SetIdleDisconnectTime(NewIdleDisconnectTime uint32) (err error) {

	request := &struct {
		NewIdleDisconnectTime string
	}{}

	if request.NewIdleDisconnectTime, err = soap.MarshalUi4(NewIdleDisconnectTime); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "SetIdleDisconnectTime", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) SetWarnDisconnectDelay(NewWarnDisconnectDelay uint32) (err error) {

	request := &struct {
		NewWarnDisconnectDelay string
	}{}

	if request.NewWarnDisconnectDelay, err = soap.MarshalUi4(NewWarnDisconnectDelay); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "SetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetStatusInfo() (NewConnectionStatus string, NewLastConnectionError string, NewUptime uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewConnectionStatus string

		NewLastConnectionError string

		NewUptime string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetStatusInfo", request, response); err != nil {
		return
	}

	if NewConnectionStatus, err = soap.UnmarshalString(response.NewConnectionStatus); err != nil {
		return
	}
	if NewLastConnectionError, err = soap.UnmarshalString(response.NewLastConnectionError); err != nil {
		return
	}
	if NewUptime, err = soap.UnmarshalUi4(response.NewUptime); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetLinkLayerMaxBitRates() (NewUpstreamMaxBitRate uint32, NewDownstreamMaxBitRate uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewUpstreamMaxBitRate string

		NewDownstreamMaxBitRate string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetLinkLayerMaxBitRates", request, response); err != nil {
		return
	}

	if NewUpstreamMaxBitRate, err = soap.UnmarshalUi4(response.NewUpstreamMaxBitRate); err != nil {
		return
	}
	if NewDownstreamMaxBitRate, err = soap.UnmarshalUi4(response.NewDownstreamMaxBitRate); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetPPPEncryptionProtocol() (NewPPPEncryptionProtocol string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewPPPEncryptionProtocol string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetPPPEncryptionProtocol", request, response); err != nil {
		return
	}

	if NewPPPEncryptionProtocol, err = soap.UnmarshalString(response.NewPPPEncryptionProtocol); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetPPPCompressionProtocol() (NewPPPCompressionProtocol string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewPPPCompressionProtocol string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetPPPCompressionProtocol", request, response); err != nil {
		return
	}

	if NewPPPCompressionProtocol, err = soap.UnmarshalString(response.NewPPPCompressionProtocol); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetPPPAuthenticationProtocol() (NewPPPAuthenticationProtocol string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewPPPAuthenticationProtocol string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetPPPAuthenticationProtocol", request, response); err != nil {
		return
	}

	if NewPPPAuthenticationProtocol, err = soap.UnmarshalString(response.NewPPPAuthenticationProtocol); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetUserName() (NewUserName string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewUserName string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetUserName", request, response); err != nil {
		return
	}

	if NewUserName, err = soap.UnmarshalString(response.NewUserName); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetPassword() (NewPassword string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewPassword string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetPassword", request, response); err != nil {
		return
	}

	if NewPassword, err = soap.UnmarshalString(response.NewPassword); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetAutoDisconnectTime() (NewAutoDisconnectTime uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewAutoDisconnectTime string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetAutoDisconnectTime", request, response); err != nil {
		return
	}

	if NewAutoDisconnectTime, err = soap.UnmarshalUi4(response.NewAutoDisconnectTime); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetIdleDisconnectTime() (NewIdleDisconnectTime uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewIdleDisconnectTime string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetIdleDisconnectTime", request, response); err != nil {
		return
	}

	if NewIdleDisconnectTime, err = soap.UnmarshalUi4(response.NewIdleDisconnectTime); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetWarnDisconnectDelay() (NewWarnDisconnectDelay uint32, err error) {

	request := interface{}(nil)

	response := &struct {
		NewWarnDisconnectDelay string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	if NewWarnDisconnectDelay, err = soap.UnmarshalUi4(response.NewWarnDisconnectDelay); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetNATRSIPStatus() (NewRSIPAvailable bool, NewNATEnabled bool, err error) {

	request := interface{}(nil)

	response := &struct {
		NewRSIPAvailable string

		NewNATEnabled string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetNATRSIPStatus", request, response); err != nil {
		return
	}

	if NewRSIPAvailable, err = soap.UnmarshalBoolean(response.NewRSIPAvailable); err != nil {
		return
	}
	if NewNATEnabled, err = soap.UnmarshalBoolean(response.NewNATEnabled); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetGenericPortMappingEntry(NewPortMappingIndex uint16) (NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {

	request := &struct {
		NewPortMappingIndex string
	}{}

	if request.NewPortMappingIndex, err = soap.MarshalUi2(NewPortMappingIndex); err != nil {
		return
	}

	response := &struct {
		NewRemoteHost string

		NewExternalPort string

		NewProtocol string

		NewInternalPort string

		NewInternalClient string

		NewEnabled string

		NewPortMappingDescription string

		NewLeaseDuration string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetGenericPortMappingEntry", request, response); err != nil {
		return
	}

	if NewRemoteHost, err = soap.UnmarshalString(response.NewRemoteHost); err != nil {
		return
	}
	if NewExternalPort, err = soap.UnmarshalUi2(response.NewExternalPort); err != nil {
		return
	}
	if NewProtocol, err = soap.UnmarshalString(response.NewProtocol); err != nil {
		return
	}
	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetSpecificPortMappingEntry(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {

	request := &struct {
		NewRemoteHost string

		NewExternalPort string

		NewProtocol string
	}{}

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}

	response := &struct {
		NewInternalPort string

		NewInternalClient string

		NewEnabled string

		NewPortMappingDescription string

		NewLeaseDuration string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetSpecificPortMappingEntry", request, response); err != nil {
		return
	}

	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) AddPortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32) (err error) {

	request := &struct {
		NewRemoteHost string

		NewExternalPort string

		NewProtocol string

		NewInternalPort string

		NewInternalClient string

		NewEnabled string

		NewPortMappingDescription string

		NewLeaseDuration string
	}{}

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	if request.NewInternalPort, err = soap.MarshalUi2(NewInternalPort); err != nil {
		return
	}
	if request.NewInternalClient, err = soap.MarshalString(NewInternalClient); err != nil {
		return
	}
	if request.NewEnabled, err = soap.MarshalBoolean(NewEnabled); err != nil {
		return
	}
	if request.NewPortMappingDescription, err = soap.MarshalString(NewPortMappingDescription); err != nil {
		return
	}
	if request.NewLeaseDuration, err = soap.MarshalUi4(NewLeaseDuration); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "AddPortMapping", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) DeletePortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (err error) {

	request := &struct {
		NewRemoteHost string

		NewExternalPort string

		NewProtocol string
	}{}

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}

	response := interface{}(nil)

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "DeletePortMapping", request, response); err != nil {
		return
	}

	return
}

func (client *WANPPPConnection1) GetExternalIPAddress() (NewExternalIPAddress string, err error) {

	request := interface{}(nil)

	response := &struct {
		NewExternalIPAddress string
	}{}

	if err = client.SOAPClient.PerformAction(URN_WANPPPConnection_1, "GetExternalIPAddress", request, response); err != nil {
		return
	}

	if NewExternalIPAddress, err = soap.UnmarshalString(response.NewExternalIPAddress); err != nil {
		return
	}

	return
}
