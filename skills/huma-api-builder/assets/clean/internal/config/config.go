package config

import (
	"os"
)

type Config struct {
	APIName        string
	APIVersion     string
	APIDescription string
	Environment    string
}

func Load() (*Config, error) {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	return &Config{
		APIName:        "My API",
		APIVersion:     "1.0.0",
		APIDescription: "A clean architecture HUMA API",
		Environment:    env,
	}, nil
}
