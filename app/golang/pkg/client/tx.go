package client

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"openchat/internal/blockchain"
	"openchat/internal/grpcserver/pb"
	"openchat/pkg/crypto"
)

// buildTx assembles and signs a blockchain.Transaction client-side. It
// reuses blockchain.Transaction's exact SigningBytes()/Sign() so the
// client and node always agree on the signed byte layout — there is only
// one implementation of "what gets signed", imported by both sides.
func buildTx(w *Wallet, toAddress string, sealed *crypto.Sealed) *blockchain.Transaction {
	tx := &blockchain.Transaction{
		From:      w.Address(),
		To:        toAddress,
		Ciphertext: sealed.Ciphertext,
		Nonce:     w.NextNonce(),
		Timestamp: time.Now().UnixMilli(),
	}
	tx.NonceAEAD = sealed.Nonce
	tx.EphemeralPubkey = sealed.SenderX25519Pub
	tx.Sign(w.Keys.SigningPrivate)
	return tx
}

// ed25519Sign is a tiny helper so client.go doesn't need to import
// crypto/ed25519 directly.
func ed25519Sign(w *Wallet, msg []byte) []byte {
	return ed25519.Sign(w.Keys.SigningPrivate, msg)
}

// decryptIncoming reverses buildTx's encryption for a message pushed by
// StreamIncomingSMS: ECDH(recipientPriv, senderX25519Pub) -> AES-256-GCM
// open.
func decryptIncoming(w *Wallet, msg *pb.SMSRequest) ([]byte, error) {
	if len(msg.NonceAEAD) != 12 || len(msg.EphemeralPubkey) != 32 {
		return nil, fmt.Errorf("client: malformed incoming message envelope")
	}
	var nonce [12]byte
	var senderPub [32]byte
	copy(nonce[:], msg.NonceAEAD)
	copy(senderPub[:], msg.EphemeralPubkey)

	return crypto.Decrypt(w.Keys.EncryptionPrivate, senderPub, nonce, msg.Ciphertext)
}

// decryptOutgoing reverses buildTx's encryption for one of THIS wallet's
// own past outgoing messages, recovered via FetchHistory. Unlike
// decryptIncoming, the transaction itself doesn't carry the key needed:
// it only stores the sender's (our) X25519 pubkey, since that's all a
// recipient ever needs. Decrypting our own old sent ciphertext instead
// needs the recipient's X25519 pubkey — ECDH is commutative
// (ourPriv*recipientPub == recipientPriv*ourPub), so it recomputes the
// exact same shared secret buildTx's Encrypt used, but the caller has to
// already know that key from somewhere else (a saved contact, or an
// earlier incoming message from that same address seen elsewhere in the
// same history scan) since the chain alone doesn't carry it.
func decryptOutgoing(w *Wallet, msg *pb.SMSRequest, recipientX25519Pub [32]byte) ([]byte, error) {
	if len(msg.NonceAEAD) != 12 {
		return nil, fmt.Errorf("client: malformed outgoing message envelope")
	}
	var nonce [12]byte
	copy(nonce[:], msg.NonceAEAD)

	return crypto.Decrypt(w.Keys.EncryptionPrivate, recipientX25519Pub, nonce, msg.Ciphertext)
}

// txHashFromPB recomputes the same hash the node reports as TxHash (see
// blockchain.Transaction.Hash) from a transaction as it appears on the
// wire, so a message recovered by FetchHistory carries the identical ID
// a live SendSMS/StreamIncomingSMS delivery of that same transaction
// would have. That lets callers dedupe a history scan against messages
// already recorded locally by tx hash instead of by (address,text,time).
func txHashFromPB(m *pb.SMSRequest) string {
	tx := &blockchain.Transaction{
		From:      m.FromAddress,
		To:        m.ToAddress,
		Ciphertext: m.Ciphertext,
		Nonce:     m.Nonce,
		Timestamp: m.Timestamp,
		Signature: m.Signature,
	}
	copy(tx.NonceAEAD[:], m.NonceAEAD)
	copy(tx.EphemeralPubkey[:], m.EphemeralPubkey)
	return tx.Hash()
}
