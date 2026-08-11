package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"tax-client/backend/internal/domain"
)

type JSONRepository struct {
	mu   sync.RWMutex
	path string
	data domain.Database
}

func NewJSONRepository(path string, seed domain.Database) (*JSONRepository, error) {
	r := &JSONRepository{path: path, data: seed}
	raw, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(raw, &r.data); err != nil {
			return nil, fmt.Errorf("decode database: %w", err)
		}
		return r, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read database: %w", err)
	}
	if err := r.persist(seed); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *JSONRepository) Snapshot(_ context.Context) (domain.Database, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	raw, err := json.Marshal(r.data)
	if err != nil {
		return domain.Database{}, err
	}
	var clone domain.Database
	if err := json.Unmarshal(raw, &clone); err != nil {
		return domain.Database{}, err
	}
	return clone, nil
}

func (r *JSONRepository) Update(_ context.Context, mutate func(*domain.Database) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	raw, err := json.Marshal(r.data)
	if err != nil {
		return err
	}
	var next domain.Database
	if err := json.Unmarshal(raw, &next); err != nil {
		return err
	}
	if err := mutate(&next); err != nil {
		return err
	}
	if err := r.persist(next); err != nil {
		return err
	}
	r.data = next
	return nil
}

func (r *JSONRepository) persist(data domain.Database) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o750); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode database: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(r.path), ".tax-db-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp database: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, r.path); err != nil {
		return fmt.Errorf("replace database: %w", err)
	}
	return nil
}
