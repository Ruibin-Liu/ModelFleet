package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/store"
)

type MachineRepository struct {
	store *store.JSONStore
}

func NewMachineRepository(store *store.JSONStore) *MachineRepository {
	return &MachineRepository{store: store}
}

func (r *MachineRepository) Create(m *models.Machine) error {
	if m.ID == "" {
		return fmt.Errorf("machine ID is required")
	}
	m.CreatedAt = time.Now().UTC()
	m.UpdatedAt = m.CreatedAt
	return r.store.Save("machines", m.ID, m)
}

func (r *MachineRepository) GetAll() ([]models.Machine, error) {
	data, err := r.store.GetAll("machines")
	if err != nil {
		return nil, err
	}

	var machines []models.Machine
	for _, item := range data {
		var m models.Machine
		if err := json.Unmarshal(item, &m); err != nil {
			continue
		}
		machines = append(machines, m)
	}
	return machines, nil
}

func (r *MachineRepository) GetByID(id string) (*models.Machine, error) {
	var m models.Machine
	if err := r.store.Get("machines", id, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MachineRepository) Update(m *models.Machine) error {
	m.UpdatedAt = time.Now().UTC()
	return r.store.Save("machines", m.ID, m)
}

func (r *MachineRepository) Delete(id string) error {
	return r.store.Delete("machines", id)
}
