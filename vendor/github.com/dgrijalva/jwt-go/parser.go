package jwt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type Parser struct {
	ValidMethods         []string 
	UseJSONNumber        bool     
	SkipClaimsValidation bool     
}

func (p *Parser) Parse(tokenString string, keyFunc Keyfunc) (*Token, error) {
	return p.ParseWithClaims(tokenString, MapClaims{}, keyFunc)
}

func (p *Parser) ParseWithClaims(tokenString string, claims Claims, keyFunc Keyfunc) (*Token, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, NewValidationError("token contains an invalid number of segments", ValidationErrorMalformed)
	}

	var err error
	token := &Token{Raw: tokenString}

	var headerBytes []byte
	if headerBytes, err = DecodeSegment(parts[0]); err != nil {
		if strings.HasPrefix(strings.ToLower(tokenString), "bearer ") {
			return token, NewValidationError("tokenstring should not contain 'bearer '", ValidationErrorMalformed)
		}
		return token, &ValidationError{Inner: err, Errors: ValidationErrorMalformed}
	}
	if err = json.Unmarshal(headerBytes, &token.Header); err != nil {
		return token, &ValidationError{Inner: err, Errors: ValidationErrorMalformed}
	}

	var claimBytes []byte
	token.Claims = claims

	if claimBytes, err = DecodeSegment(parts[1]); err != nil {
		return token, &ValidationError{Inner: err, Errors: ValidationErrorMalformed}
	}
	dec := json.NewDecoder(bytes.NewBuffer(claimBytes))
	if p.UseJSONNumber {
		dec.UseNumber()
	}

	if c, ok := token.Claims.(MapClaims); ok {
		err = dec.Decode(&c)
	} else {
		err = dec.Decode(&claims)
	}

	if err != nil {
		return token, &ValidationError{Inner: err, Errors: ValidationErrorMalformed}
	}

	if method, ok := token.Header["alg"].(string); ok {
		if token.Method = GetSigningMethod(method); token.Method == nil {
			return token, NewValidationError("signing method (alg) is unavailable.", ValidationErrorUnverifiable)
		}
	} else {
		return token, NewValidationError("signing method (alg) is unspecified.", ValidationErrorUnverifiable)
	}

	if p.ValidMethods != nil {
		var signingMethodValid = false
		var alg = token.Method.Alg()
		for _, m := range p.ValidMethods {
			if m == alg {
				signingMethodValid = true
				break
			}
		}
		if !signingMethodValid {

			return token, NewValidationError(fmt.Sprintf("signing method %v is invalid", alg), ValidationErrorSignatureInvalid)
		}
	}

	var key interface{}
	if keyFunc == nil {

		return token, NewValidationError("no Keyfunc was provided.", ValidationErrorUnverifiable)
	}
	if key, err = keyFunc(token); err != nil {

		return token, &ValidationError{Inner: err, Errors: ValidationErrorUnverifiable}
	}

	vErr := &ValidationError{}

	if !p.SkipClaimsValidation {
		if err := token.Claims.Valid(); err != nil {

			if e, ok := err.(*ValidationError); !ok {
				vErr = &ValidationError{Inner: err, Errors: ValidationErrorClaimsInvalid}
			} else {
				vErr = e
			}
		}
	}

	token.Signature = parts[2]
	if err = token.Method.Verify(strings.Join(parts[0:2], "."), token.Signature, key); err != nil {
		vErr.Inner = err
		vErr.Errors |= ValidationErrorSignatureInvalid
	}

	if vErr.valid() {
		token.Valid = true
		return token, nil
	}

	return token, vErr
}
