package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"go-telegram-cs-admin/internal/db"
	"go-telegram-cs-admin/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendPoll(bot *tgbotapi.BotAPI, chatID int64) {
	options := utils.GeneratePollOptions()
	question := "📅 Выберите удобные дни"

	pollCfg := tgbotapi.NewPoll(chatID, question, options...)
	pollCfg.AllowsMultipleAnswers = true
	pollCfg.IsAnonymous = false

	// 1) Отправляем опрос
	sent, err := bot.Send(pollCfg)
	if err != nil {
		log.Println("Ошибка отправки еженедельного опроса:", err)
		return
	}
	if sent.Poll == nil {
		log.Println("⚠️ Не удалось получить Poll от Telegram (нет sent.Poll)")
		return
	}

	// Сохраняем Poll в базе
	newPoll := db.Poll{
		PollID:       sent.Poll.ID,
		ChatID:       chatID,
		MessageID:    sent.MessageID,
		IsClosed:     false,
		OptionsCount: len(options),
	}
	if e := db.DB.Create(&newPoll).Error; e != nil {
		log.Println("❌ Ошибка сохранения еженедельного опроса:", e)
		return
	}
	log.Printf("✅ Еженедельный опрос создан. PollID=%s, MessageID=%d", sent.Poll.ID, sent.MessageID)

	// 2) Создаём инлайн-кнопки
	inlineKeys := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Закрыть", fmt.Sprintf("close_%d", sent.MessageID)),
			tgbotapi.NewInlineKeyboardButtonData("Удалить", fmt.Sprintf("delete_%d", sent.MessageID)),
		),
	)

	// 3) "Редактируем" сообщение с опросом, чтобы к нему прикрепить инлайн-кнопки
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, sent.MessageID, inlineKeys)
	if _, err2 := bot.Send(editMarkup); err2 != nil {
		log.Printf("Ошибка добавления инлайн-кнопок к опросу: %v", err2)
	}
}

func SendDayPoll(bot *tgbotapi.BotAPI, chatID int64, dayStr string) {
	t, err := parseDDMM(dayStr)
	if err != nil {
		log.Println("Ошибка parseDDMM:", err)
		return
	}
	weekday := getRussianWeekday(t.Weekday())
	pollDay := fmt.Sprintf("%s (%s)", weekday, t.Format("02.01"))

	pollOptions := []string{"да", "нет", "мне только посмотреть"}
	question := fmt.Sprintf("Соберёмся %s?", dayStr)

	cfg := tgbotapi.NewPoll(chatID, question, pollOptions...)
	cfg.AllowsMultipleAnswers = false
	cfg.IsAnonymous = false

	// 1) Отправляем опрос
	sentPoll, err := bot.Send(cfg)
	if err != nil || sentPoll.Poll == nil {
		log.Println("Ошибка отправки дневного опроса:", err)
		return
	}

	// Сохраняем Poll в базе
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

	// 2) Создаём инлайн-кнопки
	inlineKeys := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Закрыть", fmt.Sprintf("close_%d", sentPoll.MessageID)),
			tgbotapi.NewInlineKeyboardButtonData("Удалить", fmt.Sprintf("delete_%d", sentPoll.MessageID)),
		),
	)

	// 3) Редактируем то же сообщение, прикрепляя инлайн-кнопки
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, sentPoll.MessageID, inlineKeys)
	if _, err2 := bot.Send(editMarkup); err2 != nil {
		log.Printf("Ошибка добавления инлайн-кнопок к дневному опросу: %v", err2)
	}
}

// parseDDMM ...
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

// getRussianWeekday ...
func getRussianWeekday(w time.Weekday) string {
	switch w {
	case time.Monday:
		return "Понедельник"
	case time.Tuesday:
		return "Вторник"
	case time.Wednesday:
		return "Среда"
	case time.Thursday:
		return "Четверг"
	case time.Friday:
		return "Пятница"
	case time.Saturday:
		return "Суббота"
	case time.Sunday:
		return "Воскресенье"
	}
	return ""
}
