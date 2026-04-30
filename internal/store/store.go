package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type JSONStore struct {
	dataDir string
	mu      sync.RWMutex
}

func New(dataDir string) (*JSONStore, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	return &JSONStore{dataDir: dataDir}, nil
}

func (s *JSONStore) Save(collection string, id string, data interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.dataDir, collection)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file := filepath.Join(dir, id+".json")
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(file, bytes, 0644)
}

func (s *JSONStore) Get(collection string, id string, dest interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file := filepath.Join(s.dataDir, collection, id+".json")
	bytes, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not found")
		}
		return err
	}

	return json.Unmarshal(bytes, dest)
}

func (s *JSONStore) GetAll(collection string) ([]json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.dataDir, collection)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []json.RawMessage{}, nil
		}
		return nil, err
	}

	var results []json.RawMessage
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		bytes, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		results = append(results, bytes)
	}

	return results, nil
}

func (s *JSONStore) Delete(collection string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file := filepath.Join(s.dataDir, collection, id+".json")
	return os.Remove(file)
}

func (s *JSONStore) CollectionExists(collection string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(filepath.Join(s.dataDir, collection))
	return err == nil
}
