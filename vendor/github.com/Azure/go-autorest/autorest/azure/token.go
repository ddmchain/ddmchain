package azure

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/dgrijalva/jwt-go"
)

const (
	defaultRefresh = 5 * time.Minute
	tokenBaseDate  = "1970-01-01T00:00:00Z"

	OAuthGrantTypeDeviceCode = "device_code"

	OAuthGrantTypeClientCredentials = "client_credentials"

	OAuthGrantTypeRefreshToken = "refresh_token"
)

var expirationBase time.Time

func init() {
	expirationBase, _ = time.Parse(time.RFC3339, tokenBaseDate)
}

type TokenRefreshCallback func(Token) error

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	ExpiresIn string `json:"expires_in"`
	ExpiresOn string `json:"expires_on"`
	NotBefore string `json:"not_before"`

	Resource string `json:"resource"`
	Type     string `json:"token_type"`
}

func (t Token) Expires() time.Time {
	s, err := strconv.Atoi(t.ExpiresOn)
	if err != nil {
		s = -3600
	}
	return expirationBase.Add(time.Duration(s) * time.Second).UTC()
}

func (t Token) IsExpired() bool {
	return t.WillExpireIn(0)
}

func (t Token) WillExpireIn(d time.Duration) bool {
	return !t.Expires().After(time.Now().Add(d))
}

func (t *Token) WithAuthorization() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			return (autorest.WithBearerAuthorization(t.AccessToken)(p)).Prepare(r)
		})
	}
}

type ServicePrincipalNoSecret struct {
}

func (noSecret *ServicePrincipalNoSecret) SetAuthenticationValues(spt *ServicePrincipalToken, v *url.Values) error {
	return fmt.Errorf("Manually created ServicePrincipalToken does not contain secret material to retrieve a new access token")
}

type ServicePrincipalSecret interface {
	SetAuthenticationValues(spt *ServicePrincipalToken, values *url.Values) error
}

type ServicePrincipalTokenSecret struct {
	ClientSecret string
}

func (tokenSecret *ServicePrincipalTokenSecret) SetAuthenticationValues(spt *ServicePrincipalToken, v *url.Values) error {
	v.Set("client_secret", tokenSecret.ClientSecret)
	return nil
}

type ServicePrincipalCertificateSecret struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

func (secret *ServicePrincipalCertificateSecret) SignJwt(spt *ServicePrincipalToken) (string, error) {
	hasher := sha1.New()
	_, err := hasher.Write(secret.Certificate.Raw)
	if err != nil {
		return "", err
	}

	thumbprint := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	jti := make([]byte, 20)
	_, err = rand.Read(jti)
	if err != nil {
		return "", err
	}

	token := jwt.New(jwt.SigningMethodRS256)
	token.Header["x5t"] = thumbprint
	token.Claims = jwt.MapClaims{
		"aud": spt.oauthConfig.TokenEndpoint.String(),
		"iss": spt.clientID,
		"sub": spt.clientID,
		"jti": base64.URLEncoding.EncodeToString(jti),
		"nbf": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	signedString, err := token.SignedString(secret.PrivateKey)
	return signedString, err
}

func (secret *ServicePrincipalCertificateSecret) SetAuthenticationValues(spt *ServicePrincipalToken, v *url.Values) error {
	jwt, err := secret.SignJwt(spt)
	if err != nil {
		return err
	}

	v.Set("client_assertion", jwt)
	v.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	return nil
}

type ServicePrincipalToken struct {
	Token

	secret        ServicePrincipalSecret
	oauthConfig   OAuthConfig
	clientID      string
	resource      string
	autoRefresh   bool
	refreshWithin time.Duration
	sender        autorest.Sender

	refreshCallbacks []TokenRefreshCallback
}

func NewServicePrincipalTokenWithSecret(oauthConfig OAuthConfig, id string, resource string, secret ServicePrincipalSecret, callbacks ...TokenRefreshCallback) (*ServicePrincipalToken, error) {
	spt := &ServicePrincipalToken{
		oauthConfig:      oauthConfig,
		secret:           secret,
		clientID:         id,
		resource:         resource,
		autoRefresh:      true,
		refreshWithin:    defaultRefresh,
		sender:           &http.Client{},
		refreshCallbacks: callbacks,
	}
	return spt, nil
}

func NewServicePrincipalTokenFromManualToken(oauthConfig OAuthConfig, clientID string, resource string, token Token, callbacks ...TokenRefreshCallback) (*ServicePrincipalToken, error) {
	spt, err := NewServicePrincipalTokenWithSecret(
		oauthConfig,
		clientID,
		resource,
		&ServicePrincipalNoSecret{},
		callbacks...)
	if err != nil {
		return nil, err
	}

	spt.Token = token

	return spt, nil
}

func NewServicePrincipalToken(oauthConfig OAuthConfig, clientID string, secret string, resource string, callbacks ...TokenRefreshCallback) (*ServicePrincipalToken, error) {
	return NewServicePrincipalTokenWithSecret(
		oauthConfig,
		clientID,
		resource,
		&ServicePrincipalTokenSecret{
			ClientSecret: secret,
		},
		callbacks...,
	)
}

func NewServicePrincipalTokenFromCertificate(oauthConfig OAuthConfig, clientID string, certificate *x509.Certificate, privateKey *rsa.PrivateKey, resource string, callbacks ...TokenRefreshCallback) (*ServicePrincipalToken, error) {
	return NewServicePrincipalTokenWithSecret(
		oauthConfig,
		clientID,
		resource,
		&ServicePrincipalCertificateSecret{
			PrivateKey:  privateKey,
			Certificate: certificate,
		},
		callbacks...,
	)
}

func (spt *ServicePrincipalToken) EnsureFresh() error {
	if spt.WillExpireIn(spt.refreshWithin) {
		return spt.Refresh()
	}
	return nil
}

func (spt *ServicePrincipalToken) InvokeRefreshCallbacks(token Token) error {
	if spt.refreshCallbacks != nil {
		for _, callback := range spt.refreshCallbacks {
			err := callback(spt.Token)
			if err != nil {
				return autorest.NewErrorWithError(err,
					"azure.ServicePrincipalToken", "InvokeRefreshCallbacks", nil, "A TokenRefreshCallback handler returned an error")
			}
		}
	}
	return nil
}

func (spt *ServicePrincipalToken) Refresh() error {
	return spt.refreshInternal(spt.resource)
}

func (spt *ServicePrincipalToken) RefreshExchange(resource string) error {
	return spt.refreshInternal(resource)
}

func (spt *ServicePrincipalToken) refreshInternal(resource string) error {
	v := url.Values{}
	v.Set("client_id", spt.clientID)
	v.Set("resource", resource)

	if spt.RefreshToken != "" {
		v.Set("grant_type", OAuthGrantTypeRefreshToken)
		v.Set("refresh_token", spt.RefreshToken)
	} else {
		v.Set("grant_type", OAuthGrantTypeClientCredentials)
		err := spt.secret.SetAuthenticationValues(spt, &v)
		if err != nil {
			return err
		}
	}

	req, _ := autorest.Prepare(&http.Request{},
		autorest.AsPost(),
		autorest.AsFormURLEncoded(),
		autorest.WithBaseURL(spt.oauthConfig.TokenEndpoint.String()),
		autorest.WithFormData(v))

	resp, err := autorest.SendWithSender(spt.sender, req)
	if err != nil {
		return autorest.NewErrorWithError(err,
			"azure.ServicePrincipalToken", "Refresh", resp, "Failure sending request for Service Principal %s",
			spt.clientID)
	}

	var newToken Token
	err = autorest.Respond(resp,
		autorest.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&newToken),
		autorest.ByClosing())
	if err != nil {
		return autorest.NewErrorWithError(err,
			"azure.ServicePrincipalToken", "Refresh", resp, "Failure handling response to Service Principal %s request",
			spt.clientID)
	}

	spt.Token = newToken

	err = spt.InvokeRefreshCallbacks(newToken)
	if err != nil {

		return err
	}

	return nil
}

func (spt *ServicePrincipalToken) SetAutoRefresh(autoRefresh bool) {
	spt.autoRefresh = autoRefresh
}

func (spt *ServicePrincipalToken) SetRefreshWithin(d time.Duration) {
	spt.refreshWithin = d
	return
}

func (spt *ServicePrincipalToken) SetSender(s autorest.Sender) {
	spt.sender = s
}

func (spt *ServicePrincipalToken) WithAuthorization() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			if spt.autoRefresh {
				err := spt.EnsureFresh()
				if err != nil {
					return r, autorest.NewErrorWithError(err,
						"azure.ServicePrincipalToken", "WithAuthorization", nil, "Failed to refresh Service Principal Token for request to %s",
						r.URL)
				}
			}
			return (autorest.WithBearerAuthorization(spt.AccessToken)(p)).Prepare(r)
		})
	}
}
