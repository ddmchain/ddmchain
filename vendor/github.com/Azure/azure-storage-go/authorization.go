
package storage

import (
	"bytes"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type authentication string

const (
	sharedKey             authentication = "sharedKey"
	sharedKeyForTable     authentication = "sharedKeyTable"
	sharedKeyLite         authentication = "sharedKeyLite"
	sharedKeyLiteForTable authentication = "sharedKeyLiteTable"

	headerAuthorization     = "Authorization"
	headerContentLength     = "Content-Length"
	headerDate              = "Date"
	headerXmsDate           = "x-ms-date"
	headerXmsVersion        = "x-ms-version"
	headerContentEncoding   = "Content-Encoding"
	headerContentLanguage   = "Content-Language"
	headerContentType       = "Content-Type"
	headerContentMD5        = "Content-MD5"
	headerIfModifiedSince   = "If-Modified-Since"
	headerIfMatch           = "If-Match"
	headerIfNoneMatch       = "If-None-Match"
	headerIfUnmodifiedSince = "If-Unmodified-Since"
	headerRange             = "Range"
)

func (c *Client) addAuthorizationHeader(verb, url string, headers map[string]string, auth authentication) (map[string]string, error) {
	authHeader, err := c.getSharedKey(verb, url, headers, auth)
	if err != nil {
		return nil, err
	}
	headers[headerAuthorization] = authHeader
	return headers, nil
}

func (c *Client) getSharedKey(verb, url string, headers map[string]string, auth authentication) (string, error) {
	canRes, err := c.buildCanonicalizedResource(url, auth)
	if err != nil {
		return "", err
	}

	canString, err := buildCanonicalizedString(verb, headers, canRes, auth)
	if err != nil {
		return "", err
	}
	return c.createAuthorizationHeader(canString, auth), nil
}

func (c *Client) buildCanonicalizedResource(uri string, auth authentication) (string, error) {
	errMsg := "buildCanonicalizedResource error: %s"
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf(errMsg, err.Error())
	}

	cr := bytes.NewBufferString("/")
	cr.WriteString(c.getCanonicalizedAccountName())

	if len(u.Path) > 0 {

		cr.WriteString(u.EscapedPath())
	}

	params, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", fmt.Errorf(errMsg, err.Error())
	}

	if auth == sharedKey {
		if len(params) > 0 {
			cr.WriteString("\n")

			keys := []string{}
			for key := range params {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			completeParams := []string{}
			for _, key := range keys {
				if len(params[key]) > 1 {
					sort.Strings(params[key])
				}

				completeParams = append(completeParams, fmt.Sprintf("%s:%s", key, strings.Join(params[key], ",")))
			}
			cr.WriteString(strings.Join(completeParams, "\n"))
		}
	} else {

		if v, ok := params["comp"]; ok {
			cr.WriteString("?comp=" + v[0])
		}
	}

	return string(cr.Bytes()), nil
}

func (c *Client) getCanonicalizedAccountName() string {

	return strings.TrimSuffix(c.accountName, "-secondary")
}

func buildCanonicalizedString(verb string, headers map[string]string, canonicalizedResource string, auth authentication) (string, error) {
	contentLength := headers[headerContentLength]
	if contentLength == "0" {
		contentLength = ""
	}
	date := headers[headerDate]
	if v, ok := headers[headerXmsDate]; ok {
		if auth == sharedKey || auth == sharedKeyLite {
			date = ""
		} else {
			date = v
		}
	}
	var canString string
	switch auth {
	case sharedKey:
		canString = strings.Join([]string{
			verb,
			headers[headerContentEncoding],
			headers[headerContentLanguage],
			contentLength,
			headers[headerContentMD5],
			headers[headerContentType],
			date,
			headers[headerIfModifiedSince],
			headers[headerIfMatch],
			headers[headerIfNoneMatch],
			headers[headerIfUnmodifiedSince],
			headers[headerRange],
			buildCanonicalizedHeader(headers),
			canonicalizedResource,
		}, "\n")
	case sharedKeyForTable:
		canString = strings.Join([]string{
			verb,
			headers[headerContentMD5],
			headers[headerContentType],
			date,
			canonicalizedResource,
		}, "\n")
	case sharedKeyLite:
		canString = strings.Join([]string{
			verb,
			headers[headerContentMD5],
			headers[headerContentType],
			date,
			buildCanonicalizedHeader(headers),
			canonicalizedResource,
		}, "\n")
	case sharedKeyLiteForTable:
		canString = strings.Join([]string{
			date,
			canonicalizedResource,
		}, "\n")
	default:
		return "", fmt.Errorf("%s authentication is not supported yet", auth)
	}
	return canString, nil
}

func buildCanonicalizedHeader(headers map[string]string) string {
	cm := make(map[string]string)

	for k, v := range headers {
		headerName := strings.TrimSpace(strings.ToLower(k))
		if strings.HasPrefix(headerName, "x-ms-") {
			cm[headerName] = v
		}
	}

	if len(cm) == 0 {
		return ""
	}

	keys := []string{}
	for key := range cm {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	ch := bytes.NewBufferString("")

	for _, key := range keys {
		ch.WriteString(key)
		ch.WriteRune(':')
		ch.WriteString(cm[key])
		ch.WriteRune('\n')
	}

	return strings.TrimSuffix(string(ch.Bytes()), "\n")
}

func (c *Client) createAuthorizationHeader(canonicalizedString string, auth authentication) string {
	signature := c.computeHmac256(canonicalizedString)
	var key string
	switch auth {
	case sharedKey, sharedKeyForTable:
		key = "SharedKey"
	case sharedKeyLite, sharedKeyLiteForTable:
		key = "SharedKeyLite"
	}
	return fmt.Sprintf("%s %s:%s", key, c.getCanonicalizedAccountName(), signature)
}
