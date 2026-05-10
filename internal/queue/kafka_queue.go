package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaQueue implements QueueInterface using kafka-go.
type KafkaQueue struct {
	writer *kafka.Writer
	ch     chan PurchaseRequest
	ctx    context.Context
}

// NewKafka creates a Kafka-backed queue. brokers should be like []string{"localhost:9092"}.
func NewKafka(brokers []string, topic string) *KafkaQueue {
	ctx := context.Background()
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
	}
	q := &KafkaQueue{
		writer: w,
		ch:     make(chan PurchaseRequest, 1000),
		ctx:    ctx,
	}

	// Start a reader that consumes the topic and pushes to the channel.
	go func() {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: "flashguard-group",
		})
		defer r.Close()
		for {
			m, err := r.ReadMessage(ctx)
			if err != nil {
				log.Printf("kafka reader error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			var req PurchaseRequest
			if err := json.Unmarshal(m.Value, &req); err != nil {
				log.Printf("kafka: invalid message: %v", err)
				continue
			}
			q.ch <- req
		}
	}()

	return q
}

// Publish sends a PurchaseRequest to Kafka.
func (k *KafkaQueue) Publish(r PurchaseRequest) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	msg := kafka.Message{Value: b}
	return k.writer.WriteMessages(k.ctx, msg)
}

// Consume returns a read-only channel of PurchaseRequest received from Kafka.
func (k *KafkaQueue) Consume() <-chan PurchaseRequest {
	return k.ch
}
