package main

import (
	"log"

	"go-telegram-cs-admin/config"
	"go-telegram-cs-admin/internal/bot"
	"go-telegram-cs-admin/internal/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Инициализация базы
	db.InitDB()

	// Загрузка конфигурации
	cfg := config.LoadConfig()

	// Создаём экземпляр бота
	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatal("Ошибка запуска бота:", err)
	}
	log.Printf("✅ Бот %s запущен", botAPI.Self.UserName)

	// Запускаем горутину напоминаний
	bot.StartReminderRoutine(botAPI)

	// Настраиваем получение обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "poll_answer", "callback_query"}
	// Добавили "callback_query", чтобы бот принимал события от инлайн-кнопок

	updates := botAPI.GetUpdatesChan(u)

	// Обрабатываем обновления
	for update := range updates {
		switch {
		case update.CallbackQuery != nil:
			// Нажатие на инлайн-кнопку
			bot.HandleCallbackQuery(botAPI, update.CallbackQuery)

		case update.PollAnswer != nil:
			// Ответ на опрос (poll_answer)
			bot.HandlePollAnswer(botAPI, update.PollAnswer)

		case update.Message != nil:
			// Обычное сообщение (текст, команда и т.д.)
			bot.HandleMessage(botAPI, update.Message)
		}
	}
}
