
package openpgp

import (
	"crypto/rsa"
	"io"
	"time"

	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/packet"
)

var PublicKeyType = "PGP PUBLIC KEY BLOCK"

var PrivateKeyType = "PGP PRIVATE KEY BLOCK"

type Entity struct {
	PrimaryKey  *packet.PublicKey
	PrivateKey  *packet.PrivateKey
	Identities  map[string]*Identity 
	Revocations []*packet.Signature
	Subkeys     []Subkey
}

type Identity struct {
	Name          string 
	UserId        *packet.UserId
	SelfSignature *packet.Signature
	Signatures    []*packet.Signature
}

type Subkey struct {
	PublicKey  *packet.PublicKey
	PrivateKey *packet.PrivateKey
	Sig        *packet.Signature
}

type Key struct {
	Entity        *Entity
	PublicKey     *packet.PublicKey
	PrivateKey    *packet.PrivateKey
	SelfSignature *packet.Signature
}

type KeyRing interface {

	KeysById(id uint64) []Key

	KeysByIdUsage(id uint64, requiredUsage byte) []Key

	DecryptionKeys() []Key
}

func (e *Entity) primaryIdentity() *Identity {
	var firstIdentity *Identity
	for _, ident := range e.Identities {
		if firstIdentity == nil {
			firstIdentity = ident
		}
		if ident.SelfSignature.IsPrimaryId != nil && *ident.SelfSignature.IsPrimaryId {
			return ident
		}
	}
	return firstIdentity
}

func (e *Entity) encryptionKey(now time.Time) (Key, bool) {
	candidateSubkey := -1

	var maxTime time.Time
	for i, subkey := range e.Subkeys {
		if subkey.Sig.FlagsValid &&
			subkey.Sig.FlagEncryptCommunications &&
			subkey.PublicKey.PubKeyAlgo.CanEncrypt() &&
			!subkey.Sig.KeyExpired(now) &&
			(maxTime.IsZero() || subkey.Sig.CreationTime.After(maxTime)) {
			candidateSubkey = i
			maxTime = subkey.Sig.CreationTime
		}
	}

	if candidateSubkey != -1 {
		subkey := e.Subkeys[candidateSubkey]
		return Key{e, subkey.PublicKey, subkey.PrivateKey, subkey.Sig}, true
	}

	i := e.primaryIdentity()
	if !i.SelfSignature.FlagsValid || i.SelfSignature.FlagEncryptCommunications &&
		e.PrimaryKey.PubKeyAlgo.CanEncrypt() &&
		!i.SelfSignature.KeyExpired(now) {
		return Key{e, e.PrimaryKey, e.PrivateKey, i.SelfSignature}, true
	}

	return Key{}, false
}

func (e *Entity) signingKey(now time.Time) (Key, bool) {
	candidateSubkey := -1

	for i, subkey := range e.Subkeys {
		if subkey.Sig.FlagsValid &&
			subkey.Sig.FlagSign &&
			subkey.PublicKey.PubKeyAlgo.CanSign() &&
			!subkey.Sig.KeyExpired(now) {
			candidateSubkey = i
			break
		}
	}

	if candidateSubkey != -1 {
		subkey := e.Subkeys[candidateSubkey]
		return Key{e, subkey.PublicKey, subkey.PrivateKey, subkey.Sig}, true
	}

	i := e.primaryIdentity()
	if !i.SelfSignature.FlagsValid || i.SelfSignature.FlagSign &&
		!i.SelfSignature.KeyExpired(now) {
		return Key{e, e.PrimaryKey, e.PrivateKey, i.SelfSignature}, true
	}

	return Key{}, false
}

type EntityList []*Entity

func (el EntityList) KeysById(id uint64) (keys []Key) {
	for _, e := range el {
		if e.PrimaryKey.KeyId == id {
			var selfSig *packet.Signature
			for _, ident := range e.Identities {
				if selfSig == nil {
					selfSig = ident.SelfSignature
				} else if ident.SelfSignature.IsPrimaryId != nil && *ident.SelfSignature.IsPrimaryId {
					selfSig = ident.SelfSignature
					break
				}
			}
			keys = append(keys, Key{e, e.PrimaryKey, e.PrivateKey, selfSig})
		}

		for _, subKey := range e.Subkeys {
			if subKey.PublicKey.KeyId == id {
				keys = append(keys, Key{e, subKey.PublicKey, subKey.PrivateKey, subKey.Sig})
			}
		}
	}
	return
}

func (el EntityList) KeysByIdUsage(id uint64, requiredUsage byte) (keys []Key) {
	for _, key := range el.KeysById(id) {
		if len(key.Entity.Revocations) > 0 {
			continue
		}

		if key.SelfSignature.RevocationReason != nil {
			continue
		}

		if key.SelfSignature.FlagsValid && requiredUsage != 0 {
			var usage byte
			if key.SelfSignature.FlagCertify {
				usage |= packet.KeyFlagCertify
			}
			if key.SelfSignature.FlagSign {
				usage |= packet.KeyFlagSign
			}
			if key.SelfSignature.FlagEncryptCommunications {
				usage |= packet.KeyFlagEncryptCommunications
			}
			if key.SelfSignature.FlagEncryptStorage {
				usage |= packet.KeyFlagEncryptStorage
			}
			if usage&requiredUsage != requiredUsage {
				continue
			}
		}

		keys = append(keys, key)
	}
	return
}

func (el EntityList) DecryptionKeys() (keys []Key) {
	for _, e := range el {
		for _, subKey := range e.Subkeys {
			if subKey.PrivateKey != nil && (!subKey.Sig.FlagsValid || subKey.Sig.FlagEncryptStorage || subKey.Sig.FlagEncryptCommunications) {
				keys = append(keys, Key{e, subKey.PublicKey, subKey.PrivateKey, subKey.Sig})
			}
		}
	}
	return
}

func ReadArmoredKeyRing(r io.Reader) (EntityList, error) {
	block, err := armor.Decode(r)
	if err == io.EOF {
		return nil, errors.InvalidArgumentError("no armored data found")
	}
	if err != nil {
		return nil, err
	}
	if block.Type != PublicKeyType && block.Type != PrivateKeyType {
		return nil, errors.InvalidArgumentError("expected public or private key block, got: " + block.Type)
	}

	return ReadKeyRing(block.Body)
}

func ReadKeyRing(r io.Reader) (el EntityList, err error) {
	packets := packet.NewReader(r)
	var lastUnsupportedError error

	for {
		var e *Entity
		e, err = ReadEntity(packets)
		if err != nil {

			if _, ok := err.(errors.UnsupportedError); ok {
				lastUnsupportedError = err
				err = readToNextPublicKey(packets)
			} else if _, ok := err.(errors.StructuralError); ok {

				lastUnsupportedError = err
				err = readToNextPublicKey(packets)
			}
			if err == io.EOF {
				err = nil
				break
			}
			if err != nil {
				el = nil
				break
			}
		} else {
			el = append(el, e)
		}
	}

	if len(el) == 0 && err == nil {
		err = lastUnsupportedError
	}
	return
}

func readToNextPublicKey(packets *packet.Reader) (err error) {
	var p packet.Packet
	for {
		p, err = packets.Next()
		if err == io.EOF {
			return
		} else if err != nil {
			if _, ok := err.(errors.UnsupportedError); ok {
				err = nil
				continue
			}
			return
		}

		if pk, ok := p.(*packet.PublicKey); ok && !pk.IsSubkey {
			packets.Unread(p)
			return
		}
	}
}

func ReadEntity(packets *packet.Reader) (*Entity, error) {
	e := new(Entity)
	e.Identities = make(map[string]*Identity)

	p, err := packets.Next()
	if err != nil {
		return nil, err
	}

	var ok bool
	if e.PrimaryKey, ok = p.(*packet.PublicKey); !ok {
		if e.PrivateKey, ok = p.(*packet.PrivateKey); !ok {
			packets.Unread(p)
			return nil, errors.StructuralError("first packet was not a public/private key")
		} else {
			e.PrimaryKey = &e.PrivateKey.PublicKey
		}
	}

	if !e.PrimaryKey.PubKeyAlgo.CanSign() {
		return nil, errors.StructuralError("primary key cannot be used for signatures")
	}

	var current *Identity
	var revocations []*packet.Signature
EachPacket:
	for {
		p, err := packets.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		switch pkt := p.(type) {
		case *packet.UserId:
			current = new(Identity)
			current.Name = pkt.Id
			current.UserId = pkt
			e.Identities[pkt.Id] = current

			for {
				p, err = packets.Next()
				if err == io.EOF {
					return nil, io.ErrUnexpectedEOF
				} else if err != nil {
					return nil, err
				}

				sig, ok := p.(*packet.Signature)
				if !ok {
					return nil, errors.StructuralError("user ID packet not followed by self-signature")
				}

				if (sig.SigType == packet.SigTypePositiveCert || sig.SigType == packet.SigTypeGenericCert) && sig.IssuerKeyId != nil && *sig.IssuerKeyId == e.PrimaryKey.KeyId {
					if err = e.PrimaryKey.VerifyUserIdSignature(pkt.Id, e.PrimaryKey, sig); err != nil {
						return nil, errors.StructuralError("user ID self-signature invalid: " + err.Error())
					}
					current.SelfSignature = sig
					break
				}
				current.Signatures = append(current.Signatures, sig)
			}
		case *packet.Signature:
			if pkt.SigType == packet.SigTypeKeyRevocation {
				revocations = append(revocations, pkt)
			} else if pkt.SigType == packet.SigTypeDirectSignature {

			} else if current == nil {
				return nil, errors.StructuralError("signature packet found before user id packet")
			} else {
				current.Signatures = append(current.Signatures, pkt)
			}
		case *packet.PrivateKey:
			if pkt.IsSubkey == false {
				packets.Unread(p)
				break EachPacket
			}
			err = addSubkey(e, packets, &pkt.PublicKey, pkt)
			if err != nil {
				return nil, err
			}
		case *packet.PublicKey:
			if pkt.IsSubkey == false {
				packets.Unread(p)
				break EachPacket
			}
			err = addSubkey(e, packets, pkt, nil)
			if err != nil {
				return nil, err
			}
		default:

		}
	}

	if len(e.Identities) == 0 {
		return nil, errors.StructuralError("entity without any identities")
	}

	for _, revocation := range revocations {
		err = e.PrimaryKey.VerifyRevocationSignature(revocation)
		if err == nil {
			e.Revocations = append(e.Revocations, revocation)
		} else {

			return nil, errors.StructuralError("revocation signature signed by alternate key")
		}
	}

	return e, nil
}

func addSubkey(e *Entity, packets *packet.Reader, pub *packet.PublicKey, priv *packet.PrivateKey) error {
	var subKey Subkey
	subKey.PublicKey = pub
	subKey.PrivateKey = priv
	p, err := packets.Next()
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	if err != nil {
		return errors.StructuralError("subkey signature invalid: " + err.Error())
	}
	var ok bool
	subKey.Sig, ok = p.(*packet.Signature)
	if !ok {
		return errors.StructuralError("subkey packet not followed by signature")
	}
	if subKey.Sig.SigType != packet.SigTypeSubkeyBinding && subKey.Sig.SigType != packet.SigTypeSubkeyRevocation {
		return errors.StructuralError("subkey signature with wrong type")
	}
	err = e.PrimaryKey.VerifyKeySignature(subKey.PublicKey, subKey.Sig)
	if err != nil {
		return errors.StructuralError("subkey signature invalid: " + err.Error())
	}
	e.Subkeys = append(e.Subkeys, subKey)
	return nil
}

const defaultRSAKeyBits = 2048

func NewEntity(name, comment, email string, config *packet.Config) (*Entity, error) {
	currentTime := config.Now()

	bits := defaultRSAKeyBits
	if config != nil && config.RSABits != 0 {
		bits = config.RSABits
	}

	uid := packet.NewUserId(name, comment, email)
	if uid == nil {
		return nil, errors.InvalidArgumentError("user id field contained invalid characters")
	}
	signingPriv, err := rsa.GenerateKey(config.Random(), bits)
	if err != nil {
		return nil, err
	}
	encryptingPriv, err := rsa.GenerateKey(config.Random(), bits)
	if err != nil {
		return nil, err
	}

	e := &Entity{
		PrimaryKey: packet.NewRSAPublicKey(currentTime, &signingPriv.PublicKey),
		PrivateKey: packet.NewRSAPrivateKey(currentTime, signingPriv),
		Identities: make(map[string]*Identity),
	}
	isPrimaryId := true
	e.Identities[uid.Id] = &Identity{
		Name:   uid.Name,
		UserId: uid,
		SelfSignature: &packet.Signature{
			CreationTime: currentTime,
			SigType:      packet.SigTypePositiveCert,
			PubKeyAlgo:   packet.PubKeyAlgoRSA,
			Hash:         config.Hash(),
			IsPrimaryId:  &isPrimaryId,
			FlagsValid:   true,
			FlagSign:     true,
			FlagCertify:  true,
			IssuerKeyId:  &e.PrimaryKey.KeyId,
		},
	}

	if config != nil && config.DefaultHash != 0 {
		e.Identities[uid.Id].SelfSignature.PreferredHash = []uint8{hashToHashId(config.DefaultHash)}
	}

	e.Subkeys = make([]Subkey, 1)
	e.Subkeys[0] = Subkey{
		PublicKey:  packet.NewRSAPublicKey(currentTime, &encryptingPriv.PublicKey),
		PrivateKey: packet.NewRSAPrivateKey(currentTime, encryptingPriv),
		Sig: &packet.Signature{
			CreationTime:              currentTime,
			SigType:                   packet.SigTypeSubkeyBinding,
			PubKeyAlgo:                packet.PubKeyAlgoRSA,
			Hash:                      config.Hash(),
			FlagsValid:                true,
			FlagEncryptStorage:        true,
			FlagEncryptCommunications: true,
			IssuerKeyId:               &e.PrimaryKey.KeyId,
		},
	}
	e.Subkeys[0].PublicKey.IsSubkey = true
	e.Subkeys[0].PrivateKey.IsSubkey = true

	return e, nil
}

func (e *Entity) SerializePrivate(w io.Writer, config *packet.Config) (err error) {
	err = e.PrivateKey.Serialize(w)
	if err != nil {
		return
	}
	for _, ident := range e.Identities {
		err = ident.UserId.Serialize(w)
		if err != nil {
			return
		}
		err = ident.SelfSignature.SignUserId(ident.UserId.Id, e.PrimaryKey, e.PrivateKey, config)
		if err != nil {
			return
		}
		err = ident.SelfSignature.Serialize(w)
		if err != nil {
			return
		}
	}
	for _, subkey := range e.Subkeys {
		err = subkey.PrivateKey.Serialize(w)
		if err != nil {
			return
		}
		err = subkey.Sig.SignKey(subkey.PublicKey, e.PrivateKey, config)
		if err != nil {
			return
		}
		err = subkey.Sig.Serialize(w)
		if err != nil {
			return
		}
	}
	return nil
}

func (e *Entity) Serialize(w io.Writer) error {
	err := e.PrimaryKey.Serialize(w)
	if err != nil {
		return err
	}
	for _, ident := range e.Identities {
		err = ident.UserId.Serialize(w)
		if err != nil {
			return err
		}
		err = ident.SelfSignature.Serialize(w)
		if err != nil {
			return err
		}
		for _, sig := range ident.Signatures {
			err = sig.Serialize(w)
			if err != nil {
				return err
			}
		}
	}
	for _, subkey := range e.Subkeys {
		err = subkey.PublicKey.Serialize(w)
		if err != nil {
			return err
		}
		err = subkey.Sig.Serialize(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Entity) SignIdentity(identity string, signer *Entity, config *packet.Config) error {
	if signer.PrivateKey == nil {
		return errors.InvalidArgumentError("signing Entity must have a private key")
	}
	if signer.PrivateKey.Encrypted {
		return errors.InvalidArgumentError("signing Entity's private key must be decrypted")
	}
	ident, ok := e.Identities[identity]
	if !ok {
		return errors.InvalidArgumentError("given identity string not found in Entity")
	}

	sig := &packet.Signature{
		SigType:      packet.SigTypeGenericCert,
		PubKeyAlgo:   signer.PrivateKey.PubKeyAlgo,
		Hash:         config.Hash(),
		CreationTime: config.Now(),
		IssuerKeyId:  &signer.PrivateKey.KeyId,
	}
	if err := sig.SignUserId(identity, e.PrimaryKey, signer.PrivateKey, config); err != nil {
		return err
	}
	ident.Signatures = append(ident.Signatures, sig)
	return nil
}
