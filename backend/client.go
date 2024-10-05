package main

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID               int64
	Conn             *websocket.Conn
	incomingMessages chan IncomingMessage
	outgoingMessages chan OutgoingMessage
	broadcastChan    chan BroadcastMessage
}

var (
	clients               = make(map[int64]Client)
	clientIDCounter int64 = 0
	broadcastChan         = make(chan BroadcastMessage, 1000)
)

type BroadcastMessage struct {
	SenderID int64
	Text     string
}

func addClient() Client {
	clientID := atomic.AddInt64(&clientIDCounter, 1)
	client := Client{
		ID:               clientID,
		incomingMessages: make(chan IncomingMessage, 100),
		outgoingMessages: make(chan OutgoingMessage, 100),
		broadcastChan:    make(chan BroadcastMessage, 100),
	}
	clients[clientID] = client
	return client
}

func deleteClient(client Client) {
	delete(clients, client.ID)
}

func (client *Client) loop(ctx context.Context) {
	for {
		select {
		case incomingMessage := <-client.incomingMessages:
			if err := client.handleIncomingMessage(incomingMessage); err != nil {
				log.Printf("error handling message: %v", err)
			}
		case broadcastMessage := <-client.broadcastChan:
			client.outgoingMessages <- OutgoingMessage{
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
	aiMsg, err := generateAiMessage(incomingMessage.Text)
	if err != nil {
		log.Printf("error handling message: %v", err)
		return err
	}

	// Store message
	storedMessagesChan <- StoredMessage{
		ClientID: client.ID,
		RawMsg:   incomingMessage.Text,
		AiMsg:    aiMsg,
	}

	// Send message to client
	client.outgoingMessages <- OutgoingMessage{
		SenderID: client.ID,
		Text:     aiMsg,
		IsUser:   true,
	}

	// Broadcast the message to all clients
	broadcastChan <- BroadcastMessage{
		SenderID: client.ID,
		Text:     incomingMessage.Text,
	}

	return nil
}

// New goroutine to handle broadcasting messages
func broadcastMessages(ctx context.Context) {
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
