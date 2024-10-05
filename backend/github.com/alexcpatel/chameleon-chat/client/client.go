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
	cli := Client{
		ID:                  clientID,
		IncomingMessageChan: make(chan IncomingMessage, 100),
		OutgoingMessageChan: make(chan OutgoingMessage, 100),
		broadcastChan:       make(chan broadcastMessage, 100),
	}
	clients[clientID] = cli
	return cli
}

func DeleteClient(cli Client) {
	delete(clients, cli.ID)
}

func (cli *Client) Loop(ctx context.Context) {
	for {
		select {
		case incomingMessage := <-cli.IncomingMessageChan:
			log.Printf("Received message from client %d: %s", cli.ID, incomingMessage.Text)
			if err := cli.handleIncomingMessage(incomingMessage); err != nil {
				log.Printf("error handling message: %v", err)
			}
		case broadcastMessage := <-cli.broadcastChan:
			cli.OutgoingMessageChan <- OutgoingMessage{
				SenderID: broadcastMessage.SenderID,
				Text:     broadcastMessage.Text,
				IsUser:   false,
			}
		case <-ctx.Done():
			log.Printf("Leaving client loop for client %d", cli.ID)
			return
		}
	}
}

func (cli *Client) handleIncomingMessage(incomingMessage IncomingMessage) error {

	// Generate AI message
	aiMsg, err := ai.GenerateMessage(incomingMessage.Character, incomingMessage.Text)
	if err != nil {
		return err
	}

	// Store message
	history.StoreMessage(history.StoredMessage{
		ClientID: cli.ID,
		RawMsg:   incomingMessage.Text,
		AiMsg:    aiMsg,
	})

	// Send message to client
	cli.OutgoingMessageChan <- OutgoingMessage{
		SenderID: cli.ID,
		Text:     aiMsg,
		IsUser:   true,
	}

	// Broadcast the message to all clients
	broadcastChan <- broadcastMessage{
		SenderID: cli.ID,
		Text:     aiMsg,
	}

	return nil
}

func BroadcastMessages(ctx context.Context) {
	for {
		select {
		case broadcastMessage := <-broadcastChan:
			for _, cli := range clients {
				if broadcastMessage.SenderID != cli.ID {
					cli.broadcastChan <- broadcastMessage
				}
			}
		case <-ctx.Done():
			log.Println("Broadcast messages shutting down")
			return
		}
	}
}
