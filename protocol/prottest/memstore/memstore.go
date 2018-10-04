// Package memstore is a protocol.Store implementation that keeps all
// blockchain state in memory.
//
// It is used in tests to avoid needing a database.
package memstore

import (
	"context"
	"fmt"
	"sync"

	"i10r.io/protocol/bc"
	"i10r.io/protocol/state"
)

// MemStore satisfies the Store interface.
type MemStore struct {
	mu     sync.Mutex
	Blocks map[uint64]*bc.Block
	State  *state.Snapshot
}

// New returns a new MemStore.
func New() *MemStore {
	return &MemStore{Blocks: make(map[uint64]*bc.Block)}
}

// Height satisfies the protocol.Store interface.
func (m *MemStore) Height(context.Context) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return uint64(len(m.Blocks)), nil
}

// SaveBlock satisfies the protocol.Store interface.
func (m *MemStore) SaveBlock(ctx context.Context, b *bc.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.Blocks[b.Height]
	if ok && existing.Hash() != b.Hash() {
		return fmt.Errorf("already have a block at height %d", b.Height)
	}
	m.Blocks[b.Height] = b
	return nil
}

// SaveSnapshot satisfies the protocol.Store interface.
func (m *MemStore) SaveSnapshot(ctx context.Context, snapshot *state.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.State = state.Copy(snapshot)
	return nil
}

// GetBlock satisfies the protocol.Store interface.
func (m *MemStore) GetBlock(ctx context.Context, height uint64) (*bc.Block, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.Blocks[height]
	if !ok {
		return nil, fmt.Errorf("memstore: no block at height %d", height)
	}
	return b, nil
}

// LatestSnapshot satisfies the protocol.Store interface.
func (m *MemStore) LatestSnapshot(context.Context) (*state.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.State == nil {
		m.State = state.Empty()
	}
	return state.Copy(m.State), nil
}

// FinalizeHeight satisfies the protocol.Store interface.
func (m *MemStore) FinalizeHeight(context.Context, uint64) error { return nil }
