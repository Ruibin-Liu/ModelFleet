package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/store"
)

type DeploymentRepository struct {
	store *store.JSONStore
}

func NewDeploymentRepository(store *store.JSONStore) *DeploymentRepository {
	return &DeploymentRepository{store: store}
}

func (r *DeploymentRepository) Create(d *models.Deployment) error {
	if d.ID == "" {
		return fmt.Errorf("deployment ID is required")
	}
	d.CreatedAt = time.Now().UTC()
	d.UpdatedAt = d.CreatedAt
	return r.store.Save("deployments", d.ID, d)
}

func (r *DeploymentRepository) GetAll() ([]models.Deployment, error) {
	data, err := r.store.GetAll("deployments")
	if err != nil {
		return nil, err
	}

	var deployments []models.Deployment
	for _, item := range data {
		var d models.Deployment
		if err := json.Unmarshal(item, &d); err != nil {
			continue
		}
		deployments = append(deployments, d)
	}
	return deployments, nil
}

func (r *DeploymentRepository) GetByID(id string) (*models.Deployment, error) {
	var d models.Deployment
	if err := r.store.Get("deployments", id, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeploymentRepository) Update(d *models.Deployment) error {
	d.UpdatedAt = time.Now().UTC()
	return r.store.Save("deployments", d.ID, d)
}

func (r *DeploymentRepository) UpdateStatus(id string, status string) error {
	d, err := r.GetByID(id)
	if err != nil {
		return err
	}
	d.Status = status
	d.UpdatedAt = time.Now().UTC()
	return r.store.Save("deployments", d.ID, d)
}

func (r *DeploymentRepository) Delete(id string) error {
	return r.store.Delete("deployments", id)
}
