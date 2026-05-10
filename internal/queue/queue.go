package queue

import (
	"encoding/json"
	"errors"
)

type PurchaseRequest struct {
	ItemID string `json:"item_id"`
	UserID string `json:"user_id"`
}

// QueueInterface abstracts different queue implementations (in-memory, Kafka, etc.).
type QueueInterface interface {
	Publish(PurchaseRequest) error
	Consume() <-chan PurchaseRequest
}

// Queue is a simple buffered channel-backed queue used for local testing.
type Queue struct {
	ch chan PurchaseRequest
}

// New creates a queue with the given buffer size.
func New(buffer int) *Queue {
	return &Queue{ch: make(chan PurchaseRequest, buffer)}
}

// Publish enqueues a purchase request. Returns error if queue is full.
func (q *Queue) Publish(r PurchaseRequest) error {
	select {
	case q.ch <- r:
		return nil
	default:
		return errors.New("queue full")
	}
}

// Consume returns a read-only channel to receive published requests.
func (q *Queue) Consume() <-chan PurchaseRequest {
	return q.ch
}

// kafkaPayload helper for marshaling into Kafka messages.
func kafkaPayload(r PurchaseRequest) ([]byte, error) {
	return json.Marshal(r)
}
