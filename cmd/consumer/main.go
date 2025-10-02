package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/metrics"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/processing"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	kafkaBrokers := []string{"localhost:9092"}
	topic := "votes"
	groupID := "vote-processor-group"
	metricsAddr := ":8081"
	redisAddr := "redis://localhost:6379/0"

	log.Printf("Starting Consumer of topic '%s' in group '%s'...\n", topic, groupID)

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appMetrics := metrics.NewProcessorMetrics("voting_system", "consumer")

	voteStore, err := store.NewRedisStore(mainCtx, redisAddr)
	if err != nil {
		log.Fatalf("Error creating state store (Redis): %v", err)
	}
	defer voteStore.Close()

	consumer, err := event.NewKafkaConsumer(kafkaBrokers, topic, groupID)
	if err != nil {
		log.Fatalf("Error creating kafka consumer: %v", err)
	}
	defer consumer.Close()

	processor := processing.NewVoteProcessor(consumer, appMetrics, voteStore)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go startMetricsServer(metricsAddr)

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

func startMetricsServer(addr string) {
	log.Printf("Metrics server listening on %s/metrics", addr)
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Error starting metrics server: %v", err)
	}
}
