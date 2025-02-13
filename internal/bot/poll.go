package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"go-telegram-cs-admin/internal/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendDayPoll(bot *tgbotapi.BotAPI, chatID int64, dayStr string) {
	t, err := parseDDMM(dayStr)
	if err != nil {
		log.Println("Ошибка parseDDMM:", err)
		return
	}
	weekday := getRussianWeekday(t.Weekday())
	pollDay := fmt.Sprintf("%s (%s)", weekday, t.Format("02.01"))

	// Создаём Telegram-опрос
	pollOptions := []string{"да", "нет", "мне только посмотреть"}
	question := fmt.Sprintf("Соберёмся %s?", dayStr) //

	cfg := tgbotapi.NewPoll(chatID, question, pollOptions...)
	cfg.AllowsMultipleAnswers = false
	cfg.IsAnonymous = false

	sentPoll, err := bot.Send(cfg)
	if err != nil || sentPoll.Poll == nil {
		log.Println("Ошибка отправки дневного опроса:", err)
		return
	}

	// Сохраняем в БД
	newPoll := db.Poll{
		PollID:       sentPoll.Poll.ID,
		ChatID:       chatID,
		MessageID:    sentPoll.MessageID,
		IsClosed:     false,
		OptionsCount: 3,
		EventDate:    t,
		PollDay:      pollDay,
	}
	if e := db.DB.Create(&newPoll).Error; e != nil {
		log.Println("❌ Ошибка сохранения дневного опроса:", e)
		return
	}

	log.Printf("✅ Дневной опрос %s создан (PollID=%s).", pollDay, sentPoll.Poll.ID)
}

func parseDDMM(ddmm string) (time.Time, error) {
	parts := strings.Split(ddmm, ".")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("формат не DD.MM")
	}
	dd, err1 := strconv.Atoi(parts[0])
	mm, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return time.Time{}, fmt.Errorf("ошибка конвертации")
	}
	loc, _ := time.LoadLocation("Asia/Novosibirsk")
	now := time.Now().In(loc)
	year := now.Year()
	t := time.Date(year, time.Month(mm), dd, 0, 0, 0, 0, loc)
	if t.Before(now) {
		t = t.AddDate(1, 0, 0)
	}
	return t, nil
}
