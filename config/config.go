package config

import (
	"fmt"
	"os"
)

type Config struct {
	RabbitMQURL string
}

func LoadConfig() (*Config, error) {
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
		fmt.Println("RABBITMQ_URL not set, using default:", rabbitMQURL)
	}

	return &Config{
		RabbitMQURL: rabbitMQURL,
	}, nil
}
