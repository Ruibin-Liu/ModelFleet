package repository

import (
	"fmt"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/store"
)

type MachineFactsRepository struct {
	store *store.JSONStore
}

func NewMachineFactsRepository(store *store.JSONStore) *MachineFactsRepository {
	return &MachineFactsRepository{store: store}
}

func (r *MachineFactsRepository) Save(facts *models.MachineFacts) error {
	if facts.MachineID == "" {
		return fmt.Errorf("machine_id is required")
	}
	facts.DetectedAt = time.Now().UTC()
	return r.store.Save("machine_facts", facts.MachineID, facts)
}

func (r *MachineFactsRepository) GetByMachineID(machineID string) (*models.MachineFacts, error) {
	var facts models.MachineFacts
	if err := r.store.Get("machine_facts", machineID, &facts); err != nil {
		return nil, err
	}
	return &facts, nil
}

func (r *MachineFactsRepository) Delete(machineID string) error {
	return r.store.Delete("machine_facts", machineID)
}
