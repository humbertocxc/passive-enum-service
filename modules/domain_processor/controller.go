package domain_processor

import (
	"context"
	"fmt"
	"log"
	"time"

	"recon-automation-microservice/pkg/rabbitmq"
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

			domain := string(d.Body)
			log.Printf("Received message from '%s': %s", c.inputQueue, domain)

			mockedSubdomain := c.domainService.ProcessDomain(domain)
			responseMessage := fmt.Sprintf("Original: %s, Mocked Subdomain: %s", domain, mockedSubdomain)

			publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := c.rbmqClient.Publish(publishCtx, "", c.outputQueue, []byte(responseMessage))
			cancel()

			if err != nil {
				log.Printf("Failed to publish message to '%s': %v", c.outputQueue, err)
				d.Nack(false, true)
			} else {
				log.Printf("Published message to '%s': %s", c.outputQueue, responseMessage)
				d.Ack(false)
			}
		}
	}
}
