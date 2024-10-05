package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Be cautious with this in production
		},
	}
)

func main() {
	e := echo.New()

	e.GET("/chat", handleChat)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the message handler in a goroutine
	go storeMessages(ctx)
	go broadcastMessages(ctx)

	// Start the server in a goroutine
	go func() {
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	// Cancel the context to stop the message handler
	cancel()

	// Shutdown the server with a timeout
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func handleChat(ec echo.Context) error {
	ws, err := upgrader.Upgrade(ec.Response(), ec.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	client := addClient()
	defer deleteClient(client)

	log.Printf("New WebSocket connection established for client: %d", client.ID)

	// Create a context for the client loop
	ctx, cancel := context.WithCancel(ec.Request().Context())
	defer cancel()

	// Start the client loop in a goroutine
	go client.loop(ctx)

	// Send messages to the client
	go func() {
		for {
			select {
			case outgoingMessage := <-client.outgoingMessages:
				// Marshal the response to JSON
				jsonResponse, err := json.Marshal(outgoingMessage)
				if err != nil {
					log.Printf("error marshaling JSON: %v", err)
					continue
				}

				// Send JSON response back to the client
				err = ws.WriteMessage(websocket.TextMessage, jsonResponse)
				if err != nil {
					log.Printf("error: %v", err)
				}
			case <-ctx.Done():
				log.Printf("Leaving write loop for client %d", client.ID)
				return
			}
		}
	}()

	// Read messages from the browser
	for {
		_, rawMessage, err := ws.ReadMessage()
		if err != nil {
			log.Printf("error: %v", err)
			break
		}

		// Parse the received message as JSON
		var incomingMessage IncomingMessage
		err = json.Unmarshal(rawMessage, &incomingMessage)
		if err != nil {
			log.Printf("error parsing JSON: %v", err)
			continue
		}

		client.incomingMessages <- incomingMessage
	}

	return nil
}
