package main

import (
	"log"

	"go-telegram-cs-admin/config"
	"go-telegram-cs-admin/internal/bot"
	"go-telegram-cs-admin/internal/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	db.InitDB()
	cfg := config.LoadConfig()
	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatal("Ошибка запуска бота:", err)
	}
	log.Printf("✅ Бот %s запущен", botAPI.Self.UserName)

	// Запускаем горутину напоминаний
	bot.StartReminderRoutine(botAPI)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "poll_answer"}
	updates := botAPI.GetUpdatesChan(u)
	for update := range updates {
		if update.PollAnswer != nil {
			bot.HandlePollAnswer(botAPI, update.PollAnswer)
		} else if update.Message != nil {
			bot.HandleMessage(botAPI, update.Message)
		}
	}
}
