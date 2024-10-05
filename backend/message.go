package main

import (
	"context"
	"log"
)

type StoredMessage struct {
	ClientID int64
	RawMsg   string
	AiMsg    string
}

type IncomingMessage struct {
	Text string `json:"text"`
}

type OutgoingMessage struct {
	SenderID int64  `json:"senderId"`
	Text     string `json:"text"`
	IsUser   bool   `json:"isUser"`
}

const MAX_MESSAGES int = 1000

var (
	storedMessages     = make([]StoredMessage, MAX_MESSAGES)
	storedMessagesChan = make(chan StoredMessage, 100)
)

func storeMessages(ctx context.Context) {
	for {
		select {
		case msg := <-storedMessagesChan:
			if len(storedMessages) > MAX_MESSAGES {
				storedMessages = storedMessages[1:]
			}
			storedMessages = append(storedMessages, msg)
		case <-ctx.Done():
			log.Println("Message handler shutting down")
			return
		}
	}
}
