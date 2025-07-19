package domain_processor

import (
	"context"
	"encoding/json"
	"log"
	"recon-automation-microservice/pkg/rabbitmq"
	"recon-automation-microservice/pkg/subfinder"
	"time"
)

type DomainMessage struct {
	Value     string `json:"value"`
	CompanyID string `json:"companyId"`
}

type queueEnvelope struct {
	Pattern string          `json:"pattern"`
	Data    json.RawMessage `json:"data"` // raw JSON instead of string
}

type Controller struct {
	rbmqClient       *rabbitmq.Client
	domainService    *Service
	subfinderService subfinder.Service
	inputQueue       string
	outputQueue      string
}

func NewController(
	rbmqClient *rabbitmq.Client,
	domainService *Service,
	subfinderService subfinder.Service,
	inputQueue, outputQueue string,
) *Controller {
	return &Controller{
		rbmqClient:       rbmqClient,
		domainService:    domainService,
		subfinderService: subfinderService,
		inputQueue:       inputQueue,
		outputQueue:      outputQueue,
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

			var env queueEnvelope
			if err := json.Unmarshal(d.Body, &env); err != nil {
				log.Printf("Failed to unmarshal envelope: %v", err)
				d.Nack(false, false)
				continue
			}

			var msg DomainMessage
			if err := json.Unmarshal(env.Data, &msg); err != nil {
				log.Printf("Failed to unmarshal data field for domain: %v", err)
				d.Nack(false, false)
				continue
			}

			log.Printf("Received message from '%s': value=%s, companyId=%s", c.inputQueue, msg.Value, msg.CompanyID)

			discoveredSubdomains, err := c.subfinderService.FindSubdomains(ctx, msg.Value)
			if err != nil {
				log.Printf("Error running Subfinder for domain %s: %v", msg.Value, err)
				d.Nack(false, false)
				continue
			}
			log.Printf("Discovered %d subdomains for %s using Subfinder.", len(discoveredSubdomains), msg.Value)

			anySubdomainPublishFailed := false

			for _, subdomain := range discoveredSubdomains {
				response := DomainMessage{
					Value:     subdomain,
					CompanyID: msg.CompanyID,
				}
				responseData, err := json.Marshal(response)
				if err != nil {
					log.Printf("Failed to marshal response data for subdomain %s (CompanyID: %s): %v", subdomain, msg.CompanyID, err)
					anySubdomainPublishFailed = true
					continue
				}

				envelope := queueEnvelope{
					Pattern: c.outputQueue,
					Data:    responseData, // now just raw JSON, not stringified
				}
				envelopeBytes, err := json.Marshal(envelope)
				if err != nil {
					log.Printf("Failed to marshal envelope for subdomain %s (CompanyID: %s): %v", subdomain, msg.CompanyID, err)
					anySubdomainPublishFailed = true
					continue
				}

				publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err = c.rbmqClient.Publish(publishCtx, "", c.outputQueue, envelopeBytes)
				cancel()

				if err != nil {
					log.Printf("Failed to publish subdomain %s to '%s': %v", subdomain, c.outputQueue, err)
					anySubdomainPublishFailed = true
				} else {
					log.Printf("Published subdomain '%s' for CompanyID '%s' to '%s'", subdomain, msg.CompanyID, c.outputQueue)
				}
			}

			if anySubdomainPublishFailed {
				log.Printf("Some subdomains failed to publish for original domain '%s'. Nacking original message with requeue.", msg.Value)
				d.Nack(false, true)
			} else {
				log.Printf("All discovered subdomains for original domain '%s' published successfully. Acknowledging original message.", msg.Value)
				d.Ack(false)
			}
		}
	}
}
