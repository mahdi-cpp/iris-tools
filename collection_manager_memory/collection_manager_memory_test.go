package collection_manager_memory

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mahdi-cpp/database-service/internal/collections/chat"
)

const databaseDirectory = "/app/tmp/chats"

func TestCreateChats(t *testing.T) {

	db, err := New[*chat.Chat](databaseDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for i := 0; i < 10; i++ {

		userID1, err := uuid.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		userID2, err := uuid.NewV7()
		if err != nil {
			t.Fatal(err)
		}

		ch := &chat.Chat{
			Username:    fmt.Sprintf("user%d", i),
			IsPinned:    true,
			Type:        "bot",
			Version:     "2",
			Description: "mahdi bot" + strconv.Itoa(i),
			Avatar:      "                                        ",
			Members: []chat.Member{
				{
					UserID:     userID1,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
				{
					UserID:     userID2,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
				{
					UserID:     userID1,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
				{
					UserID:     userID2,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
				{
					UserID:     userID1,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
				{
					UserID:     userID2,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
				{
					UserID:     userID1,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
				{
					UserID:     userID2,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
				{
					UserID:     userID1,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
				{
					UserID:     userID2,
					Role:       "member",
					IsActive:   true,
					JoinedAt:   time.Now(),
					LastActive: time.Now(),
				},
			},
			CreatedAt: time.Now(),
		}

		_, err = db.Create(ch)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestReadChats(t *testing.T) {

	db, err := New[*chat.Chat](databaseDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	all, err := db.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	sh := &chat.SearchOptions{
		Type:      "private",
		SortOrder: "end",
		Sort:      "createdAt",
	}

	filterItems := chat.Search(all, sh)

	for _, ch := range filterItems {
		fmt.Println(ch.Description)
	}
}

func TestUpdate(t *testing.T) {
	db, err := New[*chat.Chat](databaseDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	all, err := db.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	updateOptions := &chat.UpdateOptions{
		Type: "bot",
	}

	for _, ch := range all {
		chat.Update(ch, updateOptions)
		_, err := db.Update(ch)
		if err != nil {
			t.Fatal(err)
		}
	}
}
