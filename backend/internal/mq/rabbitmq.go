package mq

import (
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQManager struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	url       string
	queueName string
}

func NewRabbitMQManager(queueName string) *RabbitMQManager {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
	}
	manager := &RabbitMQManager{
		url:       url,
		queueName: queueName,
	}
	manager.connect()
	return manager
}

func (m *RabbitMQManager) connect() {
	var err error
	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		m.conn, err = amqp.Dial(m.url)
		if err == nil {
			log.Println("RabbitMQ connection established.")
			m.openChannel()
			go m.handleReconnect()
			return
		}
		log.Printf("Failed to connect to RabbitMQ (attempt %d/%d): %v", i, maxRetries, err)
		time.Sleep(5 * time.Second)
	}
	log.Fatalf("Could not connect to RabbitMQ after %d attempts.", maxRetries)
}

func (m *RabbitMQManager) openChannel() {
	var err error
	m.channel, err = m.conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a RabbitMQ channel: %v", err)
	}
	_, err = m.channel.QueueDeclare(m.queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}
}

func (m *RabbitMQManager) handleReconnect() {
	closeChan := make(chan *amqp.Error)
	m.conn.NotifyClose(closeChan)

	err := <-closeChan
	log.Printf("RabbitMQ connection closed: %v. Reconnecting...", err)
	m.connect()
}

func (m *RabbitMQManager) GetChannel() *amqp.Channel {
	// 检查 channel 是否有效
	if m.channel == nil || m.conn.IsClosed() {
		log.Println("RabbitMQ channel is not available. Attempting to reconnect...")
		m.connect()
	}
	return m.channel
}
