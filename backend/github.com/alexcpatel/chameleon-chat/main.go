package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/alexcpatel/chameleon-chat/ai"
	"github.com/alexcpatel/chameleon-chat/client"
	"github.com/alexcpatel/chameleon-chat/history"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Be cautious with this in production
		},
	}
)

func loadEnvironment() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Determine which .env file to load
	var envFile string
	if os.Getenv("GO_ENV") == "production" {
		envFile = ".env.production"
	} else if os.Getenv("GO_ENV") == "development" {
		envFile = ".env.development"
	}

	// Load the appropriate .env file
	err = godotenv.Load(envFile)
	if err != nil {
		log.Fatalf("Error loading %s file", envFile)
	}
}

func setCors(e *echo.Echo) {
	allowedOrigins := []string{os.Getenv("CORS_ALLOWED_ORIGINS")}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))
}

func startGoroutines(ctx context.Context) {
	go history.StoreMessages(ctx)
	go client.BroadcastMessages(ctx)
}

func main() {
	loadEnvironment()

	e := echo.New()
	setCors(e)

	e.GET("/characters", getCharacters)
	e.GET("/chat", handleChat)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startGoroutines(ctx)

	// Start the server in a goroutine
	go func() {
		address := fmt.Sprintf(":%s", os.Getenv("PORT"))
		if err := e.Start(address); err != nil && err != http.ErrServerClosed {
			e.Logger.Error(err)
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

func getCharacters(c echo.Context) error {
	type CharacterInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	characters := ai.GetCharacters()
	characterInfos := make([]CharacterInfo, len(characters))
	for i, char := range characters {
		characterInfos[i] = CharacterInfo{
			Name:        char.Name,
			Description: char.Description,
		}
	}

	return c.JSON(http.StatusOK, characters)
}

func handleChat(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	cli := client.AddClient()
	defer client.DeleteClient(cli)

	log.Printf("New WebSocket connection established for client: %d", cli.ID)

	// Create a context for the client loop
	ctx, cancel := context.WithCancel(c.Request().Context())
	defer cancel()

	// Start the client loop in a goroutine
	go cli.Loop(ctx)

	// Send messages to the client
	go func() {
		for {
			select {
			case outgoingMessage := <-cli.OutgoingMessageChan:
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
				log.Printf("Leaving write loop for client %d", cli.ID)
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
		var incomingMessage client.IncomingMessage
		err = json.Unmarshal(rawMessage, &incomingMessage)
		if err != nil {
			log.Printf("error parsing JSON: %v", err)
			continue
		}

		cli.IncomingMessageChan <- incomingMessage
	}

	return nil
}
