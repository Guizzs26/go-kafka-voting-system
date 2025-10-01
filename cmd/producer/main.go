package main

import (
	"context"
	"log"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/simulation"
)

func main() {
	kafkaBrokers := []string{"localhost:9092"}
	topic := "votes"
	concurrency := 50    // 50 goroutines publishing in parallel
	totalVotes := 100000 // 100,000 votes in total

	log.Println("Starting Producer in stress test mode...")

	publisher, err := event.NewKafkaPublisher(kafkaBrokers, topic)
	if err != nil {
		log.Fatalf("Error creating Kafka publisher: %v", err)
	}
	defer publisher.Close()

	sim := simulation.New(publisher, concurrency, totalVotes)

	if err := sim.Run(context.Background()); err != nil {
		log.Fatalf("Error during stress test: %v", err)
	}
}
