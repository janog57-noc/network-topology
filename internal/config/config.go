package config

import (
	"fmt"
	"os"
)

type Config struct {
	NetboxURL   string
	NetboxToken string
}

func Load() (*Config, error) {
	url := os.Getenv("NETBOX_URL")
	token := os.Getenv("NETBOX_TOKEN")

	if url == "" || token == "" {
		return nil, fmt.Errorf("NETBOX_URL and NETBOX_TOKEN must be set")
	}

	return &Config{
		NetboxURL:   url,
		NetboxToken: token,
	}, nil
}
