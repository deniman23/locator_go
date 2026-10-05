// Package messaging Package config messaging/rabbitmq_client.go
package messaging

import (
	"github.com/rabbitmq/amqp091-go"
)

const (
	LocationEventsQueue = "location_events"
	LocationEventsDLQ   = "location_events_dlq"
)

// RabbitMQClient отвечает за установку соединения и создание канала для работы с RabbitMQ.
type RabbitMQClient struct {
	URL     string
	Conn    *amqp091.Connection
	Channel *amqp091.Channel
}

// NewRabbitMQClient устанавливает соединение с брокером и возвращает экземпляр RabbitMQClient.
func NewRabbitMQClient(url string) (*RabbitMQClient, error) {
	client := &RabbitMQClient{URL: url}
	if err := client.Reconnect(); err != nil {
		return nil, err
	}
	return client, nil
}

// Reconnect открывает новое соединение. URL в лог не пишется.
func (c *RabbitMQClient) Reconnect() error {
	if c.Channel != nil {
		_ = c.Channel.Close()
	}
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
	conn, err := amqp091.Dial(c.URL)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	c.Conn = conn
	c.Channel = ch
	return nil
}

// EnsureLocationQueue объявляет очередь визитов с DLQ.
// Если очередь уже существует с другими аргументами и в ней нет сообщений, она пересоздаётся.
// Непустую очередь не удаляем: битые сообщения тогда просто не возвращаются в неё.
func (c *RabbitMQClient) EnsureLocationQueue() error {
	if c.Conn != nil {
		if probe, err := c.Conn.Channel(); err == nil {
			q, inspectErr := probe.QueueInspect(LocationEventsQueue)
			_ = probe.Close()
			if inspectErr == nil && q.Messages == 0 && c.Channel != nil {
				_, _ = c.Channel.QueueDelete(LocationEventsQueue, false, false, false)
			}
		}
	}
	if c.Channel == nil {
		return amqp091.ErrClosed
	}
	if _, err := c.Channel.QueueDeclare(LocationEventsDLQ, true, false, false, false, nil); err != nil {
		return err
	}
	args := amqp091.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": LocationEventsDLQ,
	}
	if _, err := c.Channel.QueueDeclare(LocationEventsQueue, true, false, false, false, args); err != nil {
		if recErr := c.Reconnect(); recErr != nil {
			return err
		}
		_, err = c.Channel.QueueDeclare(LocationEventsQueue, true, false, false, false, nil)
		return err
	}
	return nil
}

// Close корректно закрывает канал и соединение.
func (c *RabbitMQClient) Close() {
	if c.Channel != nil {
		err := c.Channel.Close()
		if err != nil {
			return
		}
	}
	if c.Conn != nil {
		err := c.Conn.Close()
		if err != nil {
			return
		}
	}
}
