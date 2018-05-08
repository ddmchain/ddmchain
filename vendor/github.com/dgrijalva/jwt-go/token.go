package jwt

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

var TimeFunc = time.Now

type Keyfunc func(*Token) (interface{}, error)

type Token struct {
	Raw       string                 
	Method    SigningMethod          
	Header    map[string]interface{} 
	Claims    Claims                 
	Signature string                 
	Valid     bool                   
}

func New(method SigningMethod) *Token {
	return NewWithClaims(method, MapClaims{})
}

func NewWithClaims(method SigningMethod, claims Claims) *Token {
	return &Token{
		Header: map[string]interface{}{
			"typ": "JWT",
			"alg": method.Alg(),
		},
		Claims: claims,
		Method: method,
	}
}

func (t *Token) SignedString(key interface{}) (string, error) {
	var sig, sstr string
	var err error
	if sstr, err = t.SigningString(); err != nil {
		return "", err
	}
	if sig, err = t.Method.Sign(sstr, key); err != nil {
		return "", err
	}
	return strings.Join([]string{sstr, sig}, "."), nil
}

func (t *Token) SigningString() (string, error) {
	var err error
	parts := make([]string, 2)
	for i, _ := range parts {
		var jsonValue []byte
		if i == 0 {
			if jsonValue, err = json.Marshal(t.Header); err != nil {
				return "", err
			}
		} else {
			if jsonValue, err = json.Marshal(t.Claims); err != nil {
				return "", err
			}
		}

		parts[i] = EncodeSegment(jsonValue)
	}
	return strings.Join(parts, "."), nil
}

func Parse(tokenString string, keyFunc Keyfunc) (*Token, error) {
	return new(Parser).Parse(tokenString, keyFunc)
}

func ParseWithClaims(tokenString string, claims Claims, keyFunc Keyfunc) (*Token, error) {
	return new(Parser).ParseWithClaims(tokenString, claims, keyFunc)
}

func EncodeSegment(seg []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(seg), "=")
}

func DecodeSegment(seg string) ([]byte, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}

	return base64.URLEncoding.DecodeString(seg)
}
