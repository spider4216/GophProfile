package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spider4216/GophProfile/internal/models"
)

type QueueInterface interface {
	SendUploadEvent(ctx context.Context, e models.AvatarUploadEvent) error
	SendDeleteEvent(ctx context.Context, e models.AvatarDeleteEvent) error
	SendProcessEvent(ctx context.Context, e models.AvatarProcessEvent) error
	DeclareConsumers() error
	DeclareExchange() error
	QueuesBind() error
	DeclareQueues() error
	ConnectClose() error
	CreateCh() (*amqp.Channel, error)
	GetUploadConsumer() <-chan amqp.Delivery
	GetDeleteConsumer() <-chan amqp.Delivery
	GetProcessConsumer() <-chan amqp.Delivery
}
