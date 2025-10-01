package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
}

/*
Balancer: &kafka.Hash{}: This sets the balancer to use a hash function,
which ensures that messages with the same key are sent to the same
partition. This is useful for maintaining order for messages with
the same key (in my case, PollID).

RequiredAcks: kafka.RequireAll: This setting ensures that
the producer waits for acknowledgment from all in-sync replicas before
considering a message as successfully sent. This enhances data durability
at the cost of increased latency.
Ensures that the message will not be lost if the leader goes down shortly after receipt.

kafka.RequireOne (value 1): The default. Fastest, but least secure. The producer considers
it successful as soon as the leader broker acknowledges.

kafka.RequireNone (value 0): "Fire and forget." The producer sends and does not wait for any
acknowledgement. Fastest of all, but with no delivery guarantee.

Compression: kafka.Snappy: Our votes will be sent as JSON (text), which compresses very well.
Using compression is an excellent practice that reduces network bandwidth usage and storage
space on brokers, resulting in cost savings and often higher throughput.
*/
func NewKafkaPublisher(brokers []string, topic string) (*KafkaPublisher, error) {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		BatchTimeout: 10 * time.Millisecond,
		MaxAttempts:  5,
		Compression:  kafka.Snappy,
	}

	return &KafkaPublisher{writer: w}, nil
}

func (kp *KafkaPublisher) PublishMessage(ctx context.Context, vote model.Vote, key string) error {
	vb, err := json.Marshal(vote)
	if err != nil {
		return fmt.Errorf("failed to marshal vote: %v", err)
	}

	msg := kafka.Message{
		Key:   []byte(key), // PollID
		Value: vb,
	}

	if err := kp.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message to kafka: %v", err)
	}

	return nil
}

func (kp *KafkaPublisher) Close() error {
	if err := kp.writer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka writer: %v", err)
	}
	return nil
}
