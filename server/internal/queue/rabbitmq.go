package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"bitly-url/internal/entity"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName = "clicks"
	exchangeType = "direct"
	queueName    = "clicks"
	routingKey   = "click"
	dlxName      = "clicks.dlx"
	dlqName      = "clicks.dlq"

	maxRetries     = 3
	reconnectBase  = 1 * time.Second
	reconnectMax   = 30 * time.Second
	consumerTag    = "click-worker"
	prefetchCount  = 100
)

type RabbitMQClient struct {
	url    string
	conn   *amqp.Connection
	ch     *amqp.Channel
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

func New(url string) (*RabbitMQClient, error) {
	c := &RabbitMQClient{
		url:  url,
		done: make(chan struct{}),
	}
	if err := c.connect(); err != nil {
		return nil, err
	}
	go c.reconnectLoop()
	return c, nil
}

func (c *RabbitMQClient) connect() error {
	conn, err := amqp.DialConfig(c.url, amqp.Config{
		Heartbeat: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}
	if err := ch.Qos(prefetchCount, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to set qos: %w", err)
	}
	if err := c.declareTopology(ch); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare topology: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.ch = ch
	c.closed = false
	c.mu.Unlock()
	return nil
}

func (c *RabbitMQClient) declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		dlxName, exchangeType, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("failed to declare dlx: %w", err)
	}
	if _, err := ch.QueueDeclare(
		dlqName, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("failed to declare dlq: %w", err)
	}
	if err := ch.QueueBind(dlqName, routingKey, dlxName, false, nil); err != nil {
		return fmt.Errorf("failed to bind dlq: %w", err)
	}
	if err := ch.ExchangeDeclare(
		exchangeName, exchangeType, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}
	args := amqp.Table{
		"x-dead-letter-exchange":    dlxName,
		"x-dead-letter-routing-key": routingKey,
	}
	if _, err := ch.QueueDeclare(
		queueName, true, false, false, false, args,
	); err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	if err := ch.QueueBind(queueName, routingKey, exchangeName, false, nil); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}
	return nil
}

func (c *RabbitMQClient) reconnectLoop() {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		notifyClose := conn.NotifyClose(make(chan *amqp.Error))
		err := <-notifyClose
		if err == nil {
			return
		}

		slog.Warn("rabbitmq connection lost, reconnecting", "error", err)

		backoff := reconnectBase
		for {
			time.Sleep(backoff)

			c.mu.Lock()
			closed := c.closed
			c.mu.Unlock()
			if closed {
				return
			}

			if err := c.connect(); err != nil {
				slog.Warn("rabbitmq reconnect failed, retrying", "error", err)
				backoff *= 2
				if backoff > reconnectMax {
					backoff = reconnectMax
				}
				continue
			}
			slog.Info("rabbitmq reconnected")
			break
		}
	}
}

func (c *RabbitMQClient) Publish(ctx context.Context, click *entity.Click) error {
	c.mu.Lock()
	ch := c.ch
	c.mu.Unlock()

	if ch == nil {
		return fmt.Errorf("rabbitmq not connected")
	}

	body, err := json.Marshal(click)
	if err != nil {
		return fmt.Errorf("failed to marshal click: %w", err)
	}

	return ch.PublishWithContext(ctx, exchangeName, routingKey, true, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (c *RabbitMQClient) Consume(ctx context.Context) (<-chan *entity.Click, error) {
	c.mu.Lock()
	ch := c.ch
	c.mu.Unlock()

	if ch == nil {
		return nil, fmt.Errorf("rabbitmq not connected")
	}

	deliveries, err := ch.ConsumeWithContext(ctx, queueName, consumerTag, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to consume: %w", err)
	}

	clickCh := make(chan *entity.Click)
	go func() {
		defer close(clickCh)

		for d := range deliveries {
			var click entity.Click
			if err := json.Unmarshal(d.Body, &click); err != nil {
				slog.Error("failed to unmarshal click", "error", err)
				continue
			}

			select {
			case clickCh <- &click:
			case <-ctx.Done():
				return
			}

			if err := d.Ack(false); err != nil {
				slog.Error("failed to ack message", "error", err)
			}
		}
	}()

	return clickCh, nil
}

func (c *RabbitMQClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	close(c.done)

	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}
