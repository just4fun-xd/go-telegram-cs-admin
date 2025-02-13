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

func checkReminders(bot *tgbotapi.BotAPI) {
	now := time.Now()
	var polls []db.Poll
	err := db.DB.Where("reminder_date IS NOT NULL AND reminded = ? AND reminder_date <= ?", false, now).
		Find(&polls).Error
	if err != nil {
		log.Println("Ошибка поиска опросов:", err)
		return
	}

	for _, p := range polls {
		wday := getRussianWeekday(p.EventDate.Weekday())
		dateFmt := fmt.Sprintf("%s (%s)", wday, p.EventDate.Format("02.01"))

		// Собираем участников — разное поведение для day/weekly
		var votes []db.Vote
		if p.OptionsCount == 3 {
			// Дневной опрос => только "да" (храним VoteDate = p.PollDay)
			db.DB.Where("poll_id = ? AND vote_date = ?", p.PollID, p.PollDay).Find(&votes)
		} else {
			// Еженедельный опрос => все
			db.DB.Where("poll_id = ?", p.PollID).Find(&votes)
		}

		visited := make(map[string]bool)
		var participants string
		for _, v := range votes {
			if !visited[v.UserName] {
				visited[v.UserName] = true
				participants += "@" + v.UserName + "\n"
			}
		}

		msgText := fmt.Sprintf(
			"⏰ Напоминаю! Встреча состоится %s.\nУчастники:\n%s\n%s\nЕсли кто-то передумал — снимите галочку или ответьте 'нет'.",
			dateFmt,
			participants,
			constants.MsgMeetingPlace, // Адрес клуба
		)

		if _, e := bot.Send(tgbotapi.NewMessage(p.ChatID, msgText)); e != nil {
			log.Printf("Ошибка отправки напоминания PollID=%s: %v", p.PollID, e)
			continue
		}

		p.Reminded = true
		db.DB.Save(&p)
		log.Printf("✅ Напоминание отправлено для PollID=%s", p.PollID)
	}
}
