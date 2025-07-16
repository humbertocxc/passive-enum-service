package domain_processor

import (
	"context"
	"encoding/json"
	"log"
	"recon-automation-microservice/pkg/rabbitmq"
	"time"
)

type Controller struct {
	rbmqClient    *rabbitmq.Client
	domainService *Service
	inputQueue    string
	outputQueue   string
}

func NewController(rbmqClient *rabbitmq.Client, domainService *Service, inputQueue, outputQueue string) *Controller {
	return &Controller{
		rbmqClient:    rbmqClient,
		domainService: domainService,
		inputQueue:    inputQueue,
		outputQueue:   outputQueue,
	}
}

type queueEnvelope struct {
	Pattern string `json:"pattern"`
	Data    string `json:"data"`
}

func (c *Controller) StartConsuming(ctx context.Context) {
	msgs, err := c.rbmqClient.Consume(c.inputQueue)
	if err != nil {
		log.Printf("Error consuming from queue %s: %v", c.inputQueue, err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer context cancelled, stopping...")
			return
		case d, ok := <-msgs:
			if !ok {
				log.Println("RabbitMQ channel closed, consumer stopping.")
				return
			}

			var env queueEnvelope
			if err := json.Unmarshal(d.Body, &env); err != nil {
				log.Printf("Failed to unmarshal envelope: %v", err)
				d.Nack(false, false)
				continue
			}

			var msg DomainMessage
			if err := json.Unmarshal([]byte(env.Data), &msg); err != nil {
				log.Printf("Failed to unmarshal data field: %v", err)
				d.Nack(false, false)
				continue
			}

			log.Printf("Received message from '%s': value=%s, companyId=%s", c.inputQueue, msg.Value, msg.CompanyID)

			mockedSubdomain := c.domainService.ProcessDomain(msg)

			response := struct {
				Value     string `json:"value"`
				CompanyID string `json:"companyId"`
			}{
				Value:     mockedSubdomain,
				CompanyID: msg.CompanyID,
			}
			responseData, err := json.Marshal(response)
			if err != nil {
				log.Printf("Failed to marshal response data: %v", err)
				d.Nack(false, false)
				continue
			}

			envelope := queueEnvelope{
				Pattern: c.outputQueue,
				Data:    string(responseData),
			}
			envelopeBytes, err := json.Marshal(envelope)
			if err != nil {
				log.Printf("Failed to marshal envelope: %v", err)
				d.Nack(false, false)
				continue
			}
			publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = c.rbmqClient.Publish(publishCtx, "", c.outputQueue, envelopeBytes)
			cancel()

			if err != nil {
				log.Printf("Failed to publish message to '%s': %v", c.outputQueue, err)
				d.Nack(false, true)
			} else {
				log.Printf("Published message to '%s': %s", c.outputQueue, string(envelopeBytes))
				d.Ack(false)
			}
		}
	}
}
