// Package messaging messaging/consumer.go
package messaging

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

// Consume начинает прослушивание очереди и вызывает handler для каждого полученного сообщения.
func (c *Consumer) Consume(handler func([]byte) error) error {
	msgs, err := c.Client.Channel.Consume(
		c.QueueName, // название очереди
		"",          // consumer tag
		false,       // auto-ack (false позволит нам вручную подтверждать сообщение)
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			if err := handler(msg.Body); err != nil {
				// Если обработка не удалась, отправляем nack, чтобы сообщение повторно доставлялось
				msg.Nack(false, true)
			} else {
				// Если всё хорошо, подтверждаем обработку сообщения
				msg.Ack(false)
			}
		}
	}()

	return nil
}
