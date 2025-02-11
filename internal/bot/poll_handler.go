package bot

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-telegram-cs-admin/config"
	"go-telegram-cs-admin/internal/constants"
	"go-telegram-cs-admin/internal/db"
	"go-telegram-cs-admin/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// dayRegex используется для парсинга строки вида "Вторник (11.02)"
var dayRegex = regexp.MustCompile(`\((\d{2}\.\d{2})\)`)

// HandlePollAnswer вызывается при получении poll_answer
func HandlePollAnswer(bot *tgbotapi.BotAPI, pollAnswer *tgbotapi.PollAnswer) {
	pollID := pollAnswer.PollID
	userID := pollAnswer.User.ID
	userName := pollAnswer.User.UserName
	if userName == "" {
		userName = strings.TrimSpace(pollAnswer.User.FirstName + " " + pollAnswer.User.LastName)
	}

	log.Printf("📩 Получен голос от: %s (ID=%d) за опрос %s", userName, userID, pollID)

	// Ищем опрос в БД
	var p db.Poll
	if err := db.DB.Where("poll_id = ?", pollID).First(&p).Error; err != nil {
		log.Printf("⚠️ Опрос %s не найден в БД", pollID)
		return
	}

	// Если опрос закрыт, по желанию можно игнорировать ответы
	if p.IsClosed {
		log.Printf("⚠️ Опрос %s уже закрыт, пропускаем ответы", pollID)
		return
	}

	// Удаляем старые голоса этого пользователя
	db.DB.Where("poll_id = ? AND user_id = ?", pollID, userID).Delete(&db.Vote{})

	// Сохраняем новые голоса
	options := utils.GeneratePollOptions() // Например, ["Понедельник (10.02)", "Вторник (11.02)", ...]
	for _, optionID := range pollAnswer.OptionIDs {
		if optionID < 0 || optionID >= len(options) {
			continue
		}
		dateChoice := options[optionID]

		vote := db.Vote{
			PollID:   pollID,
			UserID:   userID,
			UserName: userName,
			ChatID:   p.ChatID,
			VoteDate: dateChoice,
		}
		if err := db.DB.Create(&vote).Error; err != nil {
			log.Printf("❌ Ошибка сохранения голоса: %v", err)
		} else {
			log.Printf("✅ %s (ID=%d) проголосовал за %s", userName, userID, dateChoice)

			// Если за этот день набралось 10 голосов, отправляем уведомление
			count := db.CountVotesForDate(pollID, dateChoice)
			if count == constants.NumbersOfPlayers {
				sendThresholdAlert(bot, p.ChatID, dateChoice, pollID)
				// При желании можно автоматически зафиксировать дату встречи.
				// Здесь мы НЕ закрываем опрос, а лишь фиксируем дату встречи и рассчитываем ReminderDate.
				autoFinalizeDay(bot, &p, dateChoice)
			}
		}
	}
}

// autoFinalizeDay устанавливает EventDate и ReminderDate для опроса (без закрытия опроса)
func autoFinalizeDay(bot *tgbotapi.BotAPI, p *db.Poll, dateChoice string) {
	log.Printf("🔒 Автофиксация даты для PollID=%s: %s", p.PollID, dateChoice)

	// Парсим дату из строки вида "Вторник (11.02)"
	eventTime, err := parseDayChoice(dateChoice)
	if err != nil {
		log.Printf("Ошибка парсинга '%s': %v", dateChoice, err)
		return
	}

	// Вычисляем ReminderDate с учетом режима отладки
	reminderTime := calcReminderTime(eventTime)

	// Сохраняем значения в опросе
	p.EventDate = eventTime
	p.ReminderDate = &reminderTime
	p.Reminded = false
	// Здесь опрос не закрывается автоматически
	if err := db.DB.Save(&p).Error; err != nil {
		log.Printf("Ошибка сохранения Poll: %v", err)
		return
	}

	msgText := fmt.Sprintf(
		"Дата встречи зафиксирована: %s.\nНапоминание придёт %s.",
		dateChoice,
		reminderTime.Format("02.01.2006 15:04:05"),
	)
	bot.Send(tgbotapi.NewMessage(p.ChatID, msgText))
}

// parseDayChoice извлекает дату из строки вида "Вторник (11.02)" и возвращает time.Time
func parseDayChoice(dayChoice string) (time.Time, error) {
	match := dayRegex.FindStringSubmatch(dayChoice)
	if len(match) < 2 {
		return time.Time{}, fmt.Errorf("не найдена дата в '%s'", dayChoice)
	}
	dateStr := match[1] // Например, "11.02"
	parts := strings.Split(dateStr, ".")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("некорректный формат '%s'", dateStr)
	}
	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	loc, err := time.LoadLocation("Asia/Novosibirsk")
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	year := now.Year()
	event := time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
	if event.Before(now) {
		// Если дата уже прошла, сдвигаем на следующий год
		event = event.AddDate(1, 0, 0)
	}
	return event, nil
}

// calcReminderTime вычисляет время напоминания.
// Если включен режим отладки (DebugReminders), возвращает время через 30 секунд.
// Иначе, возвращает EventTime минус 2 дня, установив время на 14:00 по Новосибирску.
func calcReminderTime(eventTime time.Time) time.Time {
	cfg := config.LoadConfig()
	if cfg.DebugReminders {
		log.Println("[DEBUG] Режим отладки включен: напоминание через 30 секунд.")
		return time.Now().Add(30 * time.Second)
	}

	loc, err := time.LoadLocation("Asia/Novosibirsk")
	if err != nil {
		log.Println("Ошибка загрузки локали Asia/Novosibirsk:", err)
		loc = time.UTC
	}

	reminderTime := eventTime.AddDate(0, 0, -2)
	reminderTime = time.Date(
		reminderTime.Year(),
		reminderTime.Month(),
		reminderTime.Day(),
		14, 0, 0, 0,
		loc,
	)
	return reminderTime
}

// sendThresholdAlert отправляет уведомление, что за конкретный день набралось 10 голосов.
func sendThresholdAlert(bot *tgbotapi.BotAPI, chatID int64, dateChoice, pollID string) {
	voters := db.GetVotersForDate(pollID, dateChoice)
	text := fmt.Sprintf("🔔 За день '%s' набрано 10 голосов!\nУчастники:\n%s", dateChoice, voters)
	bot.Send(tgbotapi.NewMessage(chatID, text))
}
