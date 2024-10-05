package history

import (
	"context"
	"log"
)

type StoredMessage struct {
	ClientID int64
	RawMsg   string
	AiMsg    string
}

const MaxMessages int = 1000

var (
	storedMessages     = NewCircularBuffer[StoredMessage](MaxMessages)
	storedMessagesChan = make(chan StoredMessage, 100)
)

func StoreMessages(ctx context.Context) {
	for {
		select {
		case msg := <-storedMessagesChan:
			storedMessages.Push(msg)
		case <-ctx.Done():
			log.Println("Message handler shutting down")
			return
		}
	}
}

func StoreMessage(msg StoredMessage) {
	storedMessagesChan <- msg
}

func GetHistory(n int) []StoredMessage {
	return storedMessages.LastN(n)
}
