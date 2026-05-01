package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/store"
)

type APIKeyRepository struct {
	store *store.JSONStore
}

func NewAPIKeyRepository(store *store.JSONStore) *APIKeyRepository {
	return &APIKeyRepository{store: store}
}

func (r *APIKeyRepository) Create(key *models.APIKey) error {
	if key.ID == "" {
		return fmt.Errorf("API key ID is required")
	}
	key.CreatedAt = time.Now().UTC()
	return r.store.Save("api_keys", key.ID, key)
}

func (r *APIKeyRepository) GetAll() ([]models.APIKey, error) {
	data, err := r.store.GetAll("api_keys")
	if err != nil {
		return nil, err
	}

	var keys []models.APIKey
	for _, item := range data {
		var k models.APIKey
		if err := json.Unmarshal(item, &k); err != nil {
			continue
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *APIKeyRepository) GetByID(id string) (*models.APIKey, error) {
	var k models.APIKey
	if err := r.store.Get("api_keys", id, &k); err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *APIKeyRepository) ValidateKey(keyHash string) (*models.APIKey, error) {
	keys, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	for _, k := range keys {
		if k.KeyHash == keyHash && k.Enabled {
			// Update last used
			now := time.Now().UTC()
			k.LastUsedAt = &now
			r.store.Save("api_keys", k.ID, k)
			return &k, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

func (r *APIKeyRepository) Update(key *models.APIKey) error {
	return r.store.Save("api_keys", key.ID, key)
}

func (r *APIKeyRepository) Delete(id string) error {
	return r.store.Delete("api_keys", id)
}

func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
