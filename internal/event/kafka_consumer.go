package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(brokers []string, topic, groupID string) (*KafkaConsumer, error) {
	rCfg := kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10kb
		MaxBytes: 10e6, // 10mb
		MaxWait:  1 * time.Second,
		// If this is a new groupID, with no offsets saved, we start reading
		// from the last post in the topic (we don't reprocess the history)
		StartOffset: kafka.FirstOffset,
	}
	r := kafka.NewReader(rCfg)

	return &KafkaConsumer{reader: r}, nil
}

func (kc *KafkaConsumer) ReadMessage(ctx context.Context) (model.Vote, error) {
	// `ReadMessage` is a blocking call. It waits until a new
	// message arrives, or the context is canceled
	msg, err := kc.reader.ReadMessage(ctx)
	if err != nil {
		// If the error is context canceled or EOF (end of stream),
		// it's a clean shutdown signal, so we return the error so
		// the loop that called us can stop
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
			return model.Vote{}, err
		}
		// For other errors, we just log and perhaps return the error
		// for more complex retry logic
		log.Printf("error reading message from Kafka: %v", err)
		return model.Vote{}, err
	}

	// sucessfull read, deserialize the message
	var vote model.Vote
	if err := json.Unmarshal(msg.Value, &vote); err != nil {
		log.Printf("error deserializing vote: %v", err)
		return model.Vote{}, err
	}

	return vote, nil
}

func (kc *KafkaConsumer) Close() error {
	if err := kc.reader.Close(); err != nil {
		return fmt.Errorf("failed to close kafka reader: %v", err)
	}
	return nil
}
