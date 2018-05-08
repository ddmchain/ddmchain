package jwt

import (
	"sync"
)

var signingMethods = map[string]func() SigningMethod{}
var signingMethodLock = new(sync.RWMutex)

type SigningMethod interface {
	Verify(signingString, signature string, key interface{}) error 
	Sign(signingString string, key interface{}) (string, error)    
	Alg() string                                                   
}

func RegisterSigningMethod(alg string, f func() SigningMethod) {
	signingMethodLock.Lock()
	defer signingMethodLock.Unlock()

	signingMethods[alg] = f
}

func GetSigningMethod(alg string) (method SigningMethod) {
	signingMethodLock.RLock()
	defer signingMethodLock.RUnlock()

	if methodF, ok := signingMethods[alg]; ok {
		method = methodF()
	}
	return
}
