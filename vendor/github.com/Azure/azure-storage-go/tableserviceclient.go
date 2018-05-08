package storage

type TableServiceClient struct {
	client Client
	auth   authentication
}

func (c *TableServiceClient) GetServiceProperties() (*ServiceProperties, error) {
	return c.client.getServiceProperties(tableServiceName, c.auth)
}

func (c *TableServiceClient) SetServiceProperties(props ServiceProperties) error {
	return c.client.setServiceProperties(props, tableServiceName, c.auth)
}
