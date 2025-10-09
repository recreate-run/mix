package storage

import (
	"fmt"
	"os"
	"strconv"
)

// LoadConfigFromEnv loads storage configuration from environment variables
func LoadConfigFromEnv() Config {
	cfg := Config{
		Type: getEnvOrDefault("STORAGE_TYPE", ProviderTypeLocal),
	}

	// Load common settings
	cfg.Endpoint = getEnvOrDefault("STORAGE_ENDPOINT", "./storage")
	cfg.PublicURLBase = os.Getenv("STORAGE_PUBLIC_URL_BASE")

	// Load Supabase-specific settings
	if cfg.Type == ProviderTypeSupabase {
		cfg.Bucket = os.Getenv("STORAGE_BUCKET")
		cfg.AccessKey = os.Getenv("STORAGE_ACCESS_KEY") // Supabase service_role key
	}

	return cfg
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("storage type is required")
	}

	if c.Type != ProviderTypeLocal && c.Type != ProviderTypeSupabase {
		return fmt.Errorf("invalid storage type: %s (must be 'local' or 'supabase')", c.Type)
	}

	if c.Endpoint == "" {
		return fmt.Errorf("storage endpoint is required")
	}

	// Supabase-specific validation
	if c.Type == ProviderTypeSupabase {
		if c.Bucket == "" {
			return fmt.Errorf("storage bucket is required for Supabase provider")
		}
		if c.AccessKey == "" {
			return fmt.Errorf("storage API key (access_key) is required for Supabase provider")
		}
	}

	return nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defaultValue
}
