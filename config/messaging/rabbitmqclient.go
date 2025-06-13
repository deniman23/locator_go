// Package messaging Package config messaging/rabbitmq_client.go
package messaging

import (
	"github.com/rabbitmq/amqp091-go"
)

// RabbitMQClient отвечает за установку соединения и создание канала для работы с RabbitMQ.
type RabbitMQClient struct {
	Conn    *amqp091.Connection
	Channel *amqp091.Channel
}

// NewRabbitMQClient устанавливает соединение с брокером и возвращает экземпляр RabbitMQClient.
func NewRabbitMQClient(url string) (*RabbitMQClient, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return &RabbitMQClient{
		Conn:    conn,
		Channel: ch,
	}, nil
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
