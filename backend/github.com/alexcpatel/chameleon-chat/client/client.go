package client

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/alexcpatel/chameleon-chat/ai"
	"github.com/alexcpatel/chameleon-chat/history"
)

type IncomingMessage struct {
	Text      string `json:"text"`
	Character string `json:"character"`
}

type OutgoingMessage struct {
	SenderID int64  `json:"senderId"`
	Text     string `json:"text"`
	IsUser   bool   `json:"isUser"`
}

type broadcastMessage struct {
	SenderID int64
	Text     string
}

type Client struct {
	ID                  int64
	IncomingMessageChan chan IncomingMessage
	OutgoingMessageChan chan OutgoingMessage
	broadcastChan       chan broadcastMessage
}

var (
	clients               = make(map[int64]Client)
	clientIDCounter int64 = 0
	broadcastChan         = make(chan broadcastMessage, 1000)
)

func AddClient() Client {
	clientID := atomic.AddInt64(&clientIDCounter, 1)
	client := Client{
		ID:                  clientID,
		IncomingMessageChan: make(chan IncomingMessage, 100),
		OutgoingMessageChan: make(chan OutgoingMessage, 100),
		broadcastChan:       make(chan broadcastMessage, 100),
	}
	clients[clientID] = client
	return client
}

func DeleteClient(client Client) {
	delete(clients, client.ID)
}

func (client *Client) Loop(ctx context.Context) {
	for {
		select {
		case incomingMessage := <-client.IncomingMessageChan:
			if err := client.handleIncomingMessage(incomingMessage); err != nil {
				log.Printf("error handling message: %v", err)
			}
		case broadcastMessage := <-client.broadcastChan:
			client.OutgoingMessageChan <- OutgoingMessage{
				SenderID: broadcastMessage.SenderID,
				Text:     broadcastMessage.Text,
				IsUser:   false,
			}
		case <-ctx.Done():
			log.Printf("Leaving client loop for client %d", client.ID)
			return
		}
	}
}

func (client *Client) handleIncomingMessage(incomingMessage IncomingMessage) error {
	// Print the received message
	log.Printf("Received message from client %d: %s", client.ID, incomingMessage.Text)

	// Generate AI message
	aiMsg, err := ai.GenerateMessage(incomingMessage.Character, incomingMessage.Text)
	if err != nil {
		log.Printf("error handling message: %v", err)
		return err
	}

	// Store message
	history.StoreMessage(history.StoredMessage{
		ClientID: client.ID,
		RawMsg:   incomingMessage.Text,
		AiMsg:    aiMsg,
	})

	// Send message to client
	client.OutgoingMessageChan <- OutgoingMessage{
		SenderID: client.ID,
		Text:     aiMsg,
		IsUser:   true,
	}

	// Broadcast the message to all clients
	broadcastChan <- broadcastMessage{
		SenderID: client.ID,
		Text:     aiMsg,
	}

	return nil
}

func BroadcastMessages(ctx context.Context) {
	for {
		select {
		case broadcastMessage := <-broadcastChan:
			for _, client := range clients {
				if broadcastMessage.SenderID != client.ID {
					client.broadcastChan <- broadcastMessage
				}
			}
		case <-ctx.Done():
			log.Println("Broadcast messages shutting down")
			return
		}
	}
}
