
package openpgp 

import (
	"crypto"
	_ "crypto/sha256"
	"hash"
	"io"
	"strconv"

	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/packet"
)

var SignatureType = "PGP SIGNATURE"

func readArmored(r io.Reader, expectedType string) (body io.Reader, err error) {
	block, err := armor.Decode(r)
	if err != nil {
		return
	}

	if block.Type != expectedType {
		return nil, errors.InvalidArgumentError("expected '" + expectedType + "', got: " + block.Type)
	}

	return block.Body, nil
}

type MessageDetails struct {
	IsEncrypted              bool                
	EncryptedToKeyIds        []uint64            
	IsSymmetricallyEncrypted bool                
	DecryptedWith            Key                 
	IsSigned                 bool                
	SignedByKeyId            uint64              
	SignedBy                 *Key                
	LiteralData              *packet.LiteralData 
	UnverifiedBody           io.Reader           

	SignatureError error               
	Signature      *packet.Signature   
	SignatureV3    *packet.SignatureV3 

	decrypted io.ReadCloser
}

type PromptFunction func(keys []Key, symmetric bool) ([]byte, error)

type keyEnvelopePair struct {
	key          Key
	encryptedKey *packet.EncryptedKey
}

func ReadMessage(r io.Reader, keyring KeyRing, prompt PromptFunction, config *packet.Config) (md *MessageDetails, err error) {
	var p packet.Packet

	var symKeys []*packet.SymmetricKeyEncrypted
	var pubKeys []keyEnvelopePair
	var se *packet.SymmetricallyEncrypted

	packets := packet.NewReader(r)
	md = new(MessageDetails)
	md.IsEncrypted = true

ParsePackets:
	for {
		p, err = packets.Next()
		if err != nil {
			return nil, err
		}
		switch p := p.(type) {
		case *packet.SymmetricKeyEncrypted:

			md.IsSymmetricallyEncrypted = true
			symKeys = append(symKeys, p)
		case *packet.EncryptedKey:

			md.EncryptedToKeyIds = append(md.EncryptedToKeyIds, p.KeyId)
			switch p.Algo {
			case packet.PubKeyAlgoRSA, packet.PubKeyAlgoRSAEncryptOnly, packet.PubKeyAlgoElGamal:
				break
			default:
				continue
			}
			var keys []Key
			if p.KeyId == 0 {
				keys = keyring.DecryptionKeys()
			} else {
				keys = keyring.KeysById(p.KeyId)
			}
			for _, k := range keys {
				pubKeys = append(pubKeys, keyEnvelopePair{k, p})
			}
		case *packet.SymmetricallyEncrypted:
			se = p
			break ParsePackets
		case *packet.Compressed, *packet.LiteralData, *packet.OnePassSignature:

			if len(symKeys) != 0 || len(pubKeys) != 0 {
				return nil, errors.StructuralError("key material not followed by encrypted message")
			}
			packets.Unread(p)
			return readSignedMessage(packets, nil, keyring)
		}
	}

	var candidates []Key
	var decrypted io.ReadCloser

FindKey:
	for {

		candidates = candidates[:0]
		candidateFingerprints := make(map[string]bool)

		for _, pk := range pubKeys {
			if pk.key.PrivateKey == nil {
				continue
			}
			if !pk.key.PrivateKey.Encrypted {
				if len(pk.encryptedKey.Key) == 0 {
					pk.encryptedKey.Decrypt(pk.key.PrivateKey, config)
				}
				if len(pk.encryptedKey.Key) == 0 {
					continue
				}
				decrypted, err = se.Decrypt(pk.encryptedKey.CipherFunc, pk.encryptedKey.Key)
				if err != nil && err != errors.ErrKeyIncorrect {
					return nil, err
				}
				if decrypted != nil {
					md.DecryptedWith = pk.key
					break FindKey
				}
			} else {
				fpr := string(pk.key.PublicKey.Fingerprint[:])
				if v := candidateFingerprints[fpr]; v {
					continue
				}
				candidates = append(candidates, pk.key)
				candidateFingerprints[fpr] = true
			}
		}

		if len(candidates) == 0 && len(symKeys) == 0 {
			return nil, errors.ErrKeyIncorrect
		}

		if prompt == nil {
			return nil, errors.ErrKeyIncorrect
		}

		passphrase, err := prompt(candidates, len(symKeys) != 0)
		if err != nil {
			return nil, err
		}

		if len(symKeys) != 0 && passphrase != nil {
			for _, s := range symKeys {
				key, cipherFunc, err := s.Decrypt(passphrase)
				if err == nil {
					decrypted, err = se.Decrypt(cipherFunc, key)
					if err != nil && err != errors.ErrKeyIncorrect {
						return nil, err
					}
					if decrypted != nil {
						break FindKey
					}
				}

			}
		}
	}

	md.decrypted = decrypted
	if err := packets.Push(decrypted); err != nil {
		return nil, err
	}
	return readSignedMessage(packets, md, keyring)
}

func readSignedMessage(packets *packet.Reader, mdin *MessageDetails, keyring KeyRing) (md *MessageDetails, err error) {
	if mdin == nil {
		mdin = new(MessageDetails)
	}
	md = mdin

	var p packet.Packet
	var h hash.Hash
	var wrappedHash hash.Hash
FindLiteralData:
	for {
		p, err = packets.Next()
		if err != nil {
			return nil, err
		}
		switch p := p.(type) {
		case *packet.Compressed:
			if err := packets.Push(p.Body); err != nil {
				return nil, err
			}
		case *packet.OnePassSignature:
			if !p.IsLast {
				return nil, errors.UnsupportedError("nested signatures")
			}

			h, wrappedHash, err = hashForSignature(p.Hash, p.SigType)
			if err != nil {
				md = nil
				return
			}

			md.IsSigned = true
			md.SignedByKeyId = p.KeyId
			keys := keyring.KeysByIdUsage(p.KeyId, packet.KeyFlagSign)
			if len(keys) > 0 {
				md.SignedBy = &keys[0]
			}
		case *packet.LiteralData:
			md.LiteralData = p
			break FindLiteralData
		}
	}

	if md.SignedBy != nil {
		md.UnverifiedBody = &signatureCheckReader{packets, h, wrappedHash, md}
	} else if md.decrypted != nil {
		md.UnverifiedBody = checkReader{md}
	} else {
		md.UnverifiedBody = md.LiteralData.Body
	}

	return md, nil
}

func hashForSignature(hashId crypto.Hash, sigType packet.SignatureType) (hash.Hash, hash.Hash, error) {
	if !hashId.Available() {
		return nil, nil, errors.UnsupportedError("hash not available: " + strconv.Itoa(int(hashId)))
	}
	h := hashId.New()

	switch sigType {
	case packet.SigTypeBinary:
		return h, h, nil
	case packet.SigTypeText:
		return h, NewCanonicalTextHash(h), nil
	}

	return nil, nil, errors.UnsupportedError("unsupported signature type: " + strconv.Itoa(int(sigType)))
}

type checkReader struct {
	md *MessageDetails
}

func (cr checkReader) Read(buf []byte) (n int, err error) {
	n, err = cr.md.LiteralData.Body.Read(buf)
	if err == io.EOF {
		mdcErr := cr.md.decrypted.Close()
		if mdcErr != nil {
			err = mdcErr
		}
	}
	return
}

type signatureCheckReader struct {
	packets        *packet.Reader
	h, wrappedHash hash.Hash
	md             *MessageDetails
}

func (scr *signatureCheckReader) Read(buf []byte) (n int, err error) {
	n, err = scr.md.LiteralData.Body.Read(buf)
	scr.wrappedHash.Write(buf[:n])
	if err == io.EOF {
		var p packet.Packet
		p, scr.md.SignatureError = scr.packets.Next()
		if scr.md.SignatureError != nil {
			return
		}

		var ok bool
		if scr.md.Signature, ok = p.(*packet.Signature); ok {
			scr.md.SignatureError = scr.md.SignedBy.PublicKey.VerifySignature(scr.h, scr.md.Signature)
		} else if scr.md.SignatureV3, ok = p.(*packet.SignatureV3); ok {
			scr.md.SignatureError = scr.md.SignedBy.PublicKey.VerifySignatureV3(scr.h, scr.md.SignatureV3)
		} else {
			scr.md.SignatureError = errors.StructuralError("LiteralData not followed by Signature")
			return
		}

		if scr.md.decrypted != nil {
			mdcErr := scr.md.decrypted.Close()
			if mdcErr != nil {
				err = mdcErr
			}
		}
	}
	return
}

func CheckDetachedSignature(keyring KeyRing, signed, signature io.Reader) (signer *Entity, err error) {
	var issuerKeyId uint64
	var hashFunc crypto.Hash
	var sigType packet.SignatureType
	var keys []Key
	var p packet.Packet

	packets := packet.NewReader(signature)
	for {
		p, err = packets.Next()
		if err == io.EOF {
			return nil, errors.ErrUnknownIssuer
		}
		if err != nil {
			return nil, err
		}

		switch sig := p.(type) {
		case *packet.Signature:
			if sig.IssuerKeyId == nil {
				return nil, errors.StructuralError("signature doesn't have an issuer")
			}
			issuerKeyId = *sig.IssuerKeyId
			hashFunc = sig.Hash
			sigType = sig.SigType
		case *packet.SignatureV3:
			issuerKeyId = sig.IssuerKeyId
			hashFunc = sig.Hash
			sigType = sig.SigType
		default:
			return nil, errors.StructuralError("non signature packet found")
		}

		keys = keyring.KeysByIdUsage(issuerKeyId, packet.KeyFlagSign)
		if len(keys) > 0 {
			break
		}
	}

	if len(keys) == 0 {
		panic("unreachable")
	}

	h, wrappedHash, err := hashForSignature(hashFunc, sigType)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(wrappedHash, signed); err != nil && err != io.EOF {
		return nil, err
	}

	for _, key := range keys {
		switch sig := p.(type) {
		case *packet.Signature:
			err = key.PublicKey.VerifySignature(h, sig)
		case *packet.SignatureV3:
			err = key.PublicKey.VerifySignatureV3(h, sig)
		default:
			panic("unreachable")
		}

		if err == nil {
			return key.Entity, nil
		}
	}

	return nil, err
}

func CheckArmoredDetachedSignature(keyring KeyRing, signed, signature io.Reader) (signer *Entity, err error) {
	body, err := readArmored(signature, SignatureType)
	if err != nil {
		return
	}

	return CheckDetachedSignature(keyring, signed, body)
}
