package clients

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/rabbitmq/amqp091-go"
)

var Publisher *RMQClient

type RMQClient struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

type CommandMessage struct {
	Command string      `json:"command"`
	Payload interface{} `json:"payload"`
}

// InitRabbitMQ connects to RabbitMQ and declares queues
func InitRabbitMQ() {
	url := config.Env.RabbitMQURL
	conn, err := amqp091.Dial(url)
	if err != nil {
		log.Fatalf("RabbitMQ: Failed to connect: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("RabbitMQ: Failed to open channel: %v", err)
	}

	// declare the queue
	_, err = ch.QueueDeclare(
		"cmd_queue", // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusve
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		log.Fatalf("RabbitMQ: Failed to declare queue: %v", err)
	}

	Publisher = &RMQClient{conn: conn, channel: ch}
	log.Println("RabbitMQ initialized and cmd_queue declared")
}

// PublishCommand sends a command and its JSON payload to the queue
func (r *RMQClient) PublishCommad(command string, payload interface{}) error {
	msg := CommandMessage{
		Command: command,
		Payload: payload,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// use context with timeout for modern amqp091 publishing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = r.channel.PublishWithContext(
		ctx,
		"",          // exchange
		"cmd_queue", // mandatory
		false,       // immediate
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent, // survive broker restarts
			Body:         body,
		})
	if err != nil {
		log.Printf("Failed to publish %s: %v", command, err)
		return err
	}

	log.Printf("Published command: %s", command)
	return nil
}

func (r *RMQClient) Close() {
	r.channel.Close()
	r.conn.Close()
}
