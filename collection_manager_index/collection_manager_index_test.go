package collection_manager_index

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mahdi-cpp/database-service/internal/collection_manager_json"
	"github.com/mahdi-cpp/database-service/internal/collections/message"
)

const databaseDirectory = "/app/tmp/messages"

func TestCreateMessages(t *testing.T) {

	db, err := New[*message.Message, *message.Index](databaseDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	v7, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}

	userID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	chatID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}

	var count int
	for i := 0; i < 1; i++ {

		msg := &message.Message{
			// DO NOT set the ID here. Let the Create function handle it.
			UserID:   userID,
			ChatID:   chatID,
			Caption:  "caption of message number " + strconv.Itoa(i),
			IsEdited: true,
			IsPinned: true,
			Medias: []*message.Media{
				{
					ID:          v7,
					MimeType:    "image/jpeg",
					Width:       1,
					Height:      1000,
					Orientation: "portrait",
					Tags: []message.Tag{
						{
							ID:       userID,
							Username: "ali",
							X:        150,
							Y:        270,
						},
						{
							ID:       userID,
							Username: "mahdi.cpp",
							X:        300,
							Y:        300,
						},
						{
							ID: userID,
							X:  1200,
							Y:  1800,
						},
					},
				},
				{
					ID:          v7,
					MimeType:    "video/mp4",
					Width:       2,
					Height:      1000,
					Orientation: "portrait",
					Tags: []message.Tag{
						{
							ID: userID,
							X:  10,
							Y:  10,
						},
						{
							ID: userID,
							X:  20,
							Y:  20,
						},
						{
							ID: userID,
							X:  30,
							Y:  30,
						},
					},
				},
				{
					ID:          v7,
					MimeType:    "video/webm",
					Width:       3,
					Height:      1000,
					Orientation: "portrait",
				},
				{
					ID:          v7,
					MimeType:    "image/jpeg",
					Width:       4,
					Height:      1000,
					Orientation: "portrait",
				},
				{
					ID:          v7,
					MimeType:    "video/mp4",
					Width:       5,
					Height:      1000,
					Orientation: "portrait",
				},
				{
					ID:          v7,
					MimeType:    "video/webm",
					Width:       6,
					Height:      1000,
					Orientation: "portrait",
				},
				{
					ID:          v7,
					MimeType:    "video/mp4",
					Width:       8,
					Height:      1000,
					Orientation: "portrait",
				},
			},
			Voice: &message.Voice{
				ID:       v7,
				MimeType: "audio/opus",
				Duration: 5000,
				FileSize: 25000,
			},
			Music: &message.Music{
				ID:       v7,
				MimeType: "audio/ogg",
				Duration: 5000,
				FileSize: 25000,
				Artist:   "Reza Golzar",
				Album:    "Golstan",
			},
			Poll: &message.Poll{
				Question: "Which programming language do you prefer?",
				Options: []message.PollOption{
					{
						Text:     "Go",
						Votes:    3,
						VoterIDs: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
					},
					{
						Text:     "Python",
						Votes:    5,
						VoterIDs: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()},
					},
					{
						Text:     "Rust",
						Votes:    2,
						VoterIDs: []uuid.UUID{uuid.New(), uuid.New()},
					},
				},
				TotalVotes:            10,
				IsAnonymous:           false,
				Type:                  "single_choice",
				AllowsMultipleAnswers: false,
				CloseDate:             time.Now().Add(24 * time.Hour),
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, err := db.Create(msg)
		if err != nil {
			t.Fatal(err)
		}
		// Store the new UUID returned by the Create function
		//createdUUIDs = append(createdUUIDs, createdMsg.GetID())
		count++
	}

	fmt.Printf("Created %d messages.\n", count)
}

func TestReadAll(t *testing.T) {

	collection, err := collection_manager_json.New[*message.Message]("/app/tmp/messages/metadata")
	collectionIndex, err := collection_manager_json.New[*message.Index]("/app/tmp/messages/index")

	manager, err := New[*message.Message, *message.Index](databaseDirectory)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()
	all := manager.GetAllIndexEntries()

	for _, entry := range all {

		read, err := manager.Read(entry.IndexData.ID)
		if err != nil {
			return
		}
		//fmt.Println(read)

		_, err = collectionIndex.Create(entry.IndexData) // create json metadata
		if err != nil {
			t.Fatal(err)
		}

		_, err = collection.Create(read) // create json metadata
		if err != nil {
			t.Fatal(err)
		}
	}

	fmt.Printf("Read %d messages.\n", len(all))
}

func TestCreateMetaData(t *testing.T) {

	collection, err := collection_manager_json.New[*message.Message]("/app/tmp/messages/metadata")
	collectionIndex, err := collection_manager_json.New[*message.Index]("/app/tmp/messages/index")

	manager, err := New[*message.Message, *message.Index](databaseDirectory)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	all := manager.GetAllIndexEntries()

	for _, entry := range all {

		read, err := manager.Read(entry.IndexData.ID)
		if err != nil {
			return
		}

		_, err = collectionIndex.Create(entry.IndexData) // create json metadata
		if err != nil {
			t.Fatal(err)
		}

		_, err = collection.Create(read) // create json metadata
		if err != nil {
			t.Fatal(err)
		}
	}

}

func TestUpdate(t *testing.T) {

	manager, err := New[*message.Message, *message.Index](databaseDirectory)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	before := manager.GetAllIndexEntries()

	for _, entry := range before { //before
		fmt.Println(entry.IndexData.ID.String())
	}

	for _, entry := range before { //update
		read, err := manager.Read(entry.IndexData.ID)
		if err != nil {
			return
		}

		read.Caption = "Mahdi"
		_, err = manager.Update(read)
		if err != nil {
			t.Fatal(err)
		}
	}

	after := manager.GetAllIndexEntries()
	fmt.Println("---------------------------------------")
	for _, entry := range after { //after
		fmt.Println(entry.IndexData.ID.String())
	}

}

func TestManagerOperations2(t *testing.T) {

	// 2. ایجاد یک نمونه جدید از Manager
	manager, err := New[*message.Message, *message.Index](databaseDirectory)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	// اضافه کردن defer برای بستن manager در پایان اجرای تابع
	defer manager.Close()

	//v7, err := uuid.NewV7()
	//if err != nil {
	//	return
	//}
	//
	//// 3. ایجاد یک پیام جدید با UUIDv7 (مطابق با درخواست شما)
	//message1 := &message.Message{
	//	ChatID:    v7,
	//	UserID:    v7,
	//	Caption:   "سلام، این یک پیام آزمایشی است.",
	//	CreatedAt: time.Now(),
	//}
	//
	//// 4. ذخیره پیام در کالکشن
	//createdMessage, err := manager.Create(message1)
	//if err != nil {
	//	t.Fatalf("Failed to create message1: %v", err)
	//}
	//t.Logf("Message created with ID: %s", createdMessage.GetID())
}

func TestReadMessages(t *testing.T) {

	//db, err := New[*message.Message, *message.Index]()
	//if err != nil {
	//	t.Fatal(err)
	//}
	//defer db.Close()
	//
	//all := db.R
	//fmt.Printf("Read %d messages.\n", len(all))
	//fmt.Printf("Cpation: %s\n", all[0].String())
	//if len(all) == 0 {
	//	t.Fatal("No messages found")

	//
	//start := time.Now()
	//
	//for i := 3000000; i < 3001000; i++ {
	//	id := all[i]
	//	//fmt.Printf("id: %s \n", id.String())
	//
	//	msg, err := db.Read(id)
	//	if err != nil {
	//		t.Fatalf("Error reading message with UUID %s: %v", id, err)
	//	}
	//	fmt.Println(msg.Caption)
	//}
	//
	//duration := time.Since(start)
	//fmt.Println("single  read duration: ", duration)

}
