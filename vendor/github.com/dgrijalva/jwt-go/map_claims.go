package jwt

import (
	"encoding/json"
	"errors"

)

type MapClaims map[string]interface{}

func (m MapClaims) VerifyAudience(cmp string, req bool) bool {
	aud, _ := m["aud"].(string)
	return verifyAud(aud, cmp, req)
}

func (m MapClaims) VerifyExpiresAt(cmp int64, req bool) bool {
	switch exp := m["exp"].(type) {
	case float64:
		return verifyExp(int64(exp), cmp, req)
	case json.Number:
		v, _ := exp.Int64()
		return verifyExp(v, cmp, req)
	}
	return req == false
}

func (m MapClaims) VerifyIssuedAt(cmp int64, req bool) bool {
	switch iat := m["iat"].(type) {
	case float64:
		return verifyIat(int64(iat), cmp, req)
	case json.Number:
		v, _ := iat.Int64()
		return verifyIat(v, cmp, req)
	}
	return req == false
}

func (m MapClaims) VerifyIssuer(cmp string, req bool) bool {
	iss, _ := m["iss"].(string)
	return verifyIss(iss, cmp, req)
}

func (m MapClaims) VerifyNotBefore(cmp int64, req bool) bool {
	switch nbf := m["nbf"].(type) {
	case float64:
		return verifyNbf(int64(nbf), cmp, req)
	case json.Number:
		v, _ := nbf.Int64()
		return verifyNbf(v, cmp, req)
	}
	return req == false
}

func (m MapClaims) Valid() error {
	vErr := new(ValidationError)
	now := TimeFunc().Unix()

	if m.VerifyExpiresAt(now, false) == false {
		vErr.Inner = errors.New("Token is expired")
		vErr.Errors |= ValidationErrorExpired
	}

	if m.VerifyIssuedAt(now, false) == false {
		vErr.Inner = errors.New("Token used before issued")
		vErr.Errors |= ValidationErrorIssuedAt
	}

	if m.VerifyNotBefore(now, false) == false {
		vErr.Inner = errors.New("Token is not valid yet")
		vErr.Errors |= ValidationErrorNotValidYet
	}

	if vErr.valid() {
		return nil
	}

	return vErr
}
