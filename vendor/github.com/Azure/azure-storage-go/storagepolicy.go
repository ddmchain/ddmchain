package storage

import (
	"strings"
	"time"
)

type AccessPolicyDetailsXML struct {
	StartTime  time.Time `xml:"Start"`
	ExpiryTime time.Time `xml:"Expiry"`
	Permission string    `xml:"Permission"`
}

type SignedIdentifier struct {
	ID           string                 `xml:"Id"`
	AccessPolicy AccessPolicyDetailsXML `xml:"AccessPolicy"`
}

type SignedIdentifiers struct {
	SignedIdentifiers []SignedIdentifier `xml:"SignedIdentifier"`
}

type AccessPolicy struct {
	SignedIdentifiersList SignedIdentifiers `xml:"SignedIdentifiers"`
}

func convertAccessPolicyToXMLStructs(id string, startTime time.Time, expiryTime time.Time, permissions string) SignedIdentifier {
	return SignedIdentifier{
		ID: id,
		AccessPolicy: AccessPolicyDetailsXML{
			StartTime:  startTime.UTC().Round(time.Second),
			ExpiryTime: expiryTime.UTC().Round(time.Second),
			Permission: permissions,
		},
	}
}

func updatePermissions(permissions, permission string) bool {
	return strings.Contains(permissions, permission)
}
