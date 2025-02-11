package bot

import (
	"fmt"
	"log"

	"go-telegram-cs-admin/internal/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	// Если ваша структура пакета БД находится по пути "go-telegram-cs-admin/db",
	// отредактируйте импорт соответственно.
)

// ClosePoll закрывает опрос вручную (команда /close_pool).
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
