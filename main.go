package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"recon-automation-microservice/config"
	"recon-automation-microservice/modules/domain_processor"
	"recon-automation-microservice/pkg/rabbitmq"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	rbmqClient, err := rabbitmq.NewClient(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rbmqClient.Close()

	log.Println("Connected to RabbitMQ successfully!")

	inputQueue := "domains_input"
	outputQueue := "subdomains_output"

	err = rbmqClient.DeclareQueue(inputQueue)
	if err != nil {
		log.Fatalf("Failed to declare input queue '%s': %v", inputQueue, err)
	}
	err = rbmqClient.DeclareQueue(outputQueue)
	if err != nil {
		log.Fatalf("Failed to declare output queue '%s': %v", outputQueue, err)
	}

	domainService := domain_processor.NewService()
	domainController := domain_processor.NewController(rbmqClient, domainService, inputQueue, outputQueue)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		log.Printf("Starting consumer for queue: %s", inputQueue)
		domainController.StartConsuming(ctx)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	cancel()
	rbmqClient.Close()
	log.Println("Application stopped.")
}
