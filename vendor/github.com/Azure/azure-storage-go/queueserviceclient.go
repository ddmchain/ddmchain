package storage

type QueueServiceClient struct {
	client Client
	auth   authentication
}

func (c *QueueServiceClient) GetServiceProperties() (*ServiceProperties, error) {
	return c.client.getServiceProperties(queueServiceName, c.auth)
}

func (c *QueueServiceClient) SetServiceProperties(props ServiceProperties) error {
	return c.client.setServiceProperties(props, queueServiceName, c.auth)
}
