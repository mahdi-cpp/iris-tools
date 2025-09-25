package collection_manager_memory

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func (a *Model) SetID(id uuid.UUID)       { a.ID = id }
func (a *Model) GetID() uuid.UUID         { return a.ID }
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

func (a *PhotoAlbums) SetID(id uuid.UUID) { a.ID = id }
func (a *PhotoAlbums) GetID() uuid.UUID   { return a.ID }
func (a *PhotoAlbums) GetRecordSize() int { return 150 }

type PhotoAlbums struct {
	AlbumID uuid.UUID `json:"albumId"`
	PhotoID uuid.UUID `json:"photoID"`
}

func TestNew(t *testing.T) {

	modelCollection, _ := New[*Model]("/app/tmp/collection_manager_memory", "model")
	photoAlbumsCollection, _ := New[*PhotoAlbums]("/app/tmp/collection_manager_memory", "photo_albums")

	m := &Model{
		ID:        uuid.New(),
		Name:      "album mahdi Abdolmaleki",
		Count:     1,
		CreatedAt: time.Now(),
	}

	_, err := modelCollection.Create(m)
	if err != nil {
		t.Fatal(err)
	}

	p := &PhotoAlbums{
		AlbumID: m.ID,
		PhotoID: m.ID,
	}
	_, err = photoAlbumsCollection.Create(p)
	if err != nil {
		t.Fatal(err)
	}
}
