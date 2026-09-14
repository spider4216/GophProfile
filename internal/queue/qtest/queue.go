package qtest

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spider4216/GophProfile/internal/models"
)

type QueueSlice struct {
	data   map[string][][]byte
	logger *slog.Logger
}

func NewQueue(logger *slog.Logger, store map[string][][]byte) *QueueSlice {
	return &QueueSlice{
		logger: logger,
		data:   store,
	}
}

func (q *QueueSlice) SendUploadEvent(ctx context.Context, e models.AvatarUploadEvent) error {
	b, err := json.Marshal(e)

	if err != nil {
		return err
	}

	q.data["uploads"] = append(q.data["uploads"], b)

	return nil
}

func (q *QueueSlice) SendDeleteEvent(ctx context.Context, e models.AvatarDeleteEvent) error {
	return nil
}

func (q *QueueSlice) SendProcessEvent(ctx context.Context, e models.AvatarProcessEvent) error {
	return nil
}

func (q *QueueSlice) DeclareConsumers() error {
	return nil
}

func (q *QueueSlice) DeclareExchange() error {
	return nil
}

func (q *QueueSlice) QueuesBind() error {
	return nil
}

func (q *QueueSlice) DeclareQueues() error {
	return nil
}

func (q *QueueSlice) ConnectClose() error {
	return nil
}

func (q *QueueSlice) CreateCh() (*amqp.Channel, error) {
	return nil, nil
}

func (q *QueueSlice) GetUploadConsumer() <-chan amqp.Delivery {
	return nil
}

func (q *QueueSlice) GetDeleteConsumer() <-chan amqp.Delivery {
	return nil
}

func (q *QueueSlice) GetProcessConsumer() <-chan amqp.Delivery {
	return nil
}
