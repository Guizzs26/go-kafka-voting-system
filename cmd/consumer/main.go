package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/metrics"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/processing"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/pubsub"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/store"
	"github.com/coder/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	kafkaBrokers := []string{"localhost:9092"}
	votesTopic := "votes"
	dlqTopic := "invalid_votes"
	groupID := "vote-processor-group"
	metricsAddr := ":8081"
	redisAddr := "redis://localhost:6379/0"
	numWorkers := runtime.NumCPU()

	hub := pubsub.NewHub()
	go hub.Run()

	log.Printf("Starting Consumer of topic '%s' in group '%s'...\n", votesTopic, groupID)

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appMetrics := metrics.NewProcessorMetrics("voting_system", "consumer")

	voteStore, err := store.NewRedisStore(mainCtx, redisAddr)
	if err != nil {
		log.Fatalf("Error creating state store (Redis): %v", err)
	}
	defer voteStore.Close()

	publisher, err := event.NewKafkaPublisher(kafkaBrokers, dlqTopic)
	if err != nil {
		log.Fatalf("Error creating kafka publisher for DLQ: %v", err)
	}

	consumer, err := event.NewKafkaConsumer(kafkaBrokers, votesTopic, groupID)
	if err != nil {
		log.Fatalf("Error creating kafka consumer: %v", err)
	}
	defer consumer.Close()

	processor := processing.NewVoteProcessor(consumer, publisher, appMetrics, voteStore, hub, numWorkers)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go startServer(metricsAddr, hub)

	go func() {
		if err := processor.Run(mainCtx); err != nil {
			log.Printf("Error during processor execution: %v", err)
		}
	}()

	log.Println("Consumer is running. Press Ctrl+C to exit.")
	log.Printf("Metrics available at http://localhost%s/metrics", metricsAddr)
	// The `main` blocks here, waiting for a shutdown signal
	<-signalChan

	log.Println("Shutdown signal received, stopping the consumer...")
	cancel()

	log.Println("Consumer terminated")
}

func startServer(addr string, hub *pubsub.Hub) {
	log.Printf("HTTP and Metrics Server listening on %s", addr)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/ws/votes/", handleWebSocket(hub))

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Failed to initialize the HTTP server: %v", err)
	}
}

func handleWebSocket(hub *pubsub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pollID := r.URL.Path[len("/ws/votes/"):]
		if pollID == "" {
			http.Error(w, "Poll ID is required", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("WebSocket accept error: %v", err)
			return
		}

		c := &pubsub.Client{
			Hub:    hub,
			Conn:   conn,
			Send:   make(chan []byte, 256),
			PollID: pollID,
		}

		c.Hub.Register <- c

		go c.WritePump()
		c.ReadPump()
	}
}
