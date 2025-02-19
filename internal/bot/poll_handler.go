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

// dayRegex для парсинга "(DD.MM)" из строки "Понедельник (17.02)"
var dayRegex = utils.DayRegex // или regexp.MustCompile(`\((\d{2})\.(\d{2})\)`)

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
		handleWeeklyPoll(bot, pollAnswer, &p, userID, userName, weekOpts)
	} else {
		handleDayPoll(bot, pollAnswer, &p, userID, userName)
	}
}

func handleWeeklyPoll(bot *tgbotapi.BotAPI, pollAnswer *tgbotapi.PollAnswer, p *db.Poll, userID int64, userName string, options []string) {
	for _, optID := range pollAnswer.OptionIDs {
		if optID < 0 || optID >= len(options) {
			continue
		}
		choice := options[optID]
		v := db.Vote{
			PollID:   p.PollID,
			UserID:   userID,
			UserName: userName,
			ChatID:   p.ChatID,
			VoteDate: choice,
		}
		db.DB.Create(&v)
		log.Printf("✅ %s => weekly: %s", userName, choice)

		// Считаем голоса за конкретный вариант
		var c int64
		db.DB.Model(&db.Vote{}).
			Where("poll_id = ? AND vote_date = ?", p.PollID, choice).
			Count(&c)
		if c == int64(constants.NumbersOfPlayers) {
			// Берём только первые N голосов, сортированные по created_at
			var earliestVotes []db.Vote
			db.DB.Where("poll_id = ? AND vote_date = ?", p.PollID, choice).
				Order("created_at ASC").
				Limit(constants.NumbersOfPlayers).
				Find(&earliestVotes)

			usersSet := make(map[string]bool)
			var usersList string
			for _, vv := range earliestVotes {
				if !usersSet[vv.UserName] {
					usersSet[vv.UserName] = true
					usersList += "@" + vv.UserName + "\n"
				}
			}

			// Вычисляем reminderTime через calcReminderTime
			evt, err := parseWeeklyDate(choice)
			if err != nil {
				log.Printf("Ошибка parseWeeklyDate: %v", err)
				return
			}
			reminderTime := calcReminderTime(evt)

			alertMsg := fmt.Sprintf("🔔 За день '%s' набралось %d голосов!\nУчастники:\n%s",
				choice, c, usersList)
			finalMsg := fmt.Sprintf("Напоминание придёт %s.",
				reminderTime.Format("02.01.2006 15:04:05"))
			fullMsg := alertMsg + "\n" + finalMsg

			sendNormalMessage(bot, p.ChatID, fullMsg)

			// Если используете модель Reminder
			rem := db.Reminder{
				PollID:       p.PollID,
				OptionDate:   choice,
				ReminderTime: reminderTime,
				Reminded:     false,
			}
			db.DB.Create(&rem)
			log.Printf("🕒 Создан Reminder для %s (PollID=%s) на %v", choice, p.PollID, reminderTime)
		}
	}
}

func handleDayPoll(bot *tgbotapi.BotAPI, pollAnswer *tgbotapi.PollAnswer, p *db.Poll, userID int64, userName string) {
	for _, optID := range pollAnswer.OptionIDs {
		// "option_0" => да
		if optID == 0 {
			v := db.Vote{
				PollID:   p.PollID,
				UserID:   userID,
				UserName: userName,
				ChatID:   p.ChatID,
				VoteDate: p.PollDay, // хранит "Понедельник (17.02)"
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

func countDayYes(pollID, pollDay string) int {
	var c int64
	db.DB.Model(&db.Vote{}).Where("poll_id = ? AND vote_date = ?", pollID, pollDay).Count(&c)
	return int(c)
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
	mInt, _ := strconv.Atoi(mm)
	t := time.Date(now.Year(), time.Month(mInt), d, 0, 0, 0, 0, loc)
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
