package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/processing"
)

func main() {
	kafkaBrokers := []string{"localhost:9092"}
	topic := "votes"
	groupID := "vote-processor-group"
	log.Printf("Starting Consumer of topic '%s' in group '%s'...\n", topic, groupID)

	consumer, err := event.NewKafkaConsumer(kafkaBrokers, topic, groupID)
	if err != nil {
		log.Fatalf("Error creating Kafka consumer: %v", err)
	}
	defer consumer.Close()

	processor := processing.NewVoteProcessor(consumer)

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

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
