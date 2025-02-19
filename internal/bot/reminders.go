package bot

import (
	"fmt"
	"log"
	"time"

	"go-telegram-cs-admin/internal/constants"
	"go-telegram-cs-admin/internal/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartReminderRoutine(bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			checkReminders(bot)
		}
	}()
}

// checkReminders — каждые 30 сек ищет записи в таблице reminders, где ReminderTime<=now и Reminded=false.
func checkReminders(bot *tgbotapi.BotAPI) {
	now := time.Now()
	var items []db.Reminder
	err := db.DB.Where("reminded = ? AND reminder_time <= ?", false, now).Find(&items).Error
	if err != nil {
		log.Println("Ошибка поиска reminders:", err)
		return
	}

	for _, r := range items {
		// Извлекаем список голосов, где VoteDate=r.OptionDate
		// чтобы собрать участников
		var earliestVotes []db.Vote
		db.DB.Where("poll_id = ? AND vote_date = ?", r.PollID, r.OptionDate).
			Order("created_at ASC").
			Limit(constants.NumbersOfPlayers).
			Find(&earliestVotes)

		// Получаем ChatID из Poll
		chatID := getChatID(r.PollID)
		if chatID == 0 {
			log.Printf("Не удалось определить ChatID для PollID=%s", r.PollID)
			continue
		}

		// Формируем список уникальных участников
		usersSet := make(map[string]bool)
		var usersList string
		for _, v := range earliestVotes {
			if !usersSet[v.UserName] {
				usersSet[v.UserName] = true
				usersList += "@" + v.UserName + "\n"
			}
		}

		// Финальное сообщение
		fullMsg := fmt.Sprintf("⏰ Напоминаю! Встреча состоится %s.\nУчастники:\n%s\n🚨 Если кто-то передумал —предупредите об этом остальных игроков\n\n%s",
			r.OptionDate,
			usersList,
			constants.MsgMeetingPlace,
		)
		sendNormalMessage(bot, chatID, fullMsg)

		// Помечаем напоминание как отправленное
		r.Reminded = true
		db.DB.Save(&r)
		log.Printf("✅ Напоминание отправлено (PollID=%s, Option=%s)", r.PollID, r.OptionDate)
	}
}

// getChatID — вытаскиваем ChatID из Poll
func getChatID(pollID string) int64 {
	var p db.Poll
	if err := db.DB.Where("poll_id = ?", pollID).First(&p).Error; err != nil {
		log.Println("Ошибка получения Poll:", err)
		return 0
	}
	return p.ChatID
}
