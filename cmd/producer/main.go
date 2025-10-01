package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/simulation"
)

func main() {
	kBrokers := []string{"localhost:9092"}
	topic := "votes"

	kp, err := event.NewKafkaPublisher(kBrokers, topic)
	if err != nil {
		log.Fatalf("failed to create kafka publisher: %v", err)
	}
	defer kp.Close()

	sim := simulation.New(kp)

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := sim.Run(mainCtx); err != nil {
			log.Printf("Error while running simulator: %v", err)
		}
	}()

	// `main` now hangs here, waiting for a shutdown signal
	log.Println("Producer is running. Press Ctrl+C to exit")
	<-signalChan

	// Upon receiving the signal, we cancel the context, which will cause sim.Run() to stop
	log.Println("Shutdown signal received, stopping the producer...")
	cancel()

	log.Println("Producer terminated")
}
