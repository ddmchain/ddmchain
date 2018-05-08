package azure

import (
	"net/url"
)

type OAuthConfig struct {
	AuthorizeEndpoint  url.URL
	TokenEndpoint      url.URL
	DeviceCodeEndpoint url.URL
}
