package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"go-telegram-cs-admin/internal/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// StartReminderRoutine запускает горутину, которая каждые 30 секунд проверяет напоминания.
func StartReminderRoutine(bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			checkReminders(bot)
		}
	}()
}

func checkReminders(bot *tgbotapi.BotAPI) {
	now := time.Now()
	var polls []db.Poll
	err := db.DB.Where("reminder_date IS NOT NULL AND reminded = ? AND reminder_date <= ?", false, now).
		Find(&polls).Error
	if err != nil {
		log.Println("Ошибка поиска опросов для напоминаний:", err)
		return
	}
	for _, p := range polls {
		participants := getAllVotersForPoll(p.PollID)
		reminderMsg := fmt.Sprintf(
			"⏰ Напоминаю, что через 2 дня (%s) состоится встреча!\nУчастники:\n%s\nЕсли кто-то передумал — снимите галочку или ответьте 'нет'.",
			p.EventDate.Format("02.01.2006"),
			participants,
		)
		_, err := bot.Send(tgbotapi.NewMessage(p.ChatID, reminderMsg))
		if err != nil {
			log.Printf("Ошибка отправки напоминания PollID=%s: %v", p.PollID, err)
			continue
		}
		p.Reminded = true
		if err := db.DB.Save(&p).Error; err != nil {
			log.Printf("Ошибка обновления Reminded для PollID=%s: %v", p.PollID, err)
		} else {
			log.Printf("✅ Отправлено напоминание для PollID=%s", p.PollID)
		}
	}
}

func getAllVotersForPoll(pollID string) string {
	var votes []db.Vote
	db.DB.Where("poll_id = ?", pollID).Find(&votes)
	unique := make(map[string]bool)
	for _, v := range votes {
		unique[v.UserName] = true
	}
	var sb strings.Builder
	for user := range unique {
		sb.WriteString("@" + user + "\n")
	}
	return sb.String()
}
