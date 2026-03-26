package config

import (
	"fmt"
	"os"
)

type Config struct {
	Env         string
	APIPort     string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	OpenAIKey   string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:         getEnv("ENV", "development"),
		APIPort:     getEnv("API_PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL": c.DatabaseURL,
		"REDIS_URL":    c.RedisURL,
		"JWT_SECRET":   c.JWTSecret,
	}
	for key, val := range required {
		if val == "" {
			return fmt.Errorf("required env var %s is not set", key)
		}
	}
	return nil
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
