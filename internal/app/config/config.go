package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTP struct {
		Port int
	}
	GRPC struct {
		Port int
	}
	DB struct {
		Url string
	}
}

// при указывании файла делаем overload environment из файла
// итоговый конфигурации вычисляются следующим приоритетом env file > env os > default
func Load(envFile string) (*Config, error) {
	if envFile != "" {
		if err := godotenv.Overload(envFile); err != nil {
			return nil, err
		}
	}

	return &Config{
		HTTP: struct{ Port int }{
			Port: getEnvInt("HTTP_PORT", 8080),
		},
		GRPC: struct{ Port int }{
			Port: getEnvInt("GRPC_PORT", 9090),
		},
		DB: struct{ Url string }{
			Url: getEnvString("DB_URL", "postgres://localhost:5432/link"),
		},
	}, nil
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
