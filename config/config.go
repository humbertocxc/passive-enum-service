package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	RabbitMQURL      string        `env:"RABBITMQ_URL,required"`
	SubfinderPath    string        `env:"SUBFINDER_PATH"`
	SubfinderTimeout time.Duration `env:"SUBFINDER_TIMEOUT" envDefault:"5m"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		RabbitMQURL:      os.Getenv("RABBITMQ_URL"),
		SubfinderPath:    os.Getenv("SUBFINDER_PATH"),
		SubfinderTimeout: 2 * time.Minute,
	}

	if timeoutStr := os.Getenv("SUBFINDER_TIMEOUT"); timeoutStr != "" {
		parsedTimeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SUBFINDER_TIMEOUT format: %w", err)
		}
		cfg.SubfinderTimeout = parsedTimeout
	}

	if cfg.RabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL environment variable is required")
	}

	return cfg, nil
}