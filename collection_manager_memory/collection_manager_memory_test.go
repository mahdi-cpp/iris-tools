package collection_manager_memory

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func (a *Model) GetID() uuid.UUID         { return a.ID }
func (a *Model) SetID(id uuid.UUID)       { a.ID = id }
func (a *Model) SetCreatedAt(t time.Time) { a.CreatedAt = t }
func (a *Model) SetUpdatedAt(t time.Time) { a.UpdatedAt = t }
func (a *Model) GetRecordSize() int       { return 250 }

type Model struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"title"`
	Count     int       `json:"count"`
	Exist     bool      `json:"exist"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func TestNew(t *testing.T) {
	c, _ := New[*Model]("/app/tmp/collection_manager_memory", "reza")

	m := &Model{
		ID:        uuid.New(),
		Name:      "album mahdi Abdolmaleki",
		Count:     1,
		CreatedAt: time.Now(),
	}

	_, err := c.Create(m)
	if err != nil {
		t.Fatal(err)
	}
}
