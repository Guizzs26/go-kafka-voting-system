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
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	kafkaBrokers := []string{"localhost:9092"}
	topic := "votes"
	groupID := "vote-processor-group"
	metricsAddrs := ":8081"

	log.Printf("Starting Consumer of topic '%s' in group '%s'...\n", topic, groupID)

	appMetrics := metrics.NewProcessorMetrics("voting_system", "consumer")

	consumer, err := event.NewKafkaConsumer(kafkaBrokers, topic, groupID)
	if err != nil {
		log.Fatalf("Error creating Kafka consumer: %v", err)
	}
	defer consumer.Close()

	processor := processing.NewVoteProcessor(consumer, appMetrics)
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go startMetricsServer(metricsAddrs)

	go func() {
		if err := processor.Run(mainCtx); err != nil {
			log.Printf("Error during processor execution: %v", err)
		}
	}()

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
