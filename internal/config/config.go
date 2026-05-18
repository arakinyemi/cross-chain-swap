package config

import (
	"os"
	"strconv"
)

type Config struct {
	BitnobClientID      string
	BitnobClientSecret  string
	BitnobBaseURL       string
	DatabaseURL         string
	BitnobWebhookSecret string
	Port                string
	SwapFeeUSDT         float64
	MinSwapAmount       float64
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		BitnobClientID:      os.Getenv("BITNOB_CLIENT_ID"),
		BitnobClientSecret:  os.Getenv("BITNOB_CLIENT_SECRET"),
		BitnobBaseURL:       "https://api.bitnob.com",
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		BitnobWebhookSecret: os.Getenv("BITNOB_WEBHOOK_SECRET"),
		Port:                port,
		SwapFeeUSDT:         parseFloat(os.Getenv("SWAP_FEE_USDT"), 2.0),
		MinSwapAmount:       parseFloat(os.Getenv("MIN_SWAP_AMOUNT"), 10.0),
	}
}

func parseFloat(raw string, fallback float64) float64 {
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}

	return value
}
