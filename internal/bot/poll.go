package bot

import (
	"log"

	"go-telegram-cs-admin/internal/db"
	"go-telegram-cs-admin/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendPoll создаёт встроенный опрос Telegram
func SendPoll(bot *tgbotapi.BotAPI, chatID int64) {
	pollOptions := utils.GeneratePollOptions() // ["Понедельник (10.02)", ...]
	poll := tgbotapi.NewPoll(chatID, "📅 Выберите удобные дни для встречи:", pollOptions...)
	poll.AllowsMultipleAnswers = true
	poll.IsAnonymous = false

	sentPoll, err := bot.Send(poll)
	if err != nil {
		log.Println("Ошибка отправки опроса:", err)
		return
	}
	if sentPoll.Poll == nil {
		log.Println("⚠️ Не удалось получить PollID от Telegram")
		return
	}

	newPoll := db.Poll{
		PollID:    sentPoll.Poll.ID,
		ChatID:    chatID,
		MessageID: sentPoll.MessageID,
		IsClosed:  false,
	}
	if err := db.DB.Create(&newPoll).Error; err != nil {
		log.Println("❌ Ошибка сохранения опроса в БД:", err)
		return
	}

	log.Printf("✅ Опрос создан в чате %d. PollID=%s, MessageID=%d",
		chatID, sentPoll.Poll.ID, sentPoll.MessageID)
}
