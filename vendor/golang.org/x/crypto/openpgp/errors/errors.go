
package errors 

import (
	"strconv"
)

type StructuralError string

func (s StructuralError) Error() string {
	return "openpgp: invalid data: " + string(s)
}

type UnsupportedError string

func (s UnsupportedError) Error() string {
	return "openpgp: unsupported feature: " + string(s)
}

type InvalidArgumentError string

func (i InvalidArgumentError) Error() string {
	return "openpgp: invalid argument: " + string(i)
}

type SignatureError string

func (b SignatureError) Error() string {
	return "openpgp: invalid signature: " + string(b)
}

type keyIncorrectError int

func (ki keyIncorrectError) Error() string {
	return "openpgp: incorrect key"
}

var ErrKeyIncorrect error = keyIncorrectError(0)

type unknownIssuerError int

func (unknownIssuerError) Error() string {
	return "openpgp: signature made by unknown entity"
}

var ErrUnknownIssuer error = unknownIssuerError(0)

type keyRevokedError int

func (keyRevokedError) Error() string {
	return "openpgp: signature made by revoked key"
}

var ErrKeyRevoked error = keyRevokedError(0)

type UnknownPacketTypeError uint8

func (upte UnknownPacketTypeError) Error() string {
	return "openpgp: unknown packet type: " + strconv.Itoa(int(upte))
}
