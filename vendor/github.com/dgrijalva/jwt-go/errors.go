package jwt

import (
	"errors"
)

var (
	ErrInvalidKey      = errors.New("key is invalid")
	ErrInvalidKeyType  = errors.New("key is of invalid type")
	ErrHashUnavailable = errors.New("the requested hash function is unavailable")
)

const (
	ValidationErrorMalformed        uint32 = 1 << iota 
	ValidationErrorUnverifiable                        
	ValidationErrorSignatureInvalid                    

	ValidationErrorAudience      
	ValidationErrorExpired       
	ValidationErrorIssuedAt      
	ValidationErrorIssuer        
	ValidationErrorNotValidYet   
	ValidationErrorId            
	ValidationErrorClaimsInvalid 
)

func NewValidationError(errorText string, errorFlags uint32) *ValidationError {
	return &ValidationError{
		text:   errorText,
		Errors: errorFlags,
	}
}

type ValidationError struct {
	Inner  error  
	Errors uint32 
	text   string 
}

func (e ValidationError) Error() string {
	if e.Inner != nil {
		return e.Inner.Error()
	} else if e.text != "" {
		return e.text
	} else {
		return "token is invalid"
	}
}

func (e *ValidationError) valid() bool {
	return e.Errors == 0
}
