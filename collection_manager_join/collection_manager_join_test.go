package collection_manager_join

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
)

func (pa *PhotoAlbums) GetRecordSize() int { return 100 }
func (pa *PhotoAlbums) GetCompositeKey() string {
	return fmt.Sprintf("%s:%s", pa.AlbumID.String(), pa.PhotoID.String())
}

type PhotoAlbums struct {
	AlbumID uuid.UUID `json:"albumId"`
	PhotoID uuid.UUID `json:"photoId"`
}

func TestA(t *testing.T) {

	// 1. ایجاد یک Manager جدید
	manager, err := New[*PhotoAlbums]("/app/tmp/collection_manager_join", "photo_albums")
	if err != nil {
		fmt.Printf("Error creating manager: %v\n", err)
		return
	}
	defer manager.Close()

	// 2. ایجاد UUID برای آلبوم و عکس
	albumID := uuid.New()
	photo1ID := uuid.New()
	photo2ID := uuid.New()

	// 3. ایجاد آیتم‌های جدید و ذخیره آن‌ها
	item1 := &PhotoAlbums{AlbumID: albumID, PhotoID: photo1ID}
	item2 := &PhotoAlbums{AlbumID: albumID, PhotoID: photo2ID}

	if _, err := manager.Create(item1); err != nil {
		fmt.Printf("Error creating item1: %v\n", err)
	}
	if _, err := manager.Create(item2); err != nil {
		fmt.Printf("Error creating item2: %v\n", err)
	}
	fmt.Println("Created two photo-album relationships.")

	// 4. خواندن تمام آیتم‌ها مربوط به یک آلبوم خاص با استفاده از GetByParentID
	fmt.Printf("\nRetrieving all photos for album ID: %s\n", albumID)
	albumPhotos, err := manager.GetByParentID(albumID)
	if err != nil {
		fmt.Printf("Error retrieving items: %v\n", err)
	} else {
		fmt.Printf("Found %d photos in album.\n", len(albumPhotos))
		for _, photo := range albumPhotos {
			fmt.Printf("  - Photo ID: %s\n", photo.PhotoID)
		}
	}

	// 5. خواندن یک آیتم خاص با استفاده از کلید ترکیبی
	keyToRead := item1.GetCompositeKey()
	fmt.Printf("\nReading a specific item using composite key: %s\n", keyToRead)
	readItem, err := manager.Read(keyToRead)
	if err != nil {
		fmt.Printf("Error reading item: %v\n", err)
	} else {
		fmt.Printf("Successfully read item with AlbumID %s and PhotoID %s\n", readItem.AlbumID, readItem.PhotoID)
	}

	// 6. حذف یکی از آیتم‌ها
	keyToDelete := item2.GetCompositeKey()
	fmt.Printf("\nDeleting item with composite key: %s\n", keyToDelete)
	if err := manager.Delete(keyToDelete); err != nil {
		fmt.Printf("Error deleting item: %v\n", err)
	} else {
		fmt.Println("Item deleted successfully.")
	}

	// 7. تأیید حذف با تلاش مجدد برای خواندن
	fmt.Println("\nAttempting to retrieve all photos for the album again...")
	albumPhotosAfterDelete, err := manager.GetByParentID(albumID)
	if err != nil {
		fmt.Printf("Error: %v (as expected)\n", err)
	} else {
		fmt.Printf("Found %d photos in album after deletion.\n", len(albumPhotosAfterDelete))
	}
}
