package autorest

import (
	"fmt"
	"net/http"
)

const (

	UndefinedStatusCode = 0
)

type DetailedError struct {
	Original error

	PackageType string

	Method string

	StatusCode interface{}

	Message string

	ServiceError []byte
}

func NewError(packageType string, method string, message string, args ...interface{}) DetailedError {
	return NewErrorWithError(nil, packageType, method, nil, message, args...)
}

func NewErrorWithResponse(packageType string, method string, resp *http.Response, message string, args ...interface{}) DetailedError {
	return NewErrorWithError(nil, packageType, method, resp, message, args...)
}

func NewErrorWithError(original error, packageType string, method string, resp *http.Response, message string, args ...interface{}) DetailedError {
	if v, ok := original.(DetailedError); ok {
		return v
	}

	statusCode := UndefinedStatusCode
	if resp != nil {
		statusCode = resp.StatusCode
	}

	return DetailedError{
		Original:    original,
		PackageType: packageType,
		Method:      method,
		StatusCode:  statusCode,
		Message:     fmt.Sprintf(message, args...),
	}
}

func (e DetailedError) Error() string {
	if e.Original == nil {
		return fmt.Sprintf("%s#%s: %s: StatusCode=%d", e.PackageType, e.Method, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("%s#%s: %s: StatusCode=%d -- Original Error: %v", e.PackageType, e.Method, e.Message, e.StatusCode, e.Original)
}
