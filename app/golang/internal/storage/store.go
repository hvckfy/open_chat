// Package storage is the outermost persistence adapter (Clean
// Architecture: it implements the blockchain.BlockStore port using
// BadgerDB, an embedded LSM-tree KV store). Blocks, the nonce/replay
// index, and (optionally) a mempool snapshot for warm restarts all live
// here, behind one on-disk database directory per node.
package storage

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"

	"openchat/internal/blockchain"
)

var (
	prefixBlock = []byte("blk:") // blk:<height be64>            -> json(Block)
	prefixNonce = []byte("nnc:") // nnc:<hex address>             -> be64(last nonce)
	keyHead     = []byte("head") // head                          -> be64(height)
)

// Store wraps a BadgerDB instance and implements blockchain.BlockStore.
type Store struct {
	db *badger.DB
}

// Open opens (or creates) a BadgerDB database at dir.
func Open(dir string) (*Store, error) {
	opts := badger.DefaultOptions(dir).
		WithLogger(nil) // node wires its own zap logger over metrics/events instead
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("storage: open badger at %s: %w", dir, err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func blockKey(height uint64) []byte {
	k := make([]byte, len(prefixBlock)+8)
	copy(k, prefixBlock)
	binary.BigEndian.PutUint64(k[len(prefixBlock):], height)
	return k
}

func nonceKey(address string) []byte {
	return append(append([]byte{}, prefixNonce...), []byte(address)...)
}

// PutBlock persists a block and, if it extends the current tip (or this is
// genesis), advances "head".
func (s *Store) PutBlock(b *blockchain.Block) error {
	data, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("storage: marshal block %d: %w", b.Height, err)
	}
	return s.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(blockKey(b.Height), data); err != nil {
			return err
		}
		var curHead uint64
		item, err := txn.Get(keyHead)
		if err == nil {
			_ = item.Value(func(v []byte) error {
				curHead = binary.BigEndian.Uint64(v)
				return nil
			})
		} else if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}
		if b.Height >= curHead {
			var hv [8]byte
			binary.BigEndian.PutUint64(hv[:], b.Height)
			if err := txn.Set(keyHead, hv[:]); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetBlock fetches a block by height.
func (s *Store) GetBlock(height uint64) (*blockchain.Block, bool, error) {
	var block blockchain.Block
	found := false
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(blockKey(height))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return item.Value(func(v []byte) error {
			return json.Unmarshal(v, &block)
		})
	})
	if err != nil {
		return nil, false, fmt.Errorf("storage: get block %d: %w", height, err)
	}
	if !found {
		return nil, false, nil
	}
	return &block, true, nil
}

// Head returns the current chain tip height (0 if the DB is fresh).
func (s *Store) Head() (uint64, error) {
	var height uint64
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(keyHead)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			height = binary.BigEndian.Uint64(v)
			return nil
		})
	})
	if err != nil {
		return 0, fmt.Errorf("storage: get head: %w", err)
	}
	return height, nil
}

// LastNonce returns the highest committed nonce seen from `address`, or 0
// if none. This is the durable half of replay protection (the in-memory
// half lives in internal/mempool for still-pending transactions).
func (s *Store) LastNonce(address string) (uint64, error) {
	var nonce uint64
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(nonceKey(address))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			nonce = binary.BigEndian.Uint64(v)
			return nil
		})
	})
	if err != nil {
		return 0, fmt.Errorf("storage: get nonce for %s: %w", address, err)
	}
	return nonce, nil
}

// SetLastNonce persists the new high-water mark nonce for `address`.
func (s *Store) SetLastNonce(address string, nonce uint64) error {
	var v [8]byte
	binary.BigEndian.PutUint64(v[:], nonce)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(nonceKey(address), v[:])
	})
}

var _ blockchain.BlockStore = (*Store)(nil)
