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

var dayRegex = regexp.MustCompile(`\((\d{2})\.(\d{2})\)`)

const thresholdVotes = 10

func HandlePollAnswer(bot *tgbotapi.BotAPI, pollAnswer *tgbotapi.PollAnswer) {
	pollID := pollAnswer.PollID
	userID := pollAnswer.User.ID
	userName := pollAnswer.User.UserName
	if userName == "" {
		userName = strings.TrimSpace(pollAnswer.User.FirstName + " " + pollAnswer.User.LastName)
	}
	log.Printf("📩 vote from %s (ID=%d) for poll %s", userName, userID, pollID)

	var p db.Poll
	if err := db.DB.Where("poll_id = ?", pollID).First(&p).Error; err != nil {
		log.Printf("⚠️ Poll %s not found: %v", pollID, err)
		return
	}
	if p.IsClosed {
		log.Printf("⚠️ Poll %s is closed. ignoring", pollID)
		return
	}

	// Удаляем предыдущие голоса этого user
	db.DB.Where("poll_id = ? AND user_id = ?", pollID, userID).Delete(&db.Vote{})

	weekOpts := utils.GeneratePollOptions()
	if p.OptionsCount == len(weekOpts) {
		// Еженедельный опрос
		handleWeeklyPoll(bot, pollAnswer, &p, userID, userName, weekOpts)
	} else {
		// Дневной опрос (3 варианта)
		handleDayPoll(bot, pollAnswer, &p, userID, userName)
	}
}

// handleWeeklyPoll — если user выбрал "Понедельник (17.02)" etc.
func handleWeeklyPoll(bot *tgbotapi.BotAPI, pollAnswer *tgbotapi.PollAnswer, p *db.Poll, userID int64, userName string, options []string) {
	for _, optID := range pollAnswer.OptionIDs {
		if optID < 0 || optID >= len(options) {
			continue
		}
		choice := options[optID] // "Понедельник (17.02)"
		v := db.Vote{
			PollID:   p.PollID,
			UserID:   userID,
			UserName: userName,
			ChatID:   p.ChatID,
			VoteDate: choice,
		}
		db.DB.Create(&v)
		log.Printf("✅ %s => weekly: %s", userName, choice)

		c := db.CountVotesForDate(p.PollID, choice)
		if c == constants.NumbersOfPlayers {
			// Уведомление
			voters := db.GetVotersForDate(p.PollID, choice)
			alert := fmt.Sprintf("🔔 За день '%s' набралось 10 голосов!\nУчастники:\n%s", choice, voters)
			sendNormalMessage(bot, p.ChatID, alert)

			finalizeWeeklyPoll(bot, p, choice)
		}
	}
}

// handleDayPoll — 3 варианта ("да","нет","посмотреть"). Сохраняем в БД только "да".
func handleDayPoll(bot *tgbotapi.BotAPI, pollAnswer *tgbotapi.PollAnswer, p *db.Poll, userID int64, userName string) {
	for _, optID := range pollAnswer.OptionIDs {
		// "option_0" => да
		if optID == 0 {
			// Записываем "да": VoteDate = p.PollDay (напр. "Понедельник (17.02)")
			v := db.Vote{
				PollID:   p.PollID,
				UserID:   userID,
				UserName: userName,
				ChatID:   p.ChatID,
				VoteDate: p.PollDay,
			}
			db.DB.Create(&v)
			log.Printf("✅ %s => day poll: da => %s", userName, p.PollDay)

			countYes := countDayYes(p.PollID, p.PollDay)
			if countYes == constants.NumbersOfPlayers {
				closeDayPoll(bot, p)
			}
		}
	}
}

// countDayYes — ищем все записи, где VoteDate = p.PollDay
func countDayYes(pollID, pollDay string) int {
	var c int64
	db.DB.Model(&db.Vote{}).Where("poll_id = ? AND vote_date = ?", pollID, pollDay).Count(&c)
	return int(c)
}

// finalizeWeeklyPoll — парсим "(17.02)" => time.Time => ReminderDate => msg
func finalizeWeeklyPoll(bot *tgbotapi.BotAPI, p *db.Poll, dateChoice string) {
	t, err := parseWeeklyDate(dateChoice)
	if err != nil {
		log.Printf("parseWeeklyDate error: %v", err)
		return
	}
	r := calcReminderTime(t)
	p.EventDate = t
	p.ReminderDate = &r
	p.Reminded = false
	db.DB.Save(&p)

	wd := getRussianWeekday(t.Weekday())
	dateF := fmt.Sprintf("%s (%s)", wd, t.Format("02.01"))
	msg := fmt.Sprintf("Дата встречи зафиксирована: %s.\nНапоминание придёт %s.",
		dateF,
		r.Format("02.01.2006 15:04:05"),
	)
	sendNormalMessage(bot, p.ChatID, msg)
}

// closeDayPoll — при 10 голосах "да"
func closeDayPoll(bot *tgbotapi.BotAPI, p *db.Poll) {
	stop := tgbotapi.NewStopPoll(p.ChatID, p.MessageID)
	bot.StopPoll(stop)

	p.IsClosed = true
	db.DB.Save(&p)

	// Собираем всех "да"
	var yesVotes []db.Vote
	db.DB.Where("poll_id = ? AND vote_date = ?", p.PollID, p.PollDay).Find(&yesVotes)

	var names string
	for _, v := range yesVotes {
		names += "@" + v.UserName + "\n"
	}
	alert := fmt.Sprintf("🔔 За день (%s) набралось 10 голосов 'да'!\nУчастники:\n%s",
		p.PollDay, names)
	sendNormalMessage(bot, p.ChatID, alert)

	// Формируем ReminderDate
	r := calcReminderTime(p.EventDate)
	p.ReminderDate = &r
	p.Reminded = false
	db.DB.Save(&p)

	wd := getRussianWeekday(p.EventDate.Weekday())
	dateF := fmt.Sprintf("%s (%s)", wd, p.EventDate.Format("02.01"))
	finalMsg := fmt.Sprintf("Дата встречи зафиксирована: %s.\nНапоминание придёт %s.",
		dateF,
		r.Format("02.01.2006 15:04:05"),
	)
	sendNormalMessage(bot, p.ChatID, finalMsg)
}

func parseWeeklyDate(str string) (time.Time, error) {
	reg := regexp.MustCompile(`\((\d{2})\.(\d{2})\)`)
	m := reg.FindStringSubmatch(str)
	if len(m) < 3 {
		return time.Time{}, fmt.Errorf("не найдена (DD.MM) в %s", str)
	}
	dd := m[1]
	mm := m[2]
	loc, _ := time.LoadLocation("Asia/Novosibirsk")
	now := time.Now().In(loc)
	d, _ := strconv.Atoi(dd)
	mon, _ := strconv.Atoi(mm)
	t := time.Date(now.Year(), time.Month(mon), d, 0, 0, 0, 0, loc)
	if t.Before(now) {
		t = t.AddDate(1, 0, 0)
	}
	return t, nil
}

// calcReminderTime — debug => +30сек, иначе -2дня, 15:00
func calcReminderTime(evt time.Time) time.Time {
	cfg := config.LoadConfig()
	if cfg.DebugReminders {
		log.Println("[DEBUG] reminder in 30 seconds")
		return time.Now().Add(30 * time.Second)
	}
	loc, _ := time.LoadLocation("Asia/Novosibirsk")
	r := evt.AddDate(0, 0, -2)
	r = time.Date(r.Year(), r.Month(), r.Day(), 15, 0, 0, 0, loc)
	return r
}

func SendPoll(bot *tgbotapi.BotAPI, chatID int64) {
	options := utils.GeneratePollOptions()
	question := "📅 Выберите удобные дни"

	// Создаём Telegram-полл
	pollCfg := tgbotapi.NewPoll(chatID, question, options...)
	pollCfg.AllowsMultipleAnswers = true // если нужен выбор нескольких дней
	pollCfg.IsAnonymous = false          // хотим видеть, кто проголосовал

	sent, err := bot.Send(pollCfg)
	if err != nil {
		log.Println("Ошибка отправки еженедельного опроса:", err)
		return
	}
	if sent.Poll == nil {
		log.Println("⚠️ Не удалось получить Poll от Telegram (нет sent.Poll)")
		return
	}

	// Сохраняем запись в БД
	newPoll := db.Poll{
		PollID:       sent.Poll.ID,
		ChatID:       chatID,
		MessageID:    sent.MessageID,
		IsClosed:     false,
		OptionsCount: len(options), // например, 5
	}

	if e := db.DB.Create(&newPoll).Error; e != nil {
		log.Println("❌ Ошибка сохранения еженедельного опроса:", e)
		return
	}

	log.Printf("✅ Еженедельный опрос создан. PollID=%s, MessageID=%d", sent.Poll.ID, sent.MessageID)
}

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
