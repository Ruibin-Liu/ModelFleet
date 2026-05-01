package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelfleet/modelfleet/internal/models"
	"github.com/modelfleet/modelfleet/internal/store"
)

type EventRepository struct {
	store *store.JSONStore
}

func NewEventRepository(store *store.JSONStore) *EventRepository {
	return &EventRepository{store: store}
}

func (r *EventRepository) Create(e *models.Event) error {
	if e.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	return r.store.Save("events", e.ID, e)
}

func (r *EventRepository) GetAll() ([]models.Event, error) {
	data, err := r.store.GetAll("events")
	if err != nil {
		return nil, err
	}

	var events []models.Event
	for _, item := range data {
		var e models.Event
		if err := json.Unmarshal(item, &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *EventRepository) GetRecent(limit int) ([]models.Event, error) {
	events, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	// Sort by created_at descending
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].CreatedAt.After(events[i].CreatedAt) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}

	return events, nil
}
