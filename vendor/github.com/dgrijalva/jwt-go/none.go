package jwt

var SigningMethodNone *signingMethodNone

const UnsafeAllowNoneSignatureType unsafeNoneMagicConstant = "none signing method allowed"

var NoneSignatureTypeDisallowedError error

type signingMethodNone struct{}
type unsafeNoneMagicConstant string

func init() {
	SigningMethodNone = &signingMethodNone{}
	NoneSignatureTypeDisallowedError = NewValidationError("'none' signature type is not allowed", ValidationErrorSignatureInvalid)

	RegisterSigningMethod(SigningMethodNone.Alg(), func() SigningMethod {
		return SigningMethodNone
	})
}

func (m *signingMethodNone) Alg() string {
	return "none"
}

func (m *signingMethodNone) Verify(signingString, signature string, key interface{}) (err error) {

	if _, ok := key.(unsafeNoneMagicConstant); !ok {
		return NoneSignatureTypeDisallowedError
	}

	if signature != "" {
		return NewValidationError(
			"'none' signing method with non-empty signature",
			ValidationErrorSignatureInvalid,
		)
	}

	return nil
}

func (m *signingMethodNone) Sign(signingString string, key interface{}) (string, error) {
	if _, ok := key.(unsafeNoneMagicConstant); ok {
		return "", nil
	}
	return "", NoneSignatureTypeDisallowedError
}
