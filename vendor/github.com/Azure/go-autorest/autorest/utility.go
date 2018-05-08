package autorest

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

type EncodedAs string

const (

	EncodedAsJSON EncodedAs = "JSON"

	EncodedAsXML EncodedAs = "XML"
)

type Decoder interface {
	Decode(v interface{}) error
}

func NewDecoder(encodedAs EncodedAs, r io.Reader) Decoder {
	if encodedAs == EncodedAsJSON {
		return json.NewDecoder(r)
	} else if encodedAs == EncodedAsXML {
		return xml.NewDecoder(r)
	}
	return nil
}

func CopyAndDecode(encodedAs EncodedAs, r io.Reader, v interface{}) (bytes.Buffer, error) {
	b := bytes.Buffer{}
	return b, NewDecoder(encodedAs, io.TeeReader(r, &b)).Decode(v)
}

func TeeReadCloser(rc io.ReadCloser, w io.Writer) io.ReadCloser {
	return &teeReadCloser{rc, io.TeeReader(rc, w)}
}

type teeReadCloser struct {
	rc io.ReadCloser
	r  io.Reader
}

func (t *teeReadCloser) Read(p []byte) (int, error) {
	return t.r.Read(p)
}

func (t *teeReadCloser) Close() error {
	return t.rc.Close()
}

func containsInt(ints []int, n int) bool {
	for _, i := range ints {
		if i == n {
			return true
		}
	}
	return false
}

func escapeValueStrings(m map[string]string) map[string]string {
	for key, value := range m {
		m[key] = url.QueryEscape(value)
	}
	return m
}

func ensureValueStrings(mapOfInterface map[string]interface{}) map[string]string {
	mapOfStrings := make(map[string]string)
	for key, value := range mapOfInterface {
		mapOfStrings[key] = ensureValueString(value)
	}
	return mapOfStrings
}

func ensureValueString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func MapToValues(m map[string]interface{}) url.Values {
	v := url.Values{}
	for key, value := range m {
		x := reflect.ValueOf(value)
		if x.Kind() == reflect.Array || x.Kind() == reflect.Slice {
			for i := 0; i < x.Len(); i++ {
				v.Add(key, ensureValueString(x.Index(i)))
			}
		} else {
			v.Add(key, ensureValueString(value))
		}
	}
	return v
}

func String(v interface{}, sep ...string) string {
	if len(sep) > 0 {
		return ensureValueString(strings.Join(v.([]string), sep[0]))
	}
	return ensureValueString(v)
}

func Encode(location string, v interface{}, sep ...string) string {
	s := String(v, sep...)
	switch strings.ToLower(location) {
	case "path":
		return pathEscape(s)
	case "query":
		return queryEscape(s)
	default:
		return s
	}
}

func pathEscape(s string) string {
	return strings.Replace(url.QueryEscape(s), "+", "%20", -1)
}

func queryEscape(s string) string {
	return url.QueryEscape(s)
}

func createQuery(v url.Values) string {
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		prefix := url.QueryEscape(k) + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(prefix)
			buf.WriteString(v)
		}
	}
	return buf.String()
}
