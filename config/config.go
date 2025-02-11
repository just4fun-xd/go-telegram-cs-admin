// config/config.go
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken  string
	DebugReminders bool
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Не удалось загрузить .env файл, используем переменные среды")
	}

	debugReminders := false
	if val, ok := os.LookupEnv("DEBUG_REMINDERS"); ok {
		// если "1" или "true" — установим debugReminders = true
		debugReminders, _ = strconv.ParseBool(val)
	}

	return &Config{
		TelegramToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		DebugReminders: debugReminders,
	}
}
