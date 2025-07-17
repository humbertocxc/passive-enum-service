package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"recon-automation-microservice/config"
	"recon-automation-microservice/modules/domain_processor"
	"recon-automation-microservice/pkg/rabbitmq"
	"recon-automation-microservice/pkg/subfinder"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file (it might not exist or be empty): %v", err)
	}

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

	inputQueue := "domains_to_passive_enum_queue"
	outputQueue := "passive_found_subs_queue"

	err = rbmqClient.DeclareQueue(inputQueue)
	if err != nil {
		log.Fatalf("Failed to declare input queue '%s': %v", inputQueue, err)
	}
	err = rbmqClient.DeclareQueue(outputQueue)
	if err != nil {
		log.Fatalf("Failed to declare output queue '%s': %v", outputQueue, err)
	}

	domainService := domain_processor.NewService()

	subfinderPath := cfg.SubfinderPath
	subfinderTimeout := cfg.SubfinderTimeout

	subfinderService := subfinder.NewService(subfinderPath, subfinderTimeout)

	domainController := domain_processor.NewController(
		rbmqClient,
		domainService,
		subfinderService,
		inputQueue,
		outputQueue,
	)

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
	log.Println("Application stopped.")
}
