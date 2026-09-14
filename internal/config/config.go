package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds all runtime configuration for ha-go-trmnl.
type Config struct {
	// Home Assistant
	HAURL     string
	HAToken   string
	HAEntities []string

	// Server
	ListenAddr  string
	BaseURL     string
	RefreshRate int

	// Display
	Title       string
	Columns     int
}

func Load() (*Config, error) {
	c := &Config{}

	flag.StringVar(&c.HAURL, "ha-url", env("HA_URL", "http://homeassistant.local:8123"), "Home Assistant base URL")
	flag.StringVar(&c.HAToken, "ha-token", env("HA_TOKEN", ""), "Home Assistant long-lived access token")
	entities := flag.String("ha-entities", env("HA_ENTITIES", ""), "Comma-separated entity IDs to display")
	flag.StringVar(&c.ListenAddr, "listen", env("LISTEN_ADDR", ":8080"), "HTTP listen address")
	flag.StringVar(&c.BaseURL, "base-url", env("BASE_URL", "http://localhost:8080"), "Publicly accessible base URL")
	flag.IntVar(&c.RefreshRate, "refresh", envInt("REFRESH_RATE", 60), "Display refresh interval in seconds")
	flag.StringVar(&c.Title, "title", env("DISPLAY_TITLE", "Home Assistant"), "Title shown on display")
	flag.IntVar(&c.Columns, "columns", envInt("DISPLAY_COLUMNS", 2), "Number of columns for entity layout (1 or 2)")
	flag.Parse()

	if c.HAURL == "" {
		return nil, fmt.Errorf("HA_URL or -ha-url is required")
	}
	if c.HAToken == "" {
		return nil, fmt.Errorf("HA_TOKEN or -ha-token is required")
	}
	if *entities == "" {
		return nil, fmt.Errorf("HA_ENTITIES or -ha-entities is required (comma-separated entity IDs)")
	}
	for _, e := range strings.Split(*entities, ",") {
		e = strings.TrimSpace(e)
		if e != "" {
			c.HAEntities = append(c.HAEntities, e)
		}
	}
	if c.Columns < 1 || c.Columns > 2 {
		c.Columns = 2
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var i int
	if _, err := fmt.Sscan(v, &i); err != nil {
		return def
	}
	return i
}
