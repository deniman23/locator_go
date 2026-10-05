// Package messaging messaging/consumer.go
package messaging

import (
	"errors"
	"strings"

	"locator/service"
)

// Consumer отвечает за получение и обработку сообщений из указанной очереди.
type Consumer struct {
	Client    *RabbitMQClient
	QueueName string
}

// NewConsumer создаёт нового Consumer для указанной очереди.
func NewConsumer(client *RabbitMQClient, queueName string) *Consumer {
	return &Consumer{
		Client:    client,
		QueueName: queueName,
	}
}

// Consume читает очередь, пока канал не закроется.
// Битый JSON не возвращается в очередь. Ошибки БД повторяются ограниченно.
func (c *Consumer) Consume(handler func([]byte) error) error {
	msgs, err := c.Client.Channel.Consume(
		c.QueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for msg := range msgs {
		err := handler(msg.Body)
		if err == nil {
			_ = msg.Ack(false)
			continue
		}
		if errors.Is(err, service.ErrPoisonMessage) {
			_ = msg.Nack(false, false)
			continue
		}
		retries := 0
		if raw, ok := msg.Headers["x-death"]; ok {
			if deaths, ok := raw.([]interface{}); ok {
				retries = len(deaths)
			}
		}
		if retries >= 5 || strings.Contains(err.Error(), "poison message") {
			_ = msg.Nack(false, false)
			continue
		}
		_ = msg.Nack(false, true)
	}
	return errors.New("consumer channel closed")
}
