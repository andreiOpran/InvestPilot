package clients

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"

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

	var conn *amqp091.Connection
	var err error
	maxRetries := 10

	// connection retry with backoff
	for i := 1; i <= maxRetries; i++ {
		conn, err = amqp091.Dial(url)
		if err == nil {
			break
		}
		log.Printf("[ERROR-PUBLISHER] RabbitMQ: Failed to connect (attempt %d/%d): %v", i, maxRetries, err)
		time.Sleep(time.Duration(i*2) * time.Second)
	}

	if err != nil {
		log.Fatalf("[ERROR-PUBLISHER] RabbitMQ: Failed to connect after %d attempts: %v", maxRetries, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("[ERROR-PUBLISHER] RabbitMQ: Failed to open channel: %v", err)
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
		log.Fatalf("[ERROR-PUBLISHER] RabbitMQ: Failed to declare queue: %v", err)
	}

	Publisher = &RMQClient{conn: conn, channel: ch}
	log.Println("[PUBLISHER] RabbitMQ initialized and cmd_queue declared")
}

// PublishCommand sends a command and its JSON payload to the queue
func (r *RMQClient) PublishCommand(command string, payload interface{}) error {
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
		log.Printf("[ERROR-PUBLISHER] Failed to publish %s: %v", command, err)
		return err
	}

	log.Printf("[PUBLISHER] Published command: %s", command)
	return nil
}

// PublishRPC sends a command and blocks until decisional node responds on the reply_to queue
func (r *RMQClient) PublishRPC(command string, payload interface{}) ([]byte, error) {
	// declare exclusive temporary reply queue
	q, err := r.channel.QueueDeclare(
		"",    // name
		false, // durable
		false, // auto-deletion
		true,  // exclusive
		false, // noWait
		nil,   // table
	)
	if err != nil {
		return nil, err
	}

	// consume from reply queue
	msgs, err := r.channel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // autoAck
		false,  // exclusive
		false,  // noLocal
		false,  // noWait
		nil,    // table
	)
	if err != nil {
		return nil, err
	}

	corrId := uuid.New().String()

	msg := CommandMessage{Command: command, Payload: payload}
	body, _ := json.Marshal(msg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30s timeout for RPC
	defer cancel()

	// publish with ReplyTo and CorrelationID
	err = r.channel.PublishWithContext(
		ctx,         // context
		"",          // exchange
		"cmd_queue", // key
		false,       // mandatory
		false,       // immediate
		amqp091.Publishing{
			ContentType:   "application/json",
			CorrelationId: corrId,
			ReplyTo:       q.Name,
			Body:          body,
		}, // msg
	)
	if err != nil {
		return nil, err
	}

	// wait for correlation ID match
	for {
		select {
		case <-ctx.Done():
			return nil, context.DeadlineExceeded
		case d := <-msgs:
			if d.CorrelationId == corrId {
				return d.Body, nil
			}
		}
	}
}

func (r *RMQClient) Close() {
	r.channel.Close()
	r.conn.Close()
}
