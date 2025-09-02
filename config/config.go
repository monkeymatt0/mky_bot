package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config contiene tutte le configurazioni dell'applicazione
type Config struct {
	Bybit    BybitConfig
	LogLevel string
}

// BybitConfig contiene le configurazioni per Bybit
type BybitConfig struct {
	APIKey    string
	SecretKey string
}

// Load carica le configurazioni dalle variabili d'ambiente
func Load() (*Config, error) {
	// Carica il file .env se esiste
	_ = godotenv.Load()

	config := &Config{
		Bybit: BybitConfig{
			APIKey:    os.Getenv("BYBIT_API_KEY"),
			SecretKey: os.Getenv("BYBIT_SECRET_KEY"),
		},
		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),
	}

	return config, nil
}

// getEnvOrDefault restituisce il valore della variabile d'ambiente o un valore di default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
