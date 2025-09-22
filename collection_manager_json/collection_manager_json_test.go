package collection_manager_json

import (
	"fmt"
	"testing"

	"github.com/mahdi-cpp/database-service/internal/collections/message"
)

func TestNew(t *testing.T) {

	collectionManager, err := New[*message.Message]("/app/tmp/messages/")
	if err != nil {
		t.Error(err)
	}

	fmt.Println("collection count: ", collectionManager.Count())

	//for i := 0; i < 1000; i++ {
	//	msg := message.Message{
	//		ID:      uuid.New(),
	//		Caption: "message_number_" + strconv.Itoa(i),
	//		Medias: []*message.Media{
	//			{
	//				ID:       uuid.New(),
	//				MimeType: "text/plain",
	//			},
	//			{
	//				ID:       uuid.New(),
	//				MimeType: "text/html",
	//			},
	//		},
	//		CreatedAt: time.Now(),
	//		UpdatedAt: time.Now(),
	//	}
	//	_, err := collectionManager.Create(&msg)
	//	if err != nil {
	//		t.Error(err)
	//	}
	//}
}
