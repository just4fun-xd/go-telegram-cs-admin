package bot

import (
	"fmt"
	"log"

	"go-telegram-cs-admin/internal/constants"
	"go-telegram-cs-admin/internal/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ClosePoll(bot *tgbotapi.BotAPI, pollID string) error {
	var p db.Poll
	if err := db.DB.Where("poll_id = ?", pollID).First(&p).Error; err != nil {
		return fmt.Errorf("опрос %s не найден: %w", pollID, err)
	}
	if p.IsClosed {
		return fmt.Errorf("опрос уже закрыт")
	}
	stopConfig := tgbotapi.NewStopPoll(p.ChatID, p.MessageID)
	if _, err := bot.StopPoll(stopConfig); err != nil {
		return fmt.Errorf("ошибка закрытия опроса: %w", err)
	}
	p.IsClosed = true
	if err := db.DB.Save(&p).Error; err != nil {
		return fmt.Errorf("ошибка сохранения статуса опроса: %w", err)
	}
	log.Printf("🛑 Опрос %s закрыт вручную", pollID)
	return nil
}

func closeDayPoll(bot *tgbotapi.BotAPI, p *db.Poll) {
	stopConfig := tgbotapi.NewStopPoll(p.ChatID, p.MessageID)
	bot.StopPoll(stopConfig)

	p.IsClosed = true
	db.DB.Save(&p)

	// Собираем всех "да" (первые N, если хотите)
	var earliestVotes []db.Vote
	db.DB.Where("poll_id = ? AND vote_date = ?", p.PollID, p.PollDay).
		Order("created_at ASC").
		Limit(constants.NumbersOfPlayers).
		Find(&earliestVotes)

	var names string
	for _, vv := range earliestVotes {
		names += "@" + vv.UserName + "\n"
	}
	alert := fmt.Sprintf("🔔 За день (%s) набралось %d голосов 'да'!\nУчастники:\n%s",
		p.PollDay, len(earliestVotes), names)
	// sendNormalMessage(bot, p.ChatID, alert)

	// Рассчитываем ReminderDate
	r := calcReminderTime(p.EventDate)
	p.ReminderDate = &r
	p.Reminded = false
	db.DB.Save(p)

	wd := getRussianWeekday(p.EventDate.Weekday())
	dateF := fmt.Sprintf("%s (%s)", wd, p.EventDate.Format("02.01"))
	finalMsg := fmt.Sprintf("Дата встречи зафиксирована: %s.\nНапоминание придёт %s.",
		dateF,
		r.Format("02.01.2006 15:04:05"),
	)
	fullMsg := alert + "/n/n" + finalMsg
	sendNormalMessage(bot, p.ChatID, fullMsg)
}
