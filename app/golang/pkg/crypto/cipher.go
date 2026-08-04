package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const aesGCMNonceSize = 12

// Sealed is the output of Encrypt: everything a recipient needs, alongside
// their own private key, to recover the plaintext. This maps 1:1 onto the
// ciphertext/nonce_aead/ephemeral_pubkey fields of pb.SMSRequest.
type Sealed struct {
	Ciphertext      []byte
	Nonce           [aesGCMNonceSize]byte
	SenderX25519Pub [32]byte
}

// Encrypt implements the sender side of E2EE: ECDH(senderPriv, recipientPub)
// -> HKDF -> AES-256-GCM seal. senderX25519Pub is included in the output so
// the recipient (who only has their own private key) can recompute the same
// shared secret.
func Encrypt(senderX25519Priv [32]byte, senderX25519Pub [32]byte, recipientX25519Pub [32]byte, plaintext []byte) (*Sealed, error) {
	aead, err := aeadFromECDH(senderX25519Priv, recipientX25519Pub)
	if err != nil {
		return nil, err
	}

	var nonce [aesGCMNonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}

	ct := aead.Seal(nil, nonce[:], plaintext, nil)

	return &Sealed{
		Ciphertext:      ct,
		Nonce:           nonce,
		SenderX25519Pub: senderX25519Pub,
	}, nil
}

// Decrypt implements the recipient side: ECDH(recipientPriv, senderPub) ->
// HKDF -> AES-256-GCM open. Because ECDH is commutative
// (priv_a*pub_b == priv_b*pub_a) this reconstructs the exact same key
// Encrypt used, without the recipient ever sending anything back.
func Decrypt(recipientX25519Priv [32]byte, senderX25519Pub [32]byte, nonce [aesGCMNonceSize]byte, ciphertext []byte) ([]byte, error) {
	aead, err := aeadFromECDH(recipientX25519Priv, senderX25519Pub)
	if err != nil {
		return nil, err
	}
	pt, err := aead.Open(nil, nonce[:], ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: authentication failed: %w", err)
	}
	return pt, nil
}

func aeadFromECDH(ourPriv [32]byte, theirPub [32]byte) (cipher.AEAD, error) {
	shared, err := curve25519.X25519(ourPriv[:], theirPub[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: ecdh: %w", err)
	}

	key := make([]byte, 32)
	kdf := hkdf.New(sha256.New, shared, nil, []byte("openchat/aes256gcm-message-key/v1"))
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, fmt.Errorf("crypto: hkdf: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm: %w", err)
	}
	return aead, nil
}
