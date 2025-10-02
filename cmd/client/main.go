package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coder/websocket"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Correct usage: go run ./cmd/client/main.go <poll-id>")
	}
	pollID := os.Args[1]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Println("Shutting 'client' down...")
		cancel()
	}()

	url := fmt.Sprintf("ws://localhost:8081/ws/votes/%s", pollID)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "client exit")

	log.Printf("Listening for updates on poll '%s'...", pollID)
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("Connection closed")
				return
			}
			log.Printf("Read error: %v", err)
			return
		}
		log.Printf("Updated score: %s", string(msg))
	}
}
