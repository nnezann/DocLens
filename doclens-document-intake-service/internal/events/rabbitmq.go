package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/doclens/document-intake-service/internal/store"
	"github.com/rabbitmq/amqp091-go"
)

type RabbitPublisher struct {
	connection *amqp091.Connection
	channel    *amqp091.Channel
	exchange   string
	logger     *slog.Logger
}

func NewRabbitPublisher(url, exchange string, logger *slog.Logger) (*RabbitPublisher, error) {
	connection, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}
	channel, err := connection.Channel()
	if err != nil {
		connection.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}
	if err := channel.ExchangeDeclare(exchange, amqp091.ExchangeTopic, true, false, false, false, nil); err != nil {
		channel.Close()
		connection.Close()
		return nil, fmt.Errorf("declare rabbitmq exchange: %w", err)
	}
	return &RabbitPublisher{connection: connection, channel: channel, exchange: exchange, logger: logger}, nil
}

func (p *RabbitPublisher) Close() error {
	channelErr := p.channel.Close()
	connectionErr := p.connection.Close()
	if channelErr != nil {
		return channelErr
	}
	return connectionErr
}

func (p *RabbitPublisher) Publish(ctx context.Context, record store.OutboxRecord) error {
	return p.channel.PublishWithContext(ctx, p.exchange, record.RoutingKey, false, false, amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		MessageId:    record.ID,
		Timestamp:    time.Now().UTC(),
		Body:         record.Payload,
	})
}

func RunOutboxPublisher(ctx context.Context, metadata *store.PostgresStore, publisher *RabbitPublisher, logger *slog.Logger) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := publishOutboxBatch(ctx, metadata, publisher, logger); err != nil {
			logger.Error("publish outbox batch", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func publishOutboxBatch(ctx context.Context, metadata *store.PostgresStore, publisher *RabbitPublisher, logger *slog.Logger) error {
	records, err := metadata.ClaimOutbox(ctx, 50)
	if err != nil {
		return err
	}
	for _, record := range records {
		publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := publisher.Publish(publishCtx, record)
		cancel()
		if err == nil {
			err = metadata.MarkOutboxPublished(ctx, record.ID, time.Now().UTC())
		}
		if err != nil {
			backoff := time.Duration(1<<min(record.AttemptCount, 6)) * time.Second
			if markErr := metadata.MarkOutboxFailed(ctx, record.ID, err.Error(), time.Now().UTC().Add(backoff)); markErr != nil {
				logger.Error("mark outbox event failed", "event_id", record.ID, "error", markErr)
			}
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
