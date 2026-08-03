package grpcserver

import (
	"fmt"

	"openchat/internal/blockchain"
	"openchat/internal/grpcserver/pb"
)

func txFromPB(req *pb.SMSRequest) (*blockchain.Transaction, error) {
	if len(req.NonceAEAD) != 12 {
		return nil, fmt.Errorf("nonce_aead must be 12 bytes, got %d", len(req.NonceAEAD))
	}
	if len(req.EphemeralPubkey) != 32 {
		return nil, fmt.Errorf("ephemeral_pubkey must be 32 bytes, got %d", len(req.EphemeralPubkey))
	}
	tx := &blockchain.Transaction{
		From:      req.FromAddress,
		To:        req.ToAddress,
		Ciphertext: req.Ciphertext,
		Nonce:     req.Nonce,
		Timestamp: req.Timestamp,
		Signature: req.Signature,
	}
	copy(tx.NonceAEAD[:], req.NonceAEAD)
	copy(tx.EphemeralPubkey[:], req.EphemeralPubkey)
	return tx, nil
}

func pbFromTx(tx *blockchain.Transaction) *pb.SMSRequest {
	return &pb.SMSRequest{
		FromAddress:     tx.From,
		ToAddress:       tx.To,
		Ciphertext:      tx.Ciphertext,
		NonceAEAD:       append([]byte{}, tx.NonceAEAD[:]...),
		EphemeralPubkey: append([]byte{}, tx.EphemeralPubkey[:]...),
		Nonce:           tx.Nonce,
		Timestamp:       tx.Timestamp,
		Signature:       tx.Signature,
	}
}

// BlockToPB converts a committed blockchain.Block to its wire form, for
// GetBlocks (relay-node chain sync/backfill). Exported so cmd/node can
// reuse the exact same conversion when ingesting TopicBlockCommit gossip
// and GetBlocks backfill responses.
func BlockToPB(b *blockchain.Block) *pb.BlockPB {
	txs := make([]*pb.SMSRequest, len(b.Transactions))
	for i, tx := range b.Transactions {
		txs[i] = pbFromTx(tx)
	}
	sigs := make([]*pb.ValidatorSigPB, len(b.CommitSigs))
	for i, cs := range b.CommitSigs {
		sigs[i] = &pb.ValidatorSigPB{Validator: cs.Validator, Round: cs.Round, Signature: cs.Signature}
	}
	return &pb.BlockPB{
		Height:       b.Height,
		PrevHash:     b.PrevHash,
		Timestamp:    b.Timestamp,
		Proposer:     b.Proposer,
		Transactions: txs,
		MerkleRoot:   b.MerkleRoot,
		CommitSigs:   sigs,
	}
}

// BlockFromPB is BlockToPB's inverse, used by a relay node ingesting
// blocks fetched via GetBlocks (or received over TopicBlockCommit).
func BlockFromPB(bp *pb.BlockPB) (*blockchain.Block, error) {
	txs := make([]*blockchain.Transaction, len(bp.Transactions))
	for i, t := range bp.Transactions {
		tx, err := txFromPB(t)
		if err != nil {
			return nil, fmt.Errorf("block %d tx %d: %w", bp.Height, i, err)
		}
		txs[i] = tx
	}
	sigs := make([]blockchain.ValidatorSig, len(bp.CommitSigs))
	for i, s := range bp.CommitSigs {
		sigs[i] = blockchain.ValidatorSig{Validator: s.Validator, Round: s.Round, Signature: s.Signature}
	}
	return &blockchain.Block{
		Height:       bp.Height,
		PrevHash:     bp.PrevHash,
		Timestamp:    bp.Timestamp,
		Proposer:     bp.Proposer,
		Transactions: txs,
		MerkleRoot:   bp.MerkleRoot,
		CommitSigs:   sigs,
	}, nil
}
