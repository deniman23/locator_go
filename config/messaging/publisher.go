package messaging

import (
	"encoding/json"
	"github.com/rabbitmq/amqp091-go"
)

// Publisher отвечает за публикацию сообщений в RabbitMQ.
type Publisher struct {
	Client     *RabbitMQClient
	Exchange   string
	RoutingKey string
}

// NewPublisher создаёт новый Publisher, используя готовый клиент, обмен и ключ маршрутизации.
func NewPublisher(client *RabbitMQClient, exchange, routingKey string) *Publisher {
	return &Publisher{
		Client:     client,
		Exchange:   exchange,
		RoutingKey: routingKey,
	}
}

// Publish отправляет message (например, JSON-сериализованное событие) в очередь.
func (p *Publisher) Publish(message []byte) error {
	return p.Client.Channel.Publish(
		p.Exchange,   // обмен
		p.RoutingKey, // ключ маршрутизации
		false,        // mandatory
		false,        // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)
}

// PublishJSON сериализует объект в JSON и публикует его.
func (p *Publisher) PublishJSON(v interface{}) error {
	message, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return p.Publish(message)
}
