package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/store"
)

type ModelRepository struct {
	store *store.JSONStore
}

func NewModelRepository(store *store.JSONStore) *ModelRepository {
	return &ModelRepository{store: store}
}

func (r *ModelRepository) Create(m *models.Model) error {
	if m.ID == "" {
		return fmt.Errorf("model ID is required")
	}
	m.CreatedAt = time.Now().UTC()
	m.UpdatedAt = m.CreatedAt
	return r.store.Save("models", m.ID, m)
}

func (r *ModelRepository) GetAll() ([]models.Model, error) {
	data, err := r.store.GetAll("models")
	if err != nil {
		return nil, err
	}

	var modelsList []models.Model
	for _, item := range data {
		var m models.Model
		if err := json.Unmarshal(item, &m); err != nil {
			continue
		}
		modelsList = append(modelsList, m)
	}
	return modelsList, nil
}

func (r *ModelRepository) GetByID(id string) (*models.Model, error) {
	var m models.Model
	if err := r.store.Get("models", id, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ModelRepository) Update(m *models.Model) error {
	m.UpdatedAt = time.Now().UTC()
	return r.store.Save("models", m.ID, m)
}

func (r *ModelRepository) Delete(id string) error {
	return r.store.Delete("models", id)
}
