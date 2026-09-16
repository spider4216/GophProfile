package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spider4216/GophProfile/internal/models"
)

type NoConfirmErr struct{}

func (err NoConfirmErr) Error() string {
	return "cannot confirm publish in queue"
}

type queueName string

func (qn queueName) String() string {
	return string(qn)
}

const (
	uploadQueueName  queueName = "upload"
	deleteQueueName  queueName = "delete"
	processQueueName queueName = "process"
	exchange         string    = "events"
)

type Queue struct {
	conn            *amqp.Connection
	logger          *slog.Logger
	uploadQueue     *amqp.Queue
	processQueue    *amqp.Queue
	deleteQueue     *amqp.Queue
	uploadConsumer  <-chan amqp.Delivery
	deleteConsumer  <-chan amqp.Delivery
	processConsumer <-chan amqp.Delivery
}

func NewQueue(dsn string, logger *slog.Logger) (*Queue, error) {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		logger.Error("cannot connect to rabbit", "error", err)
		return nil, err
	}

	logger.Debug("connected to rabbitmq", "dsn", dsn)

	return &Queue{
		logger: logger,
		conn:   conn,
	}, nil
}

func (q *Queue) GetUploadConsumer() <-chan amqp.Delivery {
	return q.uploadConsumer
}

func (q *Queue) GetDeleteConsumer() <-chan amqp.Delivery {
	return q.deleteConsumer
}

func (q *Queue) GetProcessConsumer() <-chan amqp.Delivery {
	return q.processConsumer
}

func (q *Queue) SendUploadEvent(ctx context.Context, e models.AvatarUploadEvent) error {
	ch, err := q.CreateCh()
	if err != nil {
		return fmt.Errorf("cannot create ch for upload event: %w", err)
	}

	defer func() {
		if err := ch.Close(); err != nil {
			q.logger.Warn("cannot close channel", "error", err)
		}
	}()

	return sendEvent(ctx, e, q.uploadQueue.Name, ch)
}

func (q *Queue) SendDeleteEvent(ctx context.Context, e models.AvatarDeleteEvent) error {
	ch, err := q.CreateCh()
	if err != nil {
		return fmt.Errorf("cannot create ch for delete event: %w", err)
	}

	defer func() {
		if err := ch.Close(); err != nil {
			q.logger.Warn("cannot close channel", "error", err)
		}
	}()

	return sendEvent(ctx, e, q.deleteQueue.Name, ch)
}

func (q *Queue) SendProcessEvent(ctx context.Context, e models.AvatarProcessEvent) error {
	ch, err := q.CreateCh()
	if err != nil {
		return fmt.Errorf("cannot create ch for process event: %w", err)
	}

	defer func() {
		if err := ch.Close(); err != nil {
			q.logger.Warn("cannot close channel", "error", err)
		}
	}()

	return sendEvent(ctx, e, q.processQueue.Name, ch)
}

func sendEvent[T any](ctx context.Context, e T, queue string, ch *amqp.Channel) error {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("cannot marshal event payload: %w", err)
	}

	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("cannot enable publisher confirms: %w", err)
	}

	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(
		publishCtx,
		exchange, // exchange пока default
		queue,    // routingKey
		false,
		false,
		amqp.Publishing{
			Body:         b,               // тело сообщения
			DeliveryMode: amqp.Persistent, // сообщение будет сохранено на диск
		},
	)
	if err != nil {
		return fmt.Errorf("cannot publish in queue %s: %w", queue, err)
	}

	// На базовом уровне будем ждать подтверждения
	select {
	case c := <-confirms:
		if c.Ack {
			return nil
		} else {
			return NoConfirmErr{}
		}
	case <-ctx.Done():
		return errors.New("context was canceled while publishing msg in queue")
	}
}

func (q *Queue) DeclareConsumers() error {
	uplC, err := q.declareConsumer(uploadQueueName)
	if err != nil {
		return fmt.Errorf("cannot declare upload consumer: %w", err)
	}

	delC, err := q.declareConsumer(deleteQueueName)
	if err != nil {
		return fmt.Errorf("cannot declare delete consumer: %w", err)
	}

	proc, err := q.declareConsumer(processQueueName)
	if err != nil {
		return fmt.Errorf("cannot declare process consumer: %w", err)
	}

	q.uploadConsumer = uplC
	q.deleteConsumer = delC
	q.processConsumer = proc

	return nil
}

func (q *Queue) DeclareExchange() error {
	ch, err := q.CreateCh()
	if err != nil {
		return fmt.Errorf("cannot decalre excahnge events")
	}

	defer func() {
		if err := ch.Close(); err != nil {
			q.logger.Warn("cannot close channel", "error", err)
		}
	}()

	return ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
}

func (q *Queue) QueuesBind() error {
	ch, err := q.CreateCh()
	if err != nil {
		return fmt.Errorf("cannot create ch in queue bind: %w", err)
	}

	defer func() {
		if err := ch.Close(); err != nil {
			q.logger.Warn("cannot close channel", "error", err)
		}
	}()

	if err := ch.QueueBind(uploadQueueName.String(), uploadQueueName.String(), exchange, false, nil); err != nil {
		return fmt.Errorf("cannot bind upload key to exchage: %w", err)
	}

	if err := ch.QueueBind(deleteQueueName.String(), deleteQueueName.String(), exchange, false, nil); err != nil {
		return fmt.Errorf("cannot bind delete key to exchage: %w", err)
	}

	if err := ch.QueueBind(processQueueName.String(), processQueueName.String(), exchange, false, nil); err != nil {
		return fmt.Errorf("cannot bind process key to exchage: %w", err)
	}

	return nil
}

func (q *Queue) DeclareQueues() error {
	uplQ, err := q.declareQueue(uploadQueueName)
	if err != nil {
		return fmt.Errorf("cannot declare upload queue: %w", err)
	}

	delQ, err := q.declareQueue(deleteQueueName)
	if err != nil {
		return fmt.Errorf("cannot declare delete queue: %w", err)
	}

	procQ, err := q.declareQueue(processQueueName)
	if err != nil {
		return fmt.Errorf("cannot declare process queue: %w", err)
	}

	q.uploadQueue = uplQ
	q.deleteQueue = delQ
	q.processQueue = procQ

	return nil
}

func (q *Queue) ConnectClose() error {
	return q.conn.Close()
}

func (q *Queue) CreateCh() (*amqp.Channel, error) {
	return q.conn.Channel()
}

func (q *Queue) declareQueue(name queueName) (*amqp.Queue, error) {
	ch, err := q.CreateCh()
	if err != nil {
		return nil, fmt.Errorf("cannot create channel while declare %s queue: %w", name, err)
	}

	defer func() {
		if err := ch.Close(); err != nil {
			q.logger.Warn("cannot close channel", "error", err)
		}
	}()

	myq, err := ch.QueueDeclare(
		name.String(), // имя очереди
		true,          // durable очередь будет сохранена на диск
		false,         // autoDelete не удалять при отсутствии потребителей
		false,         // exclusive очередь доступна другим соединениям
		false,         // noWait ждать подтверждения от сервера
		nil,           // args дополнительные аргументы
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create %s queue: %w", name, err)
	}

	return &myq, nil
}

func (q *Queue) declareConsumer(name queueName) (<-chan amqp.Delivery, error) {
	ch, err := q.CreateCh()
	if err != nil {
		return nil, fmt.Errorf("cannot create channel while declare %s consumer: %w", name, err)
	}

	return ch.Consume(
		name.String(),
		name.String(),
		false,
		false,
		false,
		false,
		nil,
	)
}
